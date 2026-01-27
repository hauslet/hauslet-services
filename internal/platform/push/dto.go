package push

import (
	"errors"
	"time"
)

// PushRequest contains the data for sending a push notification
type PushRequest struct {
	// Tokens are the device registration tokens to send to (required for direct send)
	Tokens []string `json:"tokens,omitempty"`

	// Topic is the FCM topic to send to (mutually exclusive with Tokens)
	Topic string `json:"topic,omitempty"`

	// Notification contains the visible notification content
	Notification NotificationPayload `json:"notification"`

	// Data contains custom key-value pairs for app-specific data
	Data map[string]string `json:"data,omitempty"`

	// Android-specific options
	AndroidPriority string `json:"android_priority,omitempty"` // "high" or "normal"

	// iOS-specific options
	IOSBadge *int   `json:"ios_badge,omitempty"`
	IOSSound string `json:"ios_sound,omitempty"`

	// TTL (Time To Live) in seconds for the notification
	TTL int `json:"ttl,omitempty"`
}

// NotificationPayload contains the visible notification content
type NotificationPayload struct {
	Title    string `json:"title"`               // Required
	Body     string `json:"body"`                // Required
	ImageURL string `json:"image_url,omitempty"` // Optional image URL
}

// Validate checks if the push request has required fields and valid format
func (r *PushRequest) Validate() error {
	// Must have either tokens or topic
	if len(r.Tokens) == 0 && r.Topic == "" {
		return errors.New("either tokens or topic must be provided")
	}

	// Cannot have both
	if len(r.Tokens) > 0 && r.Topic != "" {
		return errors.New("cannot specify both tokens and topic")
	}

	// Validate tokens
	if len(r.Tokens) > 500 {
		return errors.New("maximum 500 tokens per request")
	}

	for _, token := range r.Tokens {
		if token == "" {
			return errors.New("empty device token provided")
		}
	}

	// Validate notification content
	if r.Notification.Title == "" {
		return errors.New("notification title is required")
	}

	if r.Notification.Body == "" {
		return errors.New("notification body is required")
	}

	if len(r.Notification.Title) > 200 {
		return errors.New("notification title exceeds maximum length of 200 characters")
	}

	if len(r.Notification.Body) > 1000 {
		return errors.New("notification body exceeds maximum length of 1000 characters")
	}

	// Validate topic name if provided
	if r.Topic != "" && len(r.Topic) > 200 {
		return errors.New("topic name exceeds maximum length")
	}

	return nil
}

// PushResponse contains the result of sending a push notification
type PushResponse struct {
	Success      bool      `json:"success"`       // Overall success status
	MessageID    string    `json:"message_id"`    // Provider's message ID (for single sends)
	MessageIDs   []string  `json:"message_ids"`   // Provider's message IDs (for batch sends)
	SuccessCount int       `json:"success_count"` // Number of successful sends
	FailureCount int       `json:"failure_count"` // Number of failed sends
	Provider     string    `json:"provider"`      // Provider used (fcm, apns, etc.)
	SentAt       time.Time `json:"sent_at"`       // When notification was sent
	Errors       []string  `json:"errors"`        // Individual error messages for batch sends
}

// TopicRequest contains data for managing topic subscriptions
type TopicRequest struct {
	Topic  string   `json:"topic"`
	Tokens []string `json:"tokens"`
}

// Validate checks if the topic request is valid
func (r *TopicRequest) Validate() error {
	if r.Topic == "" {
		return errors.New("topic is required")
	}
	if len(r.Tokens) == 0 {
		return ErrNoTokensProvided
	}
	if len(r.Tokens) > 1000 {
		return errors.New("maximum 1000 tokens per topic subscription request")
	}
	return nil
}
