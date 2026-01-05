package domain

import (
	"time"

	"github.com/google/uuid"
)

// Interaction represents a user interaction event in the domain model
type Interaction struct {
	ID        uuid.UUID  `json:"id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`    // Nullable for anonymous users
	SessionID string     `json:"session_id"`            // Required (cookie/fingerprint)

	// Core Data
	Type       InteractionType    `json:"type"`
	EntityType EntityType         `json:"entity_type"`
	EntityID   *uuid.UUID         `json:"entity_id,omitempty"`

	// Context and Metadata
	Context  map[string]any `json:"context,omitempty"`  // Source, Position in list, SearchQuery, etc.
	Metadata InteractionMeta `json:"metadata"`

	// Technical
	IsBot     bool      `json:"is_bot"`
	CreatedAt time.Time `json:"created_at"`
}

// InteractionMeta contains technical metadata about the interaction
type InteractionMeta struct {
	DeviceType DeviceType `json:"device_type"`
	Platform   Platform   `json:"platform"`
	IPHash     string     `json:"ip_hash,omitempty"`     // Anonymized IP for unique counting
	UserAgent  string     `json:"user_agent,omitempty"`
	Referrer   string     `json:"referrer,omitempty"`
}

// NewInteraction creates a new interaction with generated ID and timestamp
func NewInteraction(
	userID *uuid.UUID,
	sessionID string,
	interactionType InteractionType,
	entityType EntityType,
	entityID *uuid.UUID,
	context map[string]any,
	metadata InteractionMeta,
) *Interaction {
	return &Interaction{
		ID:         uuid.New(),
		UserID:     userID,
		SessionID:  sessionID,
		Type:       interactionType,
		EntityType: entityType,
		EntityID:   entityID,
		Context:    context,
		Metadata:   metadata,
		IsBot:      false,
		CreatedAt:  time.Now(),
	}
}

// Validate checks if the interaction is valid
func (i *Interaction) Validate() error {
	if i.SessionID == "" {
		return ErrInvalidSessionID
	}

	if !i.Type.IsValid() {
		return ErrInvalidInteractionType
	}

	if !i.EntityType.IsValid() {
		return ErrInvalidEntityType
	}

	// Entity ID is required for most interaction types except search
	if i.EntityType != EntityTypeSearch && i.EntityID == nil {
		return ErrMissingEntityID
	}

	return nil
}

// IsAnonymous returns true if the interaction is from an anonymous user
func (i *Interaction) IsAnonymous() bool {
	return i.UserID == nil
}

// IsHighValue returns true if this is a high-value interaction
func (i *Interaction) IsHighValue() bool {
	return i.Type.IsHighValue()
}

// GetUserIdentifier returns user ID if available, otherwise session ID
func (i *Interaction) GetUserIdentifier() string {
	if i.UserID != nil {
		return i.UserID.String()
	}
	return i.SessionID
}
