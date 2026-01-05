package sms

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Client is the high-level SMS service with automatic fallback
type Client struct {
	primary  Provider
	fallback Provider
	logger   *slog.Logger
}

// New creates a new SMS client with primary and fallback providers
func New(primary, fallback Provider, logger *slog.Logger) *Client {
	return &Client{
		primary:  primary,
		fallback: fallback,
		logger:   logger,
	}
}

// Send sends an SMS with automatic fallback on failure
func (c *Client) Send(ctx context.Context, req SMSRequest) (*SMSResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid sms request: %w", err)
	}

	// Try primary provider first
	startTime := time.Now()
	resp, err := c.primary.Send(ctx, req)

	if err == nil {
		resp.SentAt = time.Now()
		c.logger.Info("sms sent successfully via primary provider",
			"provider", c.primary.Provider(),
			"to", req.To,
			"message_id", resp.MessageID,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return resp, nil
	}

	// Log primary failure
	c.logger.Warn("primary sms provider failed, trying fallback",
		"provider", c.primary.Provider(),
		"error", err.Error(),
		"to", req.To,
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	// Try fallback provider
	startTime = time.Now()
	resp, fallbackErr := c.fallback.Send(ctx, req)

	if fallbackErr == nil {
		resp.SentAt = time.Now()
		c.logger.Info("sms sent successfully via fallback provider",
			"provider", c.fallback.Provider(),
			"to", req.To,
			"message_id", resp.MessageID,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return resp, nil
	}

	// Both providers failed
	c.logger.Error("all sms providers failed",
		"primary_provider", c.primary.Provider(),
		"primary_error", err.Error(),
		"fallback_provider", c.fallback.Provider(),
		"fallback_error", fallbackErr.Error(),
		"to", req.To,
	)

	return nil, fmt.Errorf("%w: primary=%s (%v), fallback=%s (%v)",
		ErrAllProvidersFailed,
		c.primary.Provider(), err,
		c.fallback.Provider(), fallbackErr,
	)
}

// SendWithProvider sends an SMS using a specific provider (no fallback)
func (c *Client) SendWithProvider(ctx context.Context, providerName string, req SMSRequest) (*SMSResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid sms request: %w", err)
	}

	var provider Provider
	switch providerName {
	case c.primary.Provider():
		provider = c.primary
	case c.fallback.Provider():
		provider = c.fallback
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	startTime := time.Now()
	resp, err := provider.Send(ctx, req)

	if err != nil {
		c.logger.Error("sms send failed",
			"provider", providerName,
			"error", err.Error(),
			"to", req.To,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return nil, err
	}

	resp.SentAt = time.Now()
	c.logger.Info("sms sent successfully",
		"provider", providerName,
		"to", req.To,
		"message_id", resp.MessageID,
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return resp, nil
}

// HealthCheck checks the health of all providers
func (c *Client) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)

	results[c.primary.Provider()] = c.primary.HealthCheck(ctx)
	results[c.fallback.Provider()] = c.fallback.HealthCheck(ctx)

	return results
}
