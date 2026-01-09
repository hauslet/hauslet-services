package events

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Publisher provides a service-friendly interface for publishing events.
// It wraps the Broker with convenience methods and error handling.
type Publisher struct {
	broker Broker
	log    *slog.Logger
}

// NewPublisher creates a new event publisher.
func NewPublisher(broker Broker, log *slog.Logger) *Publisher {
	return &Publisher{
		broker: broker,
		log:    log,
	}
}

// PublishOptions configures event publishing behavior.
type PublishOptions struct {
	// TenantID for multi-tenant filtering
	TenantID *string

	// ActorID is the user who triggered this event
	ActorID *string

	// Metadata for additional context
	Metadata map[string]string

	// FailSilently determines whether to log errors or return them.
	// Default is true (log only) since subscriptions are optional.
	FailSilently bool
}

// Publish publishes an event to a channel.
// This is the main method services should use.
func (p *Publisher) Publish(ctx context.Context, channel Channel, eventType EventType, entityID string, payload interface{}, opts *PublishOptions) error {
	// Create event
	event, err := NewEvent(eventType, entityID, payload)
	if err != nil {
		return p.handleError(fmt.Errorf("failed to create event: %w", err), opts)
	}

	// Apply options
	if opts != nil {
		if opts.TenantID != nil {
			event.WithTenant(*opts.TenantID)
		}
		if opts.ActorID != nil {
			event.WithActor(*opts.ActorID)
		}
		if opts.Metadata != nil {
			for k, v := range opts.Metadata {
				event.WithMetadata(k, v)
			}
		}
	}

	// Publish with timeout to prevent blocking
	publishCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := p.broker.Publish(publishCtx, channel, event); err != nil {
		return p.handleError(fmt.Errorf("failed to publish event: %w", err), opts)
	}

	return nil
}

// PublishBookingEvent is a convenience method for booking events.
func (p *Publisher) PublishBookingEvent(ctx context.Context, eventType EventType, bookingID string, payload interface{}, tenantID *string) error {
	return p.Publish(ctx, ChannelBookings, eventType, bookingID, payload, &PublishOptions{
		TenantID:     tenantID,
		FailSilently: true,
	})
}

// PublishPaymentEvent is a convenience method for payment events.
func (p *Publisher) PublishPaymentEvent(ctx context.Context, eventType EventType, paymentID string, payload interface{}, tenantID *string) error {
	return p.Publish(ctx, ChannelPayments, eventType, paymentID, payload, &PublishOptions{
		TenantID:     tenantID,
		FailSilently: true,
	})
}

// PublishPropertyEvent is a convenience method for property events.
func (p *Publisher) PublishPropertyEvent(ctx context.Context, eventType EventType, propertyID string, payload interface{}, tenantID *string) error {
	return p.Publish(ctx, ChannelProperties, eventType, propertyID, payload, &PublishOptions{
		TenantID:     tenantID,
		FailSilently: true,
	})
}

// PublishReviewEvent is a convenience method for review events.
func (p *Publisher) PublishReviewEvent(ctx context.Context, eventType EventType, reviewID string, payload interface{}) error {
	return p.Publish(ctx, ChannelReviews, eventType, reviewID, payload, &PublishOptions{
		FailSilently: true,
	})
}

// PublishLeadEvent is a convenience method for lead events.
func (p *Publisher) PublishLeadEvent(ctx context.Context, eventType EventType, leadID string, payload interface{}, tenantID *string) error {
	return p.Publish(ctx, ChannelLeads, eventType, leadID, payload, &PublishOptions{
		TenantID:     tenantID,
		FailSilently: true,
	})
}

// PublishVerificationEvent is a convenience method for verification events.
func (p *Publisher) PublishVerificationEvent(ctx context.Context, eventType EventType, verificationID string, payload interface{}, userID *string) error {
	return p.Publish(ctx, ChannelVerifications, eventType, verificationID, payload, &PublishOptions{
		ActorID:      userID,
		FailSilently: true,
	})
}

// handleError handles publishing errors based on options.
func (p *Publisher) handleError(err error, opts *PublishOptions) error {
	failSilently := true
	if opts != nil {
		failSilently = opts.FailSilently
	}

	if failSilently {
		// Log error but don't fail the operation
		p.log.Error("Event publishing failed (non-critical)", "error", err)
		return nil
	}

	return err
}

// Health checks if the publisher's broker is healthy.
func (p *Publisher) Health(ctx context.Context) error {
	return p.broker.Health(ctx)
}
