package aimoderation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// FallbackOptions controls when and how fallback is triggered.
type FallbackOptions struct {
	Enabled          bool
	AttemptThreshold int // minimum AttemptNumber to allow fallback (e.g. 3 => fallback on 3rd attempt)
}

// FallbackClient tries the primary provider first and, when allowed, falls back to a secondary provider.
type FallbackClient struct {
	primary   AIClient
	secondary AIClient
	opts      FallbackOptions
	log       *slog.Logger
}

// NewFallbackClient wraps two AI providers with fallback behavior.
func NewFallbackClient(primary, secondary AIClient, opts FallbackOptions, log *slog.Logger) AIClient {
	// Enforce sane defaults
	if opts.AttemptThreshold <= 0 {
		opts.AttemptThreshold = 3
	}
	return &FallbackClient{
		primary:   primary,
		secondary: secondary,
		log:       log,
		opts:      opts,
	}
}

func (f *FallbackClient) Provider() string {
	return fmt.Sprintf("%s->%s", f.primary.Provider(), f.secondary.Provider())
}

// HealthCheck proxies to the primary provider; if fallback is enabled and primary fails, try secondary.
func (f *FallbackClient) HealthCheck(ctx context.Context) error {
	if err := f.primary.HealthCheck(ctx); err != nil {
		f.log.Warn("primary provider health check failed", "provider", f.primary.Provider(), "error", err)
		if f.opts.Enabled {
			f.log.Info("attempting health check on secondary provider", "provider", f.secondary.Provider())
			return f.secondary.HealthCheck(ctx)
		}
		return err
	}
	return nil
}

func (f *FallbackClient) Moderate(ctx context.Context, input AIModerationInput) (*AIModerationResult, error) {
	// Always try primary first.
	f.log.Info("attempting moderation with primary provider",
		"provider", f.primary.Provider(), "target", input.ContentID, "attempt", input.AttemptNumber, "max_attempts", input.MaxAttempts)

	res, err := f.primary.Moderate(ctx, input)
	if err == nil {
		f.log.Info("primary provider succeeded",
			"provider", f.primary.Provider(), "target", input.ContentID)
		return res, nil
	}

	f.log.Warn("primary provider failed",
		"provider", f.primary.Provider(), "target", input.ContentID, "attempt", input.AttemptNumber, "max_attempts", input.MaxAttempts, "error", err)

	// Determine if fallback is allowed for this request.
	if !f.shouldFallback(input, err) {
		f.log.Info("fallback not triggered",
			"target", input.ContentID, "enabled", f.opts.Enabled, "attempt", input.AttemptNumber, "threshold", f.opts.AttemptThreshold)
		return nil, err
	}

	// Fallback to secondary.
	f.log.Info("triggering fallback to secondary provider",
		"provider", f.secondary.Provider(), "target", input.ContentID)

	res2, err2 := f.secondary.Moderate(ctx, input)
	if err2 == nil {
		f.log.Info("secondary provider succeeded after primary failure",
			"provider", f.secondary.Provider(), "target", input.ContentID)
		return res2, nil
	}

	f.log.Error("both providers failed",
		"target", input.ContentID, "primary_provider", f.primary.Provider(), "primary_error", err,
		"secondary_provider", f.secondary.Provider(), "secondary_error", err2)

	// Return both errors for observability.
	return nil, fmt.Errorf("primary(%s) failed: %v; secondary(%s) failed: %w", f.primary.Provider(), err, f.secondary.Provider(), err2)
}

func (f *FallbackClient) shouldFallback(input AIModerationInput, primaryErr error) bool {
	if !f.opts.Enabled {
		f.log.Info("fallback disabled",
			"target", input.ContentID)
		return false
	}

	// Never fallback for video.
	if input.Video != nil && input.Video.Key != "" {
		f.log.Info("fallback skipped for video content",
			"target", input.ContentID)
		return false
	}

	// Must meet attempt threshold.
	if input.AttemptNumber < f.opts.AttemptThreshold {
		f.log.Info("fallback skipped: attempt below threshold",
			"attempt", input.AttemptNumber, "threshold", f.opts.AttemptThreshold, "target", input.ContentID)
		return false
	}

	// Only fallback for overload-like errors or known model-not-found cases.
	if !isRetryableFallbackErr(primaryErr) {
		f.log.Info("fallback skipped: error not retryable",
			"target", input.ContentID)
		return false
	}

	return true
}

// isRetryableFallbackErr attempts to detect provider errors that should trigger fallback.
func isRetryableFallbackErr(err error) bool {
	if err == nil {
		return false
	}
	// Common signals: HTTP 503, "overload", "unavailable", "not found"/404 for model issues.
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "503") ||
		strings.Contains(msg, "overload") ||
		strings.Contains(msg, "unavailable") ||
		strings.Contains(msg, "404") ||
		strings.Contains(msg, "not found") {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded)
}
