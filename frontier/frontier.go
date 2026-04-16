// Package frontier provides the URL queue and visited tracking for the Anansi
// web crawler. The Frontier interface defines the contract; InMemory is the
// default implementation using a buffered channel and sync.Map.
//
// In a production system, swap InMemory for a Redis/RabbitMQ-backed
// implementation via the Frontier interface - no crawler changes needed.
package frontier

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
)

// Frontier is the URL queue + dedup layer for the crawler.
// Enqueue checks visited state internally - callers do not need to
// coordinate IsVisited + Enqueue separately.
type Frontier interface {
	// Enqueue adds a URL to the queue if it hasn't been visited.
	// Already-visited URLs are silently skipped (logged at debug level).
	Enqueue(ctx context.Context, fu *FrontierURL) error

	// Dequeue returns the next URL to process, blocking until one is
	// available or the context is cancelled.
	Dequeue(ctx context.Context) (*FrontierURL, error)

	// Done signals that a dequeued URL has been fully processed.
	// Decrements the pending work counter.
	Done()

	// Pending returns the number of URLs that have been enqueued but
	// not yet fully processed (enqueued - done).
	Pending() int32

	// IsDone returns true when all enqueued URLs have been fully processed.
	// Deterministic - no polling races. Pending reaches 0 only when every
	// URL that was ever enqueued has had Done() called.
	IsDone() bool

	// Len returns the number of URLs currently in the queue.
	Len() int

	// Clear drains the queue and resets the visited set.
	Clear()
}

// InMemory is a Frontier backed by a buffered channel (queue) and
// sync.Map (visited set). Safe for concurrent use by multiple goroutines.
type InMemory struct {
	queue   chan *FrontierURL
	visited sync.Map
	pending atomic.Int32
	logger  *slog.Logger
}

// NewInMemory creates an InMemory frontier with the given buffer size.
// If bufferSize is less than 1, it defaults to defaultBufferSize.
// The visited set is the real bound on growth - the buffer just needs
// to be large enough to avoid blocking writers.
func NewInMemory(bufferSize int, logger *slog.Logger) *InMemory {
	if bufferSize < 1 {
		bufferSize = defaultBufferSize
	}

	return &InMemory{
		queue:  make(chan *FrontierURL, bufferSize),
		logger: logger,
	}
}

// Enqueue adds fu to the queue if its URL hasn't been visited. Duplicate
// URLs are skipped and logged at debug level. Returns an error if the
// context is cancelled while waiting for buffer space.
func (f *InMemory) Enqueue(ctx context.Context, fu *FrontierURL) error {

	key := fu.URL.String()

	if _, loaded := f.visited.LoadOrStore(key, true); loaded {
		f.logger.Debug("url already visited, skipping", "url", key)
		return nil
	}

	select {
	case f.queue <- fu:
		f.pending.Add(1)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("enqueueing %s: %w", key, ctx.Err())
	}
}

// Dequeue returns the next FrontierURL from the queue, blocking until
// one is available or the context is cancelled.
func (f *InMemory) Dequeue(ctx context.Context) (*FrontierURL, error) {
	select {
	case fu := <-f.queue:
		return fu, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Done signals that a dequeued URL has been fully processed.
func (f *InMemory) Done() {
	f.pending.Add(-1)
}

// Pending returns the count of URLs enqueued but not yet fully processed.
func (f *InMemory) Pending() int32 {
	return f.pending.Load()
}

// IsDone returns true when all enqueued URLs have been fully processed
// and the queue is physically empty. Both conditions guard against edge
// cases - pending counter alone could mask a bug (e.g. double Done call).
func (f *InMemory) IsDone() bool {
	return f.pending.Load() <= 0 && len(f.queue) == 0
}

// Len returns the number of URLs currently in the queue.
func (f *InMemory) Len() int {
	return len(f.queue)
}

// Clear drains the queue and resets the visited set.
func (f *InMemory) Clear() {
	for {
		select {
		case <-f.queue:
		default:
			f.visited = sync.Map{}
			return
		}
	}
}
