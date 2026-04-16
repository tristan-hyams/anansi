package weaver

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// retryConfig holds parameters for withRetry.
type retryConfig struct {
	maxRetries int
	baseDelay  time.Duration
	logger     *slog.Logger
	label      string // identifies the operation in logs (e.g. URL string)
}

// withRetry executes fn up to maxRetries+1 times with exponential backoff.
// fn returns a result, an error, and whether the error is retryable.
// On non-retryable errors or context cancellation, returns immediately.
func withRetry[T any](
	ctx context.Context, cfg retryConfig,
	fn func() (T, error, bool),
) (T, error) {
	maxAttempts := max(cfg.maxRetries+1, 1)

	var zero T
	var lastErr error

	for attempt := range maxAttempts {
		result, err, retryable := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !retryable || ctx.Err() != nil || attempt == maxAttempts-1 {
			break
		}

		delay := cfg.baseDelay << attempt
		if cfg.logger != nil {
			cfg.logger.Warn("retrying transient error",
				"label", cfg.label,
				"attempt", attempt+1,
				"delay", delay,
				logKeyError, lastErr,
			)
		}

		// time.After is acceptable here — retry loops are short-lived (≤3 iterations).
		// For long-lived loops, prefer time.NewTicker (see monitorCompletion).
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}

	// Context cancellation takes precedence over the last transient error.
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}

	return zero, fmt.Errorf("%s after %d attempts: %w",
		cfg.label, maxAttempts, lastErr)
}
