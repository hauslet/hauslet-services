package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/leads/domain"
	"hauslet/internal/modules/leads/repository"
	"hauslet/internal/platform/events"
	"hauslet/internal/platform/events/payoads"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateLead creates a new lead with validation, spam detection, and rate limiting
// HYBRID endpoint: Works with or without authentication
// - Authenticated users: Auto-fills from profile, marked as verified, lower spam score
// - Anonymous users: Manual entry, spam checks applied, rate limited
func (s *ServiceImpl) CreateLead(ctx context.Context, input CreateLeadInput) (*domain.Lead, error) {
	isVerified := false

	// HYBRID: If UserID provided, auto-fill from profile
	if input.UserID != nil {
		s.log.Info("authenticated user creating lead", "user_id", input.UserID)

		// Fetch profile data via hooks (if available)
		if s.profileHooks != nil {
			profile, err := s.profileHooks.GetUserProfile(ctx, *input.UserID)
			if err != nil {
				s.log.Error("failed to fetch user profile", "error", err, "user_id", input.UserID)
				// Don't fail - fall back to manual entry
			} else if profile != nil {
				// Auto-fill from verified profile
				// Override input fields
				input.Name = profile.FullName

				input.Email = profile.Email

				input.PhoneNumber = profile.Phone

				isVerified = true // Mark as verified since from authenticated user
				s.log.Info("auto-filled lead from profile", "user_id", input.UserID, "name", input.Name, "is_verified", isVerified)
			}
		}
	}

	// 1. Validate input
	if err := s.validator.ValidateCreateLeadInput(input); err != nil {
		s.log.Warn("lead validation failed", "error", err, "email", input.Email)
		return nil, err
	}

	// Normalize email
	normalizedEmail, err := s.validator.ValidateEmail(input.Email)
	if err != nil {
		return nil, err
	}
	input.Email = normalizedEmail

	// 2. Verify listing exists
	exists, err := s.propertyHooks.ListingExists(ctx, input.ListingID)
	if err != nil {
		s.log.Error("failed to verify listing", "error", err, "listing_id", input.ListingID)
		return nil, fmt.Errorf("failed to verify listing: %w", err)
	}
	if !exists {
		return nil, domain.ErrListingNotFound
	}

	// 3. Check rate limits
	ipAddress := ""
	if input.IPAddress != nil {
		ipAddress = *input.IPAddress
	}

	if err := s.rateLimiter.CheckRateLimit(ctx, input.UserID, input.Email, ipAddress, input.ListingID); err != nil {
		s.log.Warn("rate limit exceeded", "email", input.Email, "ip", ipAddress, "listing_id", input.ListingID)
		return nil, err
	}

	// 4. Spam detection (adjusted for verified users)
	spamScore := s.spamDetector.CalculateSpamScore(input.Name, input.Email, input.Message)

	// Verified users get benefit of doubt - reduce spam score by 50%
	if isVerified {
		spamScore = spamScore * 0.5
		s.log.Debug("spam score reduced for verified user", "original_score", spamScore*2, "adjusted_score", spamScore)
	}

	isSpam := s.spamDetector.IsSpam(spamScore)

	if isSpam {
		s.log.Warn("spam detected", "email", input.Email, "spam_score", spamScore, "is_verified", isVerified)
		// Still create the lead but mark as spam
	}

	// 5. Check for duplicates (same email + listing in last 1 hour)
	since := time.Now().Add(-1 * time.Hour)
	existing, err := s.leadRepo.GetLeadByEmailAndListing(ctx, input.Email, input.ListingID, since)
	if err != nil && err != gorm.ErrRecordNotFound {
		s.log.Error("failed to check for duplicate", "error", err)
		return nil, fmt.Errorf("failed to check for duplicate: %w", err)
	}

	if existing != nil {
		// Return existing lead instead of creating duplicate
		s.log.Info("duplicate lead detected, returning existing", "lead_id", existing.ID)
		return domain.MapLeadFromSchema(existing), nil
	}

	// 6. Get listing ownership for business_id
	ownerInfo, err := s.propertyHooks.GetListingOwner(ctx, input.ListingID)
	if err != nil {
		s.log.Error("failed to get listing owner", "error", err)
		return nil, fmt.Errorf("failed to get listing owner: %w", err)
	}

	// 7. Create lead domain model
	lead := &domain.Lead{
		ID:             uuid.New(),
		ListingID:      input.ListingID,
		BusinessID:     ownerInfo.BusinessID,
		UserID:         input.UserID, // Track authenticated user (nil for anonymous)
		IsVerified:     isVerified,   // True if from authenticated user with profile
		Name:           input.Name,
		Email:          input.Email,
		PhoneNumber:    input.PhoneNumber,
		Message:        input.Message,
		Source:         input.Source,
		Status:         domain.StatusNew,
		SpamScore:      spamScore,
		IsSpam:         isSpam,
		UserAgent:      input.UserAgent,
		IPAddress:      input.IPAddress,
		ReferrerURL:    input.ReferrerURL,
		UTMParams:      input.UTMParams,
		CustomMetadata: make(map[string]any),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Auto-mark as spam if score is high
	if isSpam {
		lead.MarkAsSpam()
	}

	// 8. Create lead and event in transaction
	err = s.leadRepo.Transaction(ctx, func(tx *gorm.DB) error {
		// Convert to schema and create
		leadSchema, err := domain.MapLeadToSchema(lead)
		if err != nil {
			return err
		}

		if err := s.leadRepo.CreateLead(ctx, leadSchema); err != nil {
			return fmt.Errorf("failed to create lead: %w", err)
		}

		// Create lead event
		event := &domain.LeadEvent{
			ID:        uuid.New(),
			LeadID:    lead.ID,
			EventType: domain.EventCreated,
			ActorType: domain.ActorSystem,
			NewStatus: &lead.Status,
			Notes:     nil,
			CreatedAt: time.Now(),
		}

		eventSchema, err := domain.MapLeadEventToSchema(event)
		if err != nil {
			return err
		}

		if err := s.eventRepo.CreateLeadEvent(ctx, eventSchema); err != nil {
			return fmt.Errorf("failed to create lead event: %w", err)
		}

		return nil
	})

	if err != nil {
		s.log.Error("failed to create lead", "error", err)
		return nil, err
	}

	// 9. Track analytics (no-op for Phase 1)
	_ = s.analyticsHooks.TrackLeadCreated(ctx, lead.ID, lead.ListingID, string(lead.Source))

	s.log.Info("lead created successfully",
		"lead_id", lead.ID,
		"listing_id", lead.ListingID,
		"user_id", lead.UserID,
		"business_id", lead.BusinessID,
		"email", lead.Email,
		"spam_score", spamScore,
		"is_spam", isSpam,
	)

	// 10. Emit LeadCreated event for async subscribers (e.g., messaging auto-create conversation)
	if s.eventPublisher != nil {
		s.log.Info("publishing lead created event", "lead_id", lead.ID)
		businessIDStr := ""
		if lead.BusinessID != nil {
			businessIDStr = lead.BusinessID.String()
		}

		payload := &payoads.LeadCreatedPayload{
			ID:        lead.ID.String(),
			ListingID: lead.ListingID.String(),
			BusinessID: func() *string {
				if businessIDStr == "" {
					return nil
				}
				return &businessIDStr
			}(),
			UserID: func() *string {
				if lead.UserID == nil {
					return nil
				}
				return func(u *uuid.UUID) *string { s := u.String(); return &s }(lead.UserID)
			}(),
			IsVerified: lead.IsVerified,
			Name:       lead.Name,
			Email:      lead.Email,
			Message:    lead.Message,
			Source:     string(lead.Source),
			Status:     string(lead.Status),
			IsSpam:     lead.IsSpam,
			CreatedAt:  lead.CreatedAt.Format(time.RFC3339),
		}

		tenantID := businessIDStr
		if err := s.eventPublisher.PublishLeadEvent(ctx, events.EventLeadCreated, lead.ID.String(), payload, &tenantID); err != nil {
			s.log.Error("failed to publish lead created event", "lead_id", lead.ID, "error", err)
			// Don't fail the operation - event publishing is non-critical
		}
		s.log.Info("lead created event published", "lead_id", lead.ID)
	}

	return lead, nil
}

// GetLead retrieves a lead by ID with authorization check
func (s *ServiceImpl) GetLead(ctx context.Context, leadID uuid.UUID, requesterID uuid.UUID) (*domain.Lead, error) {
	leadSchema, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrLeadNotFound
		}
		return nil, fmt.Errorf("failed to get lead: %w", err)
	}

	lead := domain.MapLeadFromSchema(leadSchema)

	// Check authorization
	if err := s.CanViewLead(ctx, lead, requesterID); err != nil {
		return nil, err
	}

	return lead, nil
}

// UpdateLeadStatus updates the status of a lead
func (s *ServiceImpl) UpdateLeadStatus(ctx context.Context, leadID uuid.UUID, status domain.LeadStatus, requesterID uuid.UUID, notes *string) (*domain.Lead, error) {
	// Validate status
	if !status.IsValid() {
		return nil, domain.ErrInvalidInput
	}

	// Get lead
	leadSchema, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrLeadNotFound
		}
		return nil, fmt.Errorf("failed to get lead: %w", err)
	}

	lead := domain.MapLeadFromSchema(leadSchema)

	// Check authorization
	if err := s.CanManageLead(ctx, lead, requesterID); err != nil {
		return nil, err
	}

	// Update status
	oldStatus := lead.Status
	if err := lead.UpdateStatus(status); err != nil {
		return nil, err
	}

	// Update in transaction
	err = s.leadRepo.Transaction(ctx, func(tx *gorm.DB) error {
		// Update lead
		updatedSchema, err := domain.MapLeadToSchema(lead)
		if err != nil {
			return err
		}

		if err := s.leadRepo.UpdateLead(ctx, updatedSchema); err != nil {
			return fmt.Errorf("failed to update lead: %w", err)
		}

		// Create event
		event := &domain.LeadEvent{
			ID:        uuid.New(),
			LeadID:    lead.ID,
			EventType: domain.EventStatusChange,
			ActorID:   &requesterID,
			ActorType: domain.ActorUser,
			OldStatus: &oldStatus,
			NewStatus: &status,
			Notes:     notes,
			CreatedAt: time.Now(),
		}

		eventSchema, err := domain.MapLeadEventToSchema(event)
		if err != nil {
			return err
		}

		return s.eventRepo.CreateLeadEvent(ctx, eventSchema)
	})

	if err != nil {
		return nil, err
	}

	s.log.Info("lead status updated",
		"lead_id", leadID,
		"old_status", oldStatus,
		"new_status", status,
		"requester_id", requesterID,
	)

	return lead, nil
}

// AssignLead assigns a lead to a user
func (s *ServiceImpl) AssignLead(ctx context.Context, leadID, assigneeID, requesterID uuid.UUID, reason domain.AssignmentReason) (*domain.Lead, error) {
	// Get lead
	leadSchema, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrLeadNotFound
		}
		return nil, fmt.Errorf("failed to get lead: %w", err)
	}

	lead := domain.MapLeadFromSchema(leadSchema)

	// Check authorization
	if err := s.CanAssignLead(ctx, lead, requesterID); err != nil {
		return nil, err
	}

	// Check if lead can be assigned
	if !lead.CanBeAssigned() {
		return nil, domain.ErrCannotAssignLead
	}

	// Track previous assignee for assignment history
	var fromUserID *uuid.UUID
	if lead.AssignedTo != nil {
		fromUserID = lead.AssignedTo
	}

	// Assign lead
	isAuto := reason == domain.ReasonAuto
	lead.Assign(assigneeID, isAuto)

	// Update in transaction
	err = s.leadRepo.Transaction(ctx, func(tx *gorm.DB) error {
		// Update lead
		updatedSchema, err := domain.MapLeadToSchema(lead)
		if err != nil {
			return err
		}

		if err := s.leadRepo.UpdateLead(ctx, updatedSchema); err != nil {
			return fmt.Errorf("failed to update lead: %w", err)
		}

		// Create assignment record
		assignment := &domain.LeadAssignment{
			ID:         uuid.New(),
			LeadID:     lead.ID,
			FromUserID: fromUserID,
			ToUserID:   assigneeID,
			Reason:     reason,
			AssignedBy: &requesterID,
			AssignedAt: time.Now(),
		}

		assignmentSchema, err := domain.MapLeadAssignmentToSchema(assignment)
		if err != nil {
			return err
		}

		if err := s.assignmentRepo.CreateLeadAssignment(ctx, assignmentSchema); err != nil {
			return fmt.Errorf("failed to create assignment: %w", err)
		}

		// Create event
		event := &domain.LeadEvent{
			ID:        uuid.New(),
			LeadID:    lead.ID,
			EventType: domain.EventAssigned,
			ActorID:   &requesterID,
			ActorType: domain.ActorUser,
			Changes: map[string]any{
				"assigned_to":   assigneeID.String(),
				"assigned_from": fromUserID,
				"reason":        string(reason),
			},
			CreatedAt: time.Now(),
		}

		eventSchema, err := domain.MapLeadEventToSchema(event)
		if err != nil {
			return err
		}

		return s.eventRepo.CreateLeadEvent(ctx, eventSchema)
	})

	if err != nil {
		return nil, err
	}

	s.log.Info("lead assigned",
		"lead_id", leadID,
		"assignee_id", assigneeID,
		"from_user_id", fromUserID,
		"reason", reason,
		"requester_id", requesterID,
	)

	return lead, nil
}

// MarkAsSpam marks a lead as spam
func (s *ServiceImpl) MarkAsSpam(ctx context.Context, leadID, requesterID uuid.UUID) error {
	leadSchema, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.ErrLeadNotFound
		}
		return fmt.Errorf("failed to get lead: %w", err)
	}

	lead := domain.MapLeadFromSchema(leadSchema)

	// Check authorization
	if err := s.CanManageLead(ctx, lead, requesterID); err != nil {
		return err
	}

	// Mark as spam
	oldStatus := lead.Status
	lead.MarkAsSpam()

	// Update in transaction
	err = s.leadRepo.Transaction(ctx, func(tx *gorm.DB) error {
		updatedSchema, err := domain.MapLeadToSchema(lead)
		if err != nil {
			return err
		}

		if err := s.leadRepo.UpdateLead(ctx, updatedSchema); err != nil {
			return fmt.Errorf("failed to update lead: %w", err)
		}

		// Create event
		event := &domain.LeadEvent{
			ID:        uuid.New(),
			LeadID:    lead.ID,
			EventType: domain.EventMarkedSpam,
			ActorID:   &requesterID,
			ActorType: domain.ActorUser,
			OldStatus: &oldStatus,
			NewStatus: &lead.Status,
			CreatedAt: time.Now(),
		}

		eventSchema, err := domain.MapLeadEventToSchema(event)
		if err != nil {
			return err
		}

		return s.eventRepo.CreateLeadEvent(ctx, eventSchema)
	})

	if err != nil {
		return err
	}

	s.log.Info("lead marked as spam", "lead_id", leadID, "requester_id", requesterID)
	return nil
}

// DeleteLead soft deletes a lead
func (s *ServiceImpl) DeleteLead(ctx context.Context, leadID, requesterID uuid.UUID) error {
	leadSchema, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.ErrLeadNotFound
		}
		return fmt.Errorf("failed to get lead: %w", err)
	}

	lead := domain.MapLeadFromSchema(leadSchema)

	// Check authorization
	if err := s.CanManageLead(ctx, lead, requesterID); err != nil {
		return err
	}

	// Delete lead
	if err := s.leadRepo.DeleteLead(ctx, leadID); err != nil {
		return fmt.Errorf("failed to delete lead: %w", err)
	}

	s.log.Info("lead deleted", "lead_id", leadID, "requester_id", requesterID)
	return nil
}

// ListLeadsByListing retrieves leads for a specific listing
func (s *ServiceImpl) ListLeadsByListing(ctx context.Context, listingID, requesterID uuid.UUID, filter LeadFilter, page Pagination) ([]*domain.Lead, int64, error) {
	// Check authorization
	if err := s.CanListLeadsForListing(ctx, listingID, requesterID); err != nil {
		return nil, 0, err
	}

	// Convert filter
	repoFilter := convertToRepoFilter(filter)
	repoPage := convertToRepoPagination(page)

	// Get leads
	leadsSchema, total, err := s.leadRepo.ListLeadsByListing(ctx, listingID, repoFilter, repoPage)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list leads: %w", err)
	}

	leads := domain.MapLeadsFromSchema(leadsSchema)
	return leads, total, nil
}

// ListLeadsByBusiness retrieves leads for a business
func (s *ServiceImpl) ListLeadsByBusiness(ctx context.Context, businessID, requesterID uuid.UUID, filter LeadFilter, page Pagination) ([]*domain.Lead, int64, error) {
	// Check authorization
	if err := s.CanListLeadsForBusiness(ctx, businessID, requesterID); err != nil {
		return nil, 0, err
	}

	// Convert filter
	repoFilter := convertToRepoFilter(filter)
	repoPage := convertToRepoPagination(page)

	// Get leads
	leadsSchema, total, err := s.leadRepo.ListLeadsByBusiness(ctx, businessID, repoFilter, repoPage)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list leads: %w", err)
	}

	leads := domain.MapLeadsFromSchema(leadsSchema)
	return leads, total, nil
}

// ListMyLeads retrieves leads assigned to the requester
func (s *ServiceImpl) ListMyLeads(ctx context.Context, requesterID uuid.UUID, filter LeadFilter, page Pagination) ([]*domain.Lead, int64, error) {
	// Convert filter
	repoFilter := convertToRepoFilter(filter)
	repoPage := convertToRepoPagination(page)

	// Get leads
	leadsSchema, total, err := s.leadRepo.ListLeadsByAssignee(ctx, requesterID, repoFilter, repoPage)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list leads: %w", err)
	}

	leads := domain.MapLeadsFromSchema(leadsSchema)
	return leads, total, nil
}

// GetLeadHistory retrieves the event history for a lead
func (s *ServiceImpl) GetLeadHistory(ctx context.Context, leadID, requesterID uuid.UUID) ([]*domain.LeadEvent, error) {
	leadSchema, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrLeadNotFound
		}
		return nil, fmt.Errorf("failed to get lead: %w", err)
	}

	lead := domain.MapLeadFromSchema(leadSchema)

	// Check authorization
	if err := s.CanViewLead(ctx, lead, requesterID); err != nil {
		return nil, err
	}

	// Get events
	eventsSchema, err := s.eventRepo.ListLeadEvents(ctx, leadID, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	events := domain.MapLeadEventsFromSchema(eventsSchema)
	return events, nil
}

// Helper functions

func convertToRepoFilter(filter LeadFilter) repository.LeadFilter {
	var statusStrings []string
	for _, status := range filter.Status {
		statusStrings = append(statusStrings, string(status))
	}

	var sourceStrings []string
	for _, source := range filter.Source {
		sourceStrings = append(sourceStrings, string(source))
	}

	return repository.LeadFilter{
		Status:     statusStrings,
		Source:     sourceStrings,
		IsSpam:     filter.IsSpam,
		Assigned:   filter.Assigned,
		DateFrom:   filter.DateFrom,
		DateTo:     filter.DateTo,
		SearchTerm: filter.SearchTerm,
	}
}

func convertToRepoPagination(page Pagination) repository.Pagination {
	return repository.Pagination{
		Limit:  page.Limit,
		Offset: page.Offset,
	}
}
