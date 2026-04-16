package benchmark

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"github.com/tristan-hyams/anansi/frontier"
)

func benchLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(
		&strings.Builder{},
		&slog.HandlerOptions{Level: slog.LevelError},
	))
}

// BenchmarkFrontierEnqueue measures the cost of Enqueue including
// sync.Map dedup and channel send. Each iteration uses a unique URL.
func BenchmarkFrontierEnqueue(b *testing.B) {
	ctx := context.Background()
	front := frontier.NewInMemory(b.N+1, benchLogger())

	urls := make([]*url.URL, b.N)
	for i := range urls {
		urls[i], _ = url.Parse(fmt.Sprintf("https://example.com/page/%d", i))
	}

	b.ResetTimer()
	for i := range b.N {
		front.Enqueue(ctx, &frontier.FrontierURL{URL: urls[i], Depth: 0})
	}
}

// BenchmarkFrontierDequeue measures Dequeue + Done throughput
// from a pre-filled queue.
func BenchmarkFrontierDequeue(b *testing.B) {
	ctx := context.Background()
	front := frontier.NewInMemory(b.N+1, benchLogger())

	for i := range b.N {
		u, _ := url.Parse(fmt.Sprintf("https://example.com/page/%d", i))
		front.Enqueue(ctx, &frontier.FrontierURL{URL: u, Depth: 0})
	}

	b.ResetTimer()
	for range b.N {
		_, _ = front.Dequeue(ctx)
		front.Done()
	}
}
