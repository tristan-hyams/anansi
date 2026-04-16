package weaver

import (
	"errors"
	"time"
)

// errMaxDepth indicates a URL was skipped because it exceeded the configured max depth.
var errMaxDepth = errors.New("max depth exceeded")

const (
	defaultUserAgent = "Anansi"

	// defaultProgressInterval is the number of URLs a crawler processes
	// before logging a progress checkpoint. Keeps logs quiet during
	// normal operation but gives visibility on long crawls.
	defaultProgressInterval = 100

	// Retry defaults for transient HTTP errors (connection reset, 5xx, timeout).
	defaultMaxRetries = 2
	baseRetryDelay    = 500 * time.Millisecond

	logKeyURL       = "url"
	logKeyDepth     = "depth"
	logKeyLinks     = "links"
	logKeyCrawlerID = "crawler_id"
	logKeyError     = "error"

	// monitorInterval is how often the completion monitor checks the
	// frontier's pending counter. Uses a ticker to avoid timer leaks.
	monitorInterval = 50 * time.Millisecond

	// serverErrorThreshold is the HTTP status code at and above which
	// responses are considered transient server errors eligible for retry.
	serverErrorThreshold = 500
)
