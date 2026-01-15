package payoads

// Domain Event Payloads
// These represent the data structure of events published to different channels.
// Each payload contains the minimal necessary information for subscribers to react.

// LeadCreatedPayload represents a lead.created event.
type LeadCreatedPayload struct {
	ID         string  `json:"id"`
	ListingID  string  `json:"listing_id"`
	BusinessID *string `json:"business_id,omitempty"`
	UserID     *string `json:"user_id,omitempty"`
	IsVerified bool    `json:"is_verified"`
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	Message    string  `json:"message"`
	Source     string  `json:"source"`
	Status     string  `json:"status"`
	IsSpam     bool    `json:"is_spam"`
	CreatedAt  string  `json:"created_at"` // RFC3339 timestamp
}

// LeadUpdatedPayload represents a lead.updated event.
type LeadUpdatedPayload struct {
	ID              string  `json:"id"`
	ListingID       string  `json:"listing_id"`
	Status          string  `json:"status"`
	AssignedTo      *string `json:"assigned_to,omitempty"`
	IsSpam          bool    `json:"is_spam"`
	FirstResponseAt *string `json:"first_response_at,omitempty"`
	UpdatedAt       string  `json:"updated_at"`
}
