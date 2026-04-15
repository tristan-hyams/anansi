package frontier

const (
	// defaultBufferSize is the queue capacity when the caller provides
	// a buffer size of zero or negative. The visited set is the real
	// bound — this just needs to be large enough to never block writers.
	defaultBufferSize = 100_000
)
