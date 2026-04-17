package weaver

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
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

	var zero T
	var lastErr error
	maxAttempts := max(cfg.maxRetries+1, 1)

	for attempt := range maxAttempts {
		result, err, retryable := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !retryable || ctx.Err() != nil || attempt == maxAttempts-1 {
			break
		}

		// Exponential backoff with small additive jitter.
		// The exponential curve dominates — dashboards see predictable
		// 500ms/1s/2s/... behaviour. The 50-200ms jitter is just enough
		// to break synchronisation between crawlers without distorting
		// the backoff shape.
		backoff := cfg.baseDelay * time.Duration(math.Pow(2, float64(attempt)))
		jitter := jitterMinMs + rand.Int64N(jitterRangeMs)
		delay := backoff + time.Duration(jitter)*time.Millisecond

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
