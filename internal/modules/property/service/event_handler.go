package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hauslet/internal/modules/property/domain"
	"hauslet/internal/platform/events"
	"hauslet/internal/platform/events/payoads" // Typo in package name supported
)

// SubscribeToVerificationEvents subscribes to verification events
func (s *ServiceImpl) SubscribeToVerificationEvents(ctx context.Context, subscriber *events.Subscriber) error {
	if subscriber == nil {
		return fmt.Errorf("subscriber is nil")
	}

	s.log.Info("subscribing to verification events")

	// Subscribe to verifications channel
	eventCh, err := subscriber.SubscribeWithFilter(ctx, events.ChannelVerifications, func(e *events.Event) bool {
		return e.Type == events.EventVerificationCompleted || e.Type == events.EventVerificationFailed
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to verification events: %w", err)
	}

	go s.processVerificationEvents(ctx, eventCh)

	return nil
}

func (s *ServiceImpl) processVerificationEvents(ctx context.Context, eventCh <-chan *events.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			// Process with timeout
			processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			if err := s.handleVerificationEvent(processCtx, event); err != nil {
				s.log.Error("failed to process verification event", "event_id", event.ID, "error", err)
			}
			cancel()
		}
	}
}

func (s *ServiceImpl) handleVerificationEvent(ctx context.Context, event *events.Event) error {
	// Filter for relevant events
	if event.Type != events.EventVerificationCompleted && event.Type != events.EventVerificationFailed {
		return nil
	}

	var payload payoads.VerificationCompletedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal verification payload: %w", err)
	}

	// Only process listing verifications
	if payload.TargetType != "listing" {
		return nil
	}

	if payload.TargetID == nil {
		return fmt.Errorf("listing verification missing target ID")
	}

	listingID := *payload.TargetID

	// Get listing details for email
	listing, err := s.GetListingByID(ctx, listingID, false)
	if err != nil {
		return fmt.Errorf("failed to get listing details for notification: %w", err)
	}

	// Get owner details
	// We might need to fetch owner email from profile service?
	// Or maybe the listing object has it if we preloaded it?
	// Listing domain model has OwnerID. We need to fetch User Profile.
	firstName, email, err := s.profiles.GetProfileData(ctx, listing.OwnerID.String())
	if err != nil {
		s.log.Error("failed to get owner profile for notification", "owner_id", listing.OwnerID, "error", err)
		// Continue? We can't send email without email address.
		// If this is just a status update, we can proceed. But if we want to send notification, we are stuck.
		// Let's assume we can proceed with DB updates at least.
	}

	if event.Type == events.EventVerificationCompleted && payload.Status == "approved" {
		// Map tier to verification level
		var level domain.VerificationLevel
		switch payload.VerificationTier {
		case "enhanced":
			level = domain.VerificationLevelPremium
		case "standard":
			level = domain.VerificationLevelPlus
		case "basic":
			level = domain.VerificationLevelBasic
		default:
			level = domain.VerificationLevelBasic
		}

		updates := map[string]any{
			"is_verified":        true,
			"verification_level": level,
			"verified_at":        payload.CompletedAt,
		}

		s.log.Info("updating listing verification status and notifying", "listing_id", listingID, "level", level)

		if _, err := s.PatchListing(ctx, listingID, updates); err != nil {
			return fmt.Errorf("failed to patch listing verification status: %w", err)
		}

		// Send success notification
		if email != "" {
			if err := s.notificationService.SendListingVerificationAcceptedNotification(ctx, listing.Title, firstName, email); err != nil {
				s.log.Error("failed to send listing verified notification", "listing_id", listingID, "error", err)
			}
		}

	} else if (event.Type == events.EventVerificationCompleted && payload.Status == "rejected") || event.Type == events.EventVerificationFailed {

		s.log.Info("listing verification rejected, sending notification", "listing_id", listingID)

		// Note: We don't update Listing status in DB?
		// Listing likely stays in its current status (Active/Pending?) but verification failed.
		// Maybe we should clear any pending flags if we have them.

		// Send rejection notification
		reason := "Verification failed"
		if payload.RejectionReason != nil {
			reason = *payload.RejectionReason
		}

		if email != "" {
			if err := s.notificationService.SendListingVerificationRejectedNotification(ctx, listing.Title, firstName, email, reason); err != nil {
				s.log.Error("failed to send listing verification rejected notification", "listing_id", listingID, "error", err)
			}
		}
	}

	return nil
}
