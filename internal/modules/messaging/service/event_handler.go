package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hauslet/internal/platform/events"
	"hauslet/internal/platform/events/payoads"

	"github.com/google/uuid"
)

// HandleLeadCreatedEvent processes lead.created events from the leads module.
// It automatically creates an inquiry conversation for authenticated leads.
func (s *messagingServiceImpl) HandleLeadCreatedEvent(ctx context.Context, event *events.Event) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	s.log.Debug("handling lead created event", "event_id", event.ID, "entity_id", event.EntityID)

	// Deserialize payload
	var payload payoads.LeadCreatedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		s.log.Error("failed to unmarshal lead created payload", "event_id", event.ID, "error", err)
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Only auto-create conversations for authenticated users
	// Anonymous leads don't have a user to initiate conversation with
	if payload.UserID == nil {
		s.log.Debug("skipping conversation creation for anonymous lead", "lead_id", payload.ID)
		return nil
	}

	leadID, err := uuid.Parse(payload.ID)
	if err != nil {
		s.log.Error("failed to parse lead ID from event", "lead_id", payload.ID, "error", err)
		return fmt.Errorf("invalid lead id: %w", err)
	}

	userID, err := uuid.Parse(*payload.UserID)
	if err != nil {
		s.log.Error("failed to parse user ID from event", "user_id", *payload.UserID, "error", err)
		return fmt.Errorf("invalid user id: %w", err)
	}

	// Create conversation using system user context
	// This bypasses lead access validation since the event is trusted
	_, err = s.getOrCreateInquiryConversation(ctx, leadID, userID, true)
	if err != nil {
		s.log.Error("failed to create inquiry conversation from event",
			"lead_id", leadID,
			"user_id", userID,
			"error", err,
		)
		// Don't fail - event processing is best-effort
		// The conversation can be created on-demand if accessed later
		return nil
	}

	s.log.Info("conversation created from lead created event",
		"lead_id", leadID,
		"user_id", userID,
		"event_id", event.ID,
	)

	return nil
}

// SubscribeToLeadEvents starts a background goroutine that listens for lead events
// and processes them. This should be called during service initialization.
func (s *messagingServiceImpl) SubscribeToLeadEvents(ctx context.Context, subscriber *events.Subscriber) error {
	if subscriber == nil {
		return fmt.Errorf("subscriber is nil")
	}

	s.log.Info("subscribing to lead events")

	// Subscribe to lead channel with filter for LeadCreated events
	eventCh, err := subscriber.SubscribeWithFilter(ctx, events.ChannelLeads, func(e *events.Event) bool {
		// Only process LeadCreated events
		return e.Type == events.EventLeadCreated
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to lead events: %w", err)
	}

	// Start background goroutine to process events
	go s.processLeadEvents(ctx, eventCh)

	return nil
}

// processLeadEvents is an internal goroutine that processes lead events as they arrive
func (s *messagingServiceImpl) processLeadEvents(ctx context.Context, eventCh <-chan *events.Event) {
	for {
		select {
		case <-ctx.Done():
			s.log.Debug("stopping lead event processor")
			return

		case event, ok := <-eventCh:
			if !ok {
				s.log.Debug("lead event channel closed")
				return
			}

			if event == nil {
				continue
			}

			// Process event with timeout to prevent blocking
			processCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			if err := s.HandleLeadCreatedEvent(processCtx, event); err != nil {
				s.log.Error("error processing lead event",
					"event_id", event.ID,
					"error", err,
				)
			}
			cancel()
		}
	}
}
