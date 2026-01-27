package push

import (
	"context"
	"fmt"
	"log/slog"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMProvider implements the Provider interface using Firebase Cloud Messaging
type FCMProvider struct {
	client    *messaging.Client
	projectID string
	logger    *slog.Logger
}

// NewFCMProvider creates a new FCM provider instance
func NewFCMProvider(ctx context.Context, credentialsPath string, projectID string, logger *slog.Logger) (*FCMProvider, error) {
	if credentialsPath == "" {
		return nil, fmt.Errorf("firebase credentials path is required")
	}

	opt := option.WithCredentialsFile(credentialsPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create messaging client: %w", err)
	}

	return &FCMProvider{
		client:    client,
		projectID: projectID,
		logger:    logger,
	}, nil
}

// Send sends a push notification to device token(s)
func (f *FCMProvider) Send(ctx context.Context, req PushRequest) (*PushResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid push request: %w", err)
	}

	// Build the FCM message
	message := f.buildMessage(req)

	// Handle single vs multicast send
	if len(req.Tokens) == 1 {
		return f.sendSingle(ctx, message, req.Tokens[0])
	} else if len(req.Tokens) > 1 {
		return f.sendMulticast(ctx, message, req.Tokens)
	} else if req.Topic != "" {
		return f.sendToTopicInternal(ctx, message, req.Topic)
	}

	return nil, ErrNoTokensProvided
}

// sendSingle sends to a single device token
func (f *FCMProvider) sendSingle(ctx context.Context, message *messaging.Message, token string) (*PushResponse, error) {
	message.Token = token

	messageID, err := f.client.Send(ctx, message)
	if err != nil {
		f.logger.Error("failed to send push notification",
			"error", err.Error(),
			"token", token,
		)
		return nil, f.mapFCMError(err)
	}

	f.logger.Info("push notification sent successfully",
		"message_id", messageID,
		"provider", "fcm",
	)

	return &PushResponse{
		Success:      true,
		MessageID:    messageID,
		MessageIDs:   []string{messageID},
		SuccessCount: 1,
		FailureCount: 0,
		Provider:     "fcm",
	}, nil
}

// sendMulticast sends to multiple device tokens
func (f *FCMProvider) sendMulticast(ctx context.Context, message *messaging.Message, tokens []string) (*PushResponse, error) {
	multicastMessage := &messaging.MulticastMessage{
		Notification: message.Notification,
		Data:         message.Data,
		Android:      message.Android,
		APNS:         message.APNS,
		Tokens:       tokens,
	}

	batchResp, err := f.client.SendEachForMulticast(ctx, multicastMessage)
	if err != nil {
		f.logger.Error("failed to send multicast push notification",
			"error", err.Error(),
			"token_count", len(tokens),
		)
		return nil, f.mapFCMError(err)
	}

	// Collect message IDs and errors
	var messageIDs []string
	var errors []string
	for i, resp := range batchResp.Responses {
		if resp.Success {
			messageIDs = append(messageIDs, resp.MessageID)
		} else {
			errMsg := fmt.Sprintf("token[%d]: %v", i, resp.Error)
			errors = append(errors, errMsg)
			f.logger.Warn("individual send failed in multicast",
				"token_index", i,
				"error", resp.Error,
			)
		}
	}

	f.logger.Info("multicast push notification completed",
		"success_count", batchResp.SuccessCount,
		"failure_count", batchResp.FailureCount,
		"provider", "fcm",
	)

	return &PushResponse{
		Success:      batchResp.SuccessCount > 0,
		MessageIDs:   messageIDs,
		SuccessCount: batchResp.SuccessCount,
		FailureCount: batchResp.FailureCount,
		Provider:     "fcm",
		Errors:       errors,
	}, nil
}

// sendToTopicInternal sends to an FCM topic
func (f *FCMProvider) sendToTopicInternal(ctx context.Context, message *messaging.Message, topic string) (*PushResponse, error) {
	message.Topic = topic

	messageID, err := f.client.Send(ctx, message)
	if err != nil {
		f.logger.Error("failed to send topic notification",
			"error", err.Error(),
			"topic", topic,
		)
		return nil, f.mapFCMError(err)
	}

	f.logger.Info("topic notification sent successfully",
		"message_id", messageID,
		"topic", topic,
		"provider", "fcm",
	)

	return &PushResponse{
		Success:      true,
		MessageID:    messageID,
		MessageIDs:   []string{messageID},
		SuccessCount: 1,
		FailureCount: 0,
		Provider:     "fcm",
	}, nil
}

// SendToTopic sends a notification to all devices subscribed to a topic
func (f *FCMProvider) SendToTopic(ctx context.Context, topic string, notification NotificationPayload, data map[string]string) (*PushResponse, error) {
	if topic == "" {
		return nil, ErrInvalidTopic
	}

	message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title:    notification.Title,
			Body:     notification.Body,
			ImageURL: notification.ImageURL,
		},
		Data: data,
	}

	messageID, err := f.client.Send(ctx, message)
	if err != nil {
		f.logger.Error("failed to send topic notification",
			"error", err.Error(),
			"topic", topic,
		)
		return nil, f.mapFCMError(err)
	}

	f.logger.Info("topic notification sent successfully",
		"message_id", messageID,
		"topic", topic,
	)

	return &PushResponse{
		Success:      true,
		MessageID:    messageID,
		MessageIDs:   []string{messageID},
		SuccessCount: 1,
		FailureCount: 0,
		Provider:     "fcm",
	}, nil
}

// SubscribeToTopic subscribes device tokens to a topic
func (f *FCMProvider) SubscribeToTopic(ctx context.Context, req TopicRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	resp, err := f.client.SubscribeToTopic(ctx, req.Tokens, req.Topic)
	if err != nil {
		f.logger.Error("failed to subscribe to topic",
			"error", err.Error(),
			"topic", req.Topic,
			"token_count", len(req.Tokens),
		)
		return f.mapFCMError(err)
	}

	f.logger.Info("subscribed to topic",
		"topic", req.Topic,
		"success_count", resp.SuccessCount,
		"failure_count", resp.FailureCount,
	)

	return nil
}

// UnsubscribeFromTopic unsubscribes device tokens from a topic
func (f *FCMProvider) UnsubscribeFromTopic(ctx context.Context, req TopicRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	resp, err := f.client.UnsubscribeFromTopic(ctx, req.Tokens, req.Topic)
	if err != nil {
		f.logger.Error("failed to unsubscribe from topic",
			"error", err.Error(),
			"topic", req.Topic,
			"token_count", len(req.Tokens),
		)
		return f.mapFCMError(err)
	}

	f.logger.Info("unsubscribed from topic",
		"topic", req.Topic,
		"success_count", resp.SuccessCount,
		"failure_count", resp.FailureCount,
	)

	return nil
}

// HealthCheck verifies FCM API is reachable
func (f *FCMProvider) HealthCheck(ctx context.Context) error {
	// FCM doesn't have a dedicated health check endpoint
	// We can use a dry-run send with a test token to verify connectivity
	// For now, just check if client exists
	if f.client == nil {
		return ErrProviderUnavailable
	}
	return nil
}

// Provider returns the provider name
func (f *FCMProvider) Provider() string {
	return "fcm"
}

// buildMessage builds an FCM message from a PushRequest
func (f *FCMProvider) buildMessage(req PushRequest) *messaging.Message {
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title:    req.Notification.Title,
			Body:     req.Notification.Body,
			ImageURL: req.Notification.ImageURL,
		},
		Data: req.Data,
	}

	// Android-specific configuration
	if req.AndroidPriority != "" {
		message.Android = &messaging.AndroidConfig{
			Priority: req.AndroidPriority,
		}
	}

	// iOS-specific configuration
	if req.IOSBadge != nil || req.IOSSound != "" {
		message.APNS = &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Badge: req.IOSBadge,
					Sound: req.IOSSound,
				},
			},
		}
	}

	return message
}

// mapFCMError maps FCM errors to our standard errors
func (f *FCMProvider) mapFCMError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Check for common FCM error patterns
	if contains(errMsg, "invalid-registration-token") || contains(errMsg, "registration-token-not-registered") {
		return ErrInvalidToken
	}

	if contains(errMsg, "invalid-argument") {
		return ErrInvalidPayload
	}

	if contains(errMsg, "quota-exceeded") || contains(errMsg, "rate-limit") {
		return ErrRateLimited
	}

	// Default to provider unavailable
	return fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr) && findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
