package frontier_test

import (
	"context"
	"log/slog"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tristan-hyams/anansi/frontier"
)

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func newFrontierURL(t *testing.T, raw string, depth int) *frontier.FrontierURL {
	t.Helper()
	return &frontier.FrontierURL{
		URL:   mustParse(t, raw),
		Depth: depth,
	}
}

func newTestFrontier(t *testing.T, bufferSize int) *frontier.InMemory {
	t.Helper()
	logger := slog.Default()
	return frontier.NewInMemory(bufferSize, logger)
}

// --- select {} behavior tests ---
// These test Go's select multiplexing directly using raw channels,
// no frontier code involved. They prove that a select with two
// non-deterministic channel cases picks up whichever fires first
// without getting stuck committed to one case.

// TestSelect_ContextCancelUnblocksBlockedChannel - a select blocks on
// a full channel send. A separate goroutine cancels the context after 2s.
// Proves ctx.Done() fires even while the channel case is blocking.
func TestSelect_ContextCancelUnblocksBlockedChannel(t *testing.T) {
	t.Parallel()

	ch := make(chan string, 1)
	ch <- "fills buffer" // full

	ctx, cancel := context.WithCancel(context.Background())

	result := make(chan string, 1)
	start := time.Now()

	go func() {
		t.Log("goroutine 1: select blocking on full channel...")
		select {
		case ch <- "blocked":
			result <- "channel"
		case <-ctx.Done():
			elapsed := time.Since(start)
			t.Logf("goroutine 1: ctx.Done() fired after %v", elapsed)
			result <- "context"
		}
	}()

	go func() {
		t.Log("goroutine 2: will cancel context in 2s...")
		time.Sleep(2 * time.Second)
		t.Log("goroutine 2: cancelling context now")
		cancel()
	}()

	winner := <-result
	elapsed := time.Since(start)
	t.Logf("select resolved via %q after %v", winner, elapsed)

	assert.Equal(t, "context", winner)
	assert.GreaterOrEqual(t, elapsed, 2*time.Second)
	assert.Less(t, elapsed, 3*time.Second)
}

// TestSelect_ChannelUnblocksWhileContextPending - a select blocks on
// a full channel send. A separate goroutine drains the channel after 2s.
// Context is never cancelled. Proves the channel case fires even while
// ctx.Done() is being watched.
func TestSelect_ChannelUnblocksWhileContextPending(t *testing.T) {
	t.Parallel()

	ch := make(chan string, 1)
	ch <- "fills buffer" // full

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := make(chan string, 1)
	start := time.Now()

	go func() {
		t.Log("goroutine 1: select blocking on full channel...")
		select {
		case ch <- "delivered":
			elapsed := time.Since(start)
			t.Logf("goroutine 1: channel send succeeded after %v", elapsed)
			result <- "channel"
		case <-ctx.Done():
			result <- "context"
		}
	}()

	go func() {
		t.Log("goroutine 2: will drain channel in 2s...")
		time.Sleep(2 * time.Second)
		val := <-ch
		t.Logf("goroutine 2: drained %q from channel", val)
	}()

	winner := <-result
	elapsed := time.Since(start)
	t.Logf("select resolved via %q after %v", winner, elapsed)

	assert.Equal(t, "channel", winner)
	assert.GreaterOrEqual(t, elapsed, 2*time.Second)
	assert.Less(t, elapsed, 3*time.Second)
}

// --- frontier unit tests ---

func TestEnqueueDequeue_FIFO(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := newTestFrontier(t, 10)

	urls := []string{
		"https://example.com/first",
		"https://example.com/second",
		"https://example.com/third",
	}

	for _, raw := range urls {
		require.NoError(t, f.Enqueue(ctx, newFrontierURL(t, raw, 0)))
	}

	for _, want := range urls {
		got, err := f.Dequeue(ctx)
		require.NoError(t, err)
		assert.Equal(t, want, got.URL.String())
	}
}

func TestEnqueue_Dedup(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := newTestFrontier(t, 10)

	require.NoError(t, f.Enqueue(ctx, newFrontierURL(t, "https://example.com/page", 0)))
	require.NoError(t, f.Enqueue(ctx, newFrontierURL(t, "https://example.com/page", 0))) // duplicate

	got, err := f.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/page", got.URL.String())

	// Channel should be empty - second enqueue was skipped.
	shortCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	_, err = f.Dequeue(shortCtx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestEnqueue_PreservesDepth(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := newTestFrontier(t, 10)

	require.NoError(t, f.Enqueue(ctx, newFrontierURL(t, "https://example.com/deep", 3)))

	got, err := f.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, got.Depth)
}

func TestDequeue_ContextCancellation(t *testing.T) {
	t.Parallel()
	f := newTestFrontier(t, 10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fu, err := f.Dequeue(ctx)
	assert.Nil(t, fu)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestEnqueue_ContextCancellation_FullBuffer(t *testing.T) {
	t.Parallel()
	f := newTestFrontier(t, 1)

	ctx := context.Background()
	require.NoError(t, f.Enqueue(ctx, newFrontierURL(t, "https://example.com/fills-buffer", 0)))

	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err := f.Enqueue(cancelledCtx, newFrontierURL(t, "https://example.com/blocked", 0))
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestConcurrentEnqueue(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := newTestFrontier(t, 100)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			fu := newFrontierURL(t, "https://example.com/"+string(rune('a'+i)), 0)
			_ = f.Enqueue(ctx, fu)
		})
	}
	wg.Wait()

	count := 0
	for {
		shortCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		_, err := f.Dequeue(shortCtx)
		cancel()
		if err != nil {
			break
		}
		count++
	}

	assert.Equal(t, 20, count)
}

func TestClear(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := newTestFrontier(t, 10)

	require.NoError(t, f.Enqueue(ctx, newFrontierURL(t, "https://example.com/a", 0)))
	require.NoError(t, f.Enqueue(ctx, newFrontierURL(t, "https://example.com/b", 0)))
	require.NoError(t, f.Enqueue(ctx, newFrontierURL(t, "https://example.com/c", 0)))

	f.Clear()

	// Queue should be empty.
	shortCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	_, err := f.Dequeue(shortCtx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Visited set should be reset - same URL can be enqueued again.
	require.NoError(t, f.Enqueue(ctx, newFrontierURL(t, "https://example.com/a", 0)))

	got, err := f.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/a", got.URL.String())
}

func TestNewInMemory_DefaultBufferSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		bufferSize int
	}{
		{"zero", 0},
		{"negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			f := frontier.NewInMemory(tt.bufferSize, slog.Default())

			require.NoError(t, f.Enqueue(ctx, newFrontierURL(t, "https://example.com/", 0)))

			got, err := f.Dequeue(ctx)
			require.NoError(t, err)
			assert.Equal(t, "https://example.com/", got.URL.String())
		})
	}
}
