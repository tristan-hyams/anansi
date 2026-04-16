package weaver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRetryConfig(maxRetries int) retryConfig {
	return retryConfig{
		maxRetries: maxRetries,
		baseDelay:  1 * time.Millisecond, // fast tests
		logger:     nil,
		label:      "test-op",
	}
}

func TestWithRetry_SucceedsFirstAttempt(t *testing.T) {
	t.Parallel()

	attempts := 0
	result, err := withRetry(context.Background(), testRetryConfig(2), func() (string, error, bool) {
		attempts++
		return "ok", nil, false
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", result)
	assert.Equal(t, 1, attempts)
}

func TestWithRetry_RetryableSucceedsOnRetry(t *testing.T) {
	t.Parallel()

	attempts := 0
	result, err := withRetry(context.Background(), testRetryConfig(2), func() (string, error, bool) {
		attempts++
		if attempts < 2 {
			return "", errors.New("transient"), true
		}
		return "recovered", nil, false
	})

	require.NoError(t, err)
	assert.Equal(t, "recovered", result)
	assert.Equal(t, 2, attempts)
}

func TestWithRetry_RetryableExhaustsAttempts(t *testing.T) {
	t.Parallel()

	attempts := 0
	result, err := withRetry(context.Background(), testRetryConfig(2), func() (string, error, bool) {
		attempts++
		return "", errors.New("always fails"), true
	})

	require.Error(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 3, attempts) // 1 initial + 2 retries
	assert.Contains(t, err.Error(), "3 attempts")
	assert.Contains(t, err.Error(), "always fails")
}

func TestWithRetry_NonRetryableReturnsImmediately(t *testing.T) {
	t.Parallel()

	attempts := 0
	result, err := withRetry(context.Background(), testRetryConfig(5), func() (string, error, bool) {
		attempts++
		return "", errors.New("permanent"), false
	})

	require.Error(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 1, attempts)
	assert.Contains(t, err.Error(), "permanent")
}

func TestWithRetry_ContextCancelledMidBackoff(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	_, err := withRetry(ctx, testRetryConfig(5), func() (string, error, bool) {
		attempts++
		cancel() // cancel during first failure's backoff
		return "", errors.New("transient"), true
	})

	require.Error(t, err)
	assert.Equal(t, 1, attempts)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestWithRetry_DisabledWithNegativeOne(t *testing.T) {
	t.Parallel()

	attempts := 0
	_, err := withRetry(context.Background(), testRetryConfig(-1), func() (string, error, bool) {
		attempts++
		return "", errors.New("fail"), true
	})

	require.Error(t, err)
	assert.Equal(t, 1, attempts) // max(0, 1) = 1 attempt
}

func TestWithRetry_ZeroRetriesSingleAttempt(t *testing.T) {
	t.Parallel()

	attempts := 0
	_, err := withRetry(context.Background(), testRetryConfig(0), func() (string, error, bool) {
		attempts++
		return "", errors.New("fail"), true
	})

	require.Error(t, err)
	assert.Equal(t, 1, attempts) // max(1, 1) = 1 attempt
}
