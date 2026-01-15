// Package events provides a Redis-based pub/sub event system for real-time subscriptions.
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of event being published.
type EventType string

// Domain event types organized by module.
const (
	// Booking events
	EventBookingCreated  EventType = "booking.created"
	EventBookingUpdated  EventType = "booking.updated"
	EventBookingCanceled EventType = "booking.canceled"

	// Payment events
	EventPaymentProcessed EventType = "payment.processed"
	EventPaymentFailed    EventType = "payment.failed"
	EventPayoutCompleted  EventType = "payout.completed"

	// Property events
	EventPropertyCreated EventType = "property.created"
	EventPropertyUpdated EventType = "property.updated"

	// Review events
	EventReviewCreated EventType = "review.created"

	// Lead events
	EventLeadCreated EventType = "lead.created"
	EventLeadUpdated EventType = "lead.updated"

	// Verification events
	EventVerificationStarted   EventType = "verification.started"
	EventVerificationCompleted EventType = "verification.completed"
	EventVerificationFailed    EventType = "verification.failed"

	// Message/Conversation events
	EventMessageSent         EventType = "message.sent"
	EventMessageRead         EventType = "message.read"
	EventConversationCreated EventType = "conversation.created"
	EventConversationUpdated EventType = "conversation.updated"
	EventTypingIndicator     EventType = "typing.indicator"
)

// Event represents a domain event that can be published and subscribed to.
type Event struct {
	// ID is a unique identifier for this event instance.
	ID string `json:"id"`

	// Type indicates what kind of event this is.
	Type EventType `json:"type"`

	// EntityID is the primary identifier of the entity that triggered this event.
	// For example: bookingID, propertyID, userID.
	EntityID string `json:"entity_id"`

	// TenantID is the business/organization ID for multi-tenant filtering.
	// Nil for user-scoped events.
	TenantID *string `json:"tenant_id,omitempty"`

	// ActorID is the user who triggered this event (optional).
	ActorID *string `json:"actor_id,omitempty"`

	// Payload contains the actual event data (marshaled domain entity).
	Payload json.RawMessage `json:"payload"`

	// Timestamp is when this event was created.
	Timestamp time.Time `json:"timestamp"`

	// Metadata for additional context (optional).
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewEvent creates a new event with a generated ID and timestamp.
func NewEvent(eventType EventType, entityID string, payload interface{}) (*Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		EntityID:  entityID,
		Payload:   data,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]string),
	}, nil
}

// WithTenant adds tenant context to the event.
func (e *Event) WithTenant(tenantID string) *Event {
	e.TenantID = &tenantID
	return e
}

// WithActor adds actor (user) context to the event.
func (e *Event) WithActor(actorID string) *Event {
	e.ActorID = &actorID
	return e
}

// WithMetadata adds custom metadata to the event.
func (e *Event) WithMetadata(key, value string) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// Channel represents a pub/sub channel name.
type Channel string

// Predefined channels organized by domain entity.
const (
	ChannelBookings      Channel = "bookings"
	ChannelPayments      Channel = "payments"
	ChannelProperties    Channel = "properties"
	ChannelReviews       Channel = "reviews"
	ChannelLeads         Channel = "leads"
	ChannelVerifications Channel = "verifications"
	ChannelMessages      Channel = "messages"
)

// String returns the string representation of the channel.
func (c Channel) String() string {
	return string(c)
}
