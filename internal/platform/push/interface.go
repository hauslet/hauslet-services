package push

import "context"

// Provider defines the interface that all push notification providers must implement
type Provider interface {
	// Send sends a push notification to device token(s) or topic
	Send(ctx context.Context, req PushRequest) (*PushResponse, error)

	// SendToTopic sends a notification to all devices subscribed to a topic
	SendToTopic(ctx context.Context, topic string, notification NotificationPayload, data map[string]string) (*PushResponse, error)

	// SubscribeToTopic subscribes device tokens to a topic
	SubscribeToTopic(ctx context.Context, req TopicRequest) error

	// UnsubscribeFromTopic unsubscribes device tokens from a topic
	UnsubscribeFromTopic(ctx context.Context, req TopicRequest) error

	// HealthCheck verifies provider API is reachable
	HealthCheck(ctx context.Context) error

	// Provider returns the provider name
	Provider() string
}
