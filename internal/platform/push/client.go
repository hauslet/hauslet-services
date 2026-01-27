package push

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Client is the high-level push notification service
type Client struct {
	provider Provider
	logger   *slog.Logger
}

// New creates a new push notification client
func New(provider Provider, logger *slog.Logger) *Client {
	return &Client{
		provider: provider,
		logger:   logger,
	}
}

// Send sends a push notification with automatic logging
func (c *Client) Send(ctx context.Context, req PushRequest) (*PushResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid push request: %w", err)
	}

	// Try provider
	startTime := time.Now()
	resp, err := c.provider.Send(ctx, req)

	if err == nil {
		resp.SentAt = time.Now()
		c.logger.Info("push notification sent successfully",
			"provider", c.provider.Provider(),
			"token_count", len(req.Tokens),
			"topic", req.Topic,
			"success_count", resp.SuccessCount,
			"failure_count", resp.FailureCount,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return resp, nil
	}

	// Log failure
	c.logger.Error("push notification failed",
		"provider", c.provider.Provider(),
		"error", err.Error(),
		"token_count", len(req.Tokens),
		"topic", req.Topic,
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return nil, err
}

// SendToToken sends a notification to a single device token (convenience method)
func (c *Client) SendToToken(ctx context.Context, token string, notification NotificationPayload, data map[string]string) (*PushResponse, error) {
	req := PushRequest{
		Tokens:       []string{token},
		Notification: notification,
		Data:         data,
	}
	return c.Send(ctx, req)
}

// SendToTokens sends a notification to multiple device tokens (convenience method)
func (c *Client) SendToTokens(ctx context.Context, tokens []string, notification NotificationPayload, data map[string]string) (*PushResponse, error) {
	req := PushRequest{
		Tokens:       tokens,
		Notification: notification,
		Data:         data,
	}
	return c.Send(ctx, req)
}

// SendToTopic sends a notification to all devices subscribed to a topic
func (c *Client) SendToTopic(ctx context.Context, topic string, notification NotificationPayload, data map[string]string) (*PushResponse, error) {
	startTime := time.Now()
	resp, err := c.provider.SendToTopic(ctx, topic, notification, data)

	if err != nil {
		c.logger.Error("topic notification failed",
			"provider", c.provider.Provider(),
			"topic", topic,
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return nil, err
	}

	resp.SentAt = time.Now()
	c.logger.Info("topic notification sent successfully",
		"provider", c.provider.Provider(),
		"topic", topic,
		"message_id", resp.MessageID,
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return resp, nil
}

// SubscribeToTopic subscribes device tokens to a topic
func (c *Client) SubscribeToTopic(ctx context.Context, tokens []string, topic string) error {
	req := TopicRequest{
		Topic:  topic,
		Tokens: tokens,
	}

	if err := req.Validate(); err != nil {
		return fmt.Errorf("invalid topic request: %w", err)
	}

	startTime := time.Now()
	err := c.provider.SubscribeToTopic(ctx, req)

	if err != nil {
		c.logger.Error("topic subscription failed",
			"provider", c.provider.Provider(),
			"topic", topic,
			"token_count", len(tokens),
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return err
	}

	c.logger.Info("topic subscription successful",
		"provider", c.provider.Provider(),
		"topic", topic,
		"token_count", len(tokens),
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return nil
}

// UnsubscribeFromTopic unsubscribes device tokens from a topic
func (c *Client) UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error {
	req := TopicRequest{
		Topic:  topic,
		Tokens: tokens,
	}

	if err := req.Validate(); err != nil {
		return fmt.Errorf("invalid topic request: %w", err)
	}

	startTime := time.Now()
	err := c.provider.UnsubscribeFromTopic(ctx, req)

	if err != nil {
		c.logger.Error("topic unsubscription failed",
			"provider", c.provider.Provider(),
			"topic", topic,
			"token_count", len(tokens),
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return err
	}

	c.logger.Info("topic unsubscription successful",
		"provider", c.provider.Provider(),
		"topic", topic,
		"token_count", len(tokens),
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return nil
}

// HealthCheck checks the health of the provider
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.provider.HealthCheck(ctx)
}
