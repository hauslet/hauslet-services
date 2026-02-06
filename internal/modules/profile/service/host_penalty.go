package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"hauslet/config"
	profilerepo "hauslet/internal/modules/profile/repository"
	profileschema "hauslet/internal/modules/profile/repository/schema"
	propertyrepo "hauslet/internal/modules/property/repository"

	"github.com/google/uuid"
)

// HostPenaltyService defines operations for progressive host cancellation penalties.
type HostPenaltyService interface {
	// GetCancellationCountInWindow returns number of cancellations in the rolling window.
	GetCancellationCountInWindow(ctx context.Context, hostID uuid.UUID) (int, error)

	// CalculatePenalty determines the penalty for the next cancellation.
	CalculatePenalty(ctx context.Context, hostID uuid.UUID) (*PenaltyResult, error)

	// RecordCancellation logs the cancellation and penalty outcome.
	RecordCancellation(ctx context.Context, input RecordCancellationInput) error

	// ApplySuspension suspends all host listings for N days.
	ApplySuspension(ctx context.Context, hostID uuid.UUID, days int, reason string) error

	// CheckAndLiftSuspensions lifts expired listing suspensions.
	CheckAndLiftSuspensions(ctx context.Context) error

	// NotifySuspensionEndingSoon sends notifications to hosts whose suspensions end within the configured notice window.
	NotifySuspensionEndingSoon(ctx context.Context, notifier SuspensionNotifier) (int, error)
}

// SuspensionNotifier defines an interface for sending suspension-related notifications.
type SuspensionNotifier interface {
	SendHostSuspensionEnding(ctx context.Context, hostName, hostEmail string, suspensionEndTime time.Time)
}

// PenaltyResult describes the penalty for a cancellation.
type PenaltyResult struct {
	CancellationCount    int
	PenaltyAmount        int64 // Minor units
	SuspensionDays       int
	RequiresReview       bool
	IsNewHostGracePeriod bool
	WarningMessage       string
}

// RecordCancellationInput captures cancellation record details.
type RecordCancellationInput struct {
	HostID        uuid.UUID
	BookingID     uuid.UUID
	CancelledAt   time.Time
	Reason        string
	PenaltyAmount int64
	PenaltyPaid   bool
	WarningSent   bool
}

// HostPenaltyServiceImpl implements HostPenaltyService.
type HostPenaltyServiceImpl struct {
	profileRepo profilerepo.ProfileRepository
	listingRepo propertyrepo.ListingRepository
	config      config.PlatformYAMLConfig
	log         *slog.Logger
	now         func() time.Time
}

// NewHostPenaltyService creates a new HostPenaltyService.
func NewHostPenaltyService(
	profileRepo profilerepo.ProfileRepository,
	listingRepo propertyrepo.ListingRepository,
	platformConfig config.PlatformYAMLConfig,
	log *slog.Logger,
) HostPenaltyService {
	return &HostPenaltyServiceImpl{
		profileRepo: profileRepo,
		listingRepo: listingRepo,
		config:      platformConfig,
		log:         log,
		now:         time.Now,
	}
}

// GetCancellationCountInWindow returns cancellation count in rolling window.
func (s *HostPenaltyServiceImpl) GetCancellationCountInWindow(ctx context.Context, hostID uuid.UUID) (int, error) {
	windowDays := s.config.HostCancellation.WindowDays
	if windowDays <= 0 {
		windowDays = 30
	}
	cutoff := s.now().AddDate(0, 0, -windowDays)
	return s.profileRepo.CountHostCancellationsSince(ctx, hostID, cutoff)
}

// CalculatePenalty determines the penalty for the next cancellation.
func (s *HostPenaltyServiceImpl) CalculatePenalty(ctx context.Context, hostID uuid.UUID) (*PenaltyResult, error) {
	if hostID == uuid.Nil {
		return nil, errors.New("hostID cannot be nil")
	}

	profile, err := s.profileRepo.GetProfileByUserID(ctx, hostID.String())
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, fmt.Errorf("profile not found for host %s", hostID.String())
	}

	graceDays := s.config.HostCancellation.NewHostGraceDays
	if graceDays > 0 {
		if s.now().Sub(profile.CreatedAt).Hours() < float64(graceDays*24) {
			return &PenaltyResult{
				CancellationCount:    1,
				IsNewHostGracePeriod: true,
				WarningMessage: fmt.Sprintf(
					"You're within your %d-day new host grace period. This cancellation won't incur a penalty, but please avoid frequent cancellations.",
					graceDays,
				),
			}, nil
		}
	}

	windowDays := s.config.HostCancellation.WindowDays
	if windowDays <= 0 {
		windowDays = 30
	}
	cutoff := s.now().AddDate(0, 0, -windowDays)
	count, err := s.profileRepo.CountHostCancellationsSince(ctx, hostID, cutoff)
	if err != nil {
		return nil, err
	}

	nextCount := count + 1
	tier := selectPenaltyTier(s.config.HostCancellation.Penalties, nextCount)

	result := &PenaltyResult{
		CancellationCount: nextCount,
		PenaltyAmount:     tier.Amount,
		SuspensionDays:    tier.SuspensionDays,
		RequiresReview:    tier.RequiresReview,
	}

	minorUnit := s.config.Currency.MinorUnit
	if minorUnit <= 0 {
		minorUnit = 100
	}

	if tier.Amount == 0 {
		result.WarningMessage = fmt.Sprintf(
			"This is your first cancellation in %d days. No penalty will apply, but repeated cancellations will incur fees.",
			windowDays,
		)
	} else {
		currencyCode := s.config.Currency.Code
		if currencyCode == "" {
			currencyCode = "NGN"
		}
		amountText := fmt.Sprintf("%s %.2f", currencyCode, float64(tier.Amount)/float64(minorUnit))
		result.WarningMessage = fmt.Sprintf(
			"This is cancellation #%d in the last %d days. A penalty of %s will be deducted from your wallet.",
			nextCount,
			windowDays,
			amountText,
		)
		if tier.SuspensionDays > 0 {
			result.WarningMessage += fmt.Sprintf(" Your listings will be suspended for %d days.", tier.SuspensionDays)
		}
	}

	return result, nil
}

// RecordCancellation logs the cancellation and penalty outcome.
func (s *HostPenaltyServiceImpl) RecordCancellation(ctx context.Context, input RecordCancellationInput) error {
	if input.HostID == uuid.Nil || input.BookingID == uuid.Nil {
		return errors.New("hostID and bookingID are required")
	}

	existing, err := s.profileRepo.GetHostCancellationRecordByBookingID(ctx, input.BookingID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	record := &profileschema.HostCancellationRecord{
		HostID:        input.HostID,
		BookingID:     input.BookingID,
		CancelledAt:   input.CancelledAt,
		PenaltyAmount: input.PenaltyAmount,
		PenaltyPaid:   input.PenaltyPaid,
		WarningSent:   input.WarningSent,
		Reason:        input.Reason,
		CreatedAt:     s.now(),
	}

	return s.profileRepo.CreateHostCancellationRecord(ctx, record)
}

// ApplySuspension suspends all host listings for N days.
func (s *HostPenaltyServiceImpl) ApplySuspension(ctx context.Context, hostID uuid.UUID, days int, reason string) error {
	if hostID == uuid.Nil || days <= 0 {
		return nil
	}
	if s.listingRepo == nil {
		return errors.New("listing repository not configured")
	}

	suspendUntil := s.now().AddDate(0, 0, days)
	filter := propertyrepo.ListingFilter{
		OwnerID:        &hostID,
		IncludeDeleted: false,
	}

	page := propertyrepo.Pagination{Limit: 200, Offset: 0}
	for {
		result, err := s.listingRepo.ListListings(ctx, filter, page)
		if err != nil {
			return err
		}
		if len(result.Items) == 0 {
			break
		}

		for _, listing := range result.Items {
			updateUntil := suspendUntil
			if listing.SuspendedUntil != nil && listing.SuspendedUntil.After(updateUntil) {
				updateUntil = *listing.SuspendedUntil
			}

			updates := map[string]any{
				"suspended_until":               updateUntil,
				"suspension_reason":             reason,
				"suspension_ending_notified_at": nil, // Reset notification flag for new suspension
			}
			if err := s.listingRepo.PatchListing(ctx, listing.ID, updates); err != nil {
				if s.log != nil {
					s.log.Warn("failed to suspend listing", "listing_id", listing.ID, "error", err)
				}
			}
		}

		page.Offset += page.Limit
	}

	return nil
}

// CheckAndLiftSuspensions lifts expired listing suspensions.
func (s *HostPenaltyServiceImpl) CheckAndLiftSuspensions(ctx context.Context) error {
	if s.listingRepo == nil {
		return nil
	}

	listings, err := s.listingRepo.FindExpiredSuspensions(ctx, s.now())
	if err != nil {
		return err
	}

	for _, listing := range listings {
		updates := map[string]any{
			"suspended_until":   nil,
			"suspension_reason": "",
		}
		if err := s.listingRepo.PatchListing(ctx, listing.ID, updates); err != nil {
			if s.log != nil {
				s.log.Warn("failed to lift listing suspension", "listing_id", listing.ID, "error", err)
			}
		}
	}

	return nil
}

// NotifySuspensionEndingSoon sends notifications to hosts with suspensions ending soon.
func (s *HostPenaltyServiceImpl) NotifySuspensionEndingSoon(ctx context.Context, notifier SuspensionNotifier) (int, error) {
	if s.listingRepo == nil || notifier == nil {
		return 0, nil
	}

	noticeHours := s.config.HostCancellation.SuspensionEndingNotice
	if noticeHours <= 0 {
		noticeHours = 24 // Default to 24 hours
	}

	now := s.now()
	noticeFrom := now
	noticeTo := now.Add(time.Duration(noticeHours) * time.Hour)

	listings, err := s.listingRepo.FindSuspensionsEndingSoon(ctx, noticeFrom, noticeTo)
	if err != nil {
		return 0, err
	}

	if len(listings) == 0 {
		return 0, nil
	}

	// Deduplicate by owner ID to avoid multiple notifications
	notifiedHosts := make(map[uuid.UUID]bool)
	notificationCount := 0

	// Collect listing IDs to mark as notified
	var listingsToMark []uuid.UUID

	for _, listing := range listings {
		listingsToMark = append(listingsToMark, listing.ID)

		if notifiedHosts[listing.OwnerID] {
			continue
		}
		notifiedHosts[listing.OwnerID] = true

		profile, err := s.profileRepo.GetProfileByUserID(ctx, listing.OwnerID.String())
		if err != nil {
			if s.log != nil {
				s.log.Warn("failed to get profile for suspension ending notification", "owner_id", listing.OwnerID, "error", err)
			}
			continue
		}
		if profile == nil || profile.Email == nil || *profile.Email == "" {
			continue
		}

		notifier.SendHostSuspensionEnding(ctx, profile.FullName, *profile.Email, *listing.SuspendedUntil)
		notificationCount++
	}

	// Mark all listings as notified to prevent duplicate notifications
	for _, listingID := range listingsToMark {
		if err := s.listingRepo.PatchListing(ctx, listingID, map[string]any{
			"suspension_ending_notified_at": now,
		}); err != nil {
			if s.log != nil {
				s.log.Warn("failed to mark listing as notified", "listing_id", listingID, "error", err)
			}
		}
	}

	if s.log != nil {
		s.log.Info("sent suspension ending notifications", "count", notificationCount)
	}

	return notificationCount, nil
}

func selectPenaltyTier(tiers []config.PenaltyTier, count int) config.PenaltyTier {
	var selected config.PenaltyTier
	for _, tier := range tiers {
		if tier.Count <= 0 {
			continue
		}
		if tier.Count <= count && tier.Count >= selected.Count {
			selected = tier
		}
	}
	return selected
}
