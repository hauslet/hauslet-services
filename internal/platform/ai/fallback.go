package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-pkgz/lgr"
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
	log       *lgr.Logger
}

// NewFallbackClient wraps two AI providers with fallback behavior.
func NewFallbackClient(primary, secondary AIClient, opts FallbackOptions, log *lgr.Logger) AIClient {
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
		f.log.Logf("[WARN] primary provider %s health check failed: %v", f.primary.Provider(), err)
		if f.opts.Enabled {
			f.log.Logf("[INFO] attempting health check on secondary provider %s", f.secondary.Provider())
			return f.secondary.HealthCheck(ctx)
		}
		return err
	}
	return nil
}

func (f *FallbackClient) Moderate(ctx context.Context, input AIModerationInput) (*AIModerationResult, error) {
	// Always try primary first.
	f.log.Logf("[INFO] attempting moderation with primary provider %s (target: %s, attempt: %d/%d)",
		f.primary.Provider(), input.ContentID, input.AttemptNumber, input.MaxAttempts)

	res, err := f.primary.Moderate(ctx, input)
	if err == nil {
		f.log.Logf("[INFO] primary provider %s succeeded for target %s", f.primary.Provider(), input.ContentID)
		return res, nil
	}

	f.log.Logf("[WARN] primary provider %s failed for target %s (attempt %d/%d): %v",
		f.primary.Provider(), input.ContentID, input.AttemptNumber, input.MaxAttempts, err)

	// Determine if fallback is allowed for this request.
	if !f.shouldFallback(input, err) {
		f.log.Logf("[INFO] fallback not triggered for target %s (enabled=%v, attempt=%d, threshold=%d)",
			input.ContentID, f.opts.Enabled, input.AttemptNumber, f.opts.AttemptThreshold)
		return nil, err
	}

	// Fallback to secondary.
	f.log.Logf("[INFO] triggering fallback to secondary provider %s for target %s",
		f.secondary.Provider(), input.ContentID)

	res2, err2 := f.secondary.Moderate(ctx, input)
	if err2 == nil {
		f.log.Logf("[INFO] secondary provider %s succeeded for target %s after primary failure",
			f.secondary.Provider(), input.ContentID)
		return res2, nil
	}

	f.log.Logf("[ERROR] both providers failed for target %s: primary(%s): %v; secondary(%s): %v",
		input.ContentID, f.primary.Provider(), err, f.secondary.Provider(), err2)

	// Return both errors for observability.
	return nil, fmt.Errorf("primary(%s) failed: %v; secondary(%s) failed: %w", f.primary.Provider(), err, f.secondary.Provider(), err2)
}

func (f *FallbackClient) shouldFallback(input AIModerationInput, primaryErr error) bool {
	if !f.opts.Enabled {
		f.log.Logf("[INFO] fallback disabled for target %s", input.ContentID)
		return false
	}

	// Never fallback for video.
	if input.Video != nil && input.Video.Key != "" {
		f.log.Logf("[INFO] fallback skipped for video content (target: %s)", input.ContentID)
		return false
	}

	// Must meet attempt threshold.
	if input.AttemptNumber < f.opts.AttemptThreshold {
		f.log.Logf("[INFO] fallback skipped: attempt %d below threshold %d (target: %s)",
			input.AttemptNumber, f.opts.AttemptThreshold, input.ContentID)
		return false
	}

	// Only fallback for overload-like errors or known model-not-found cases.
	if !isRetryableFallbackErr(primaryErr) {
		f.log.Logf("[INFO] fallback skipped: error not retryable (target: %s)", input.ContentID)
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
