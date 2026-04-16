package weaver

import (
	"errors"
	"time"
)

// errMaxDepth indicates a URL was skipped because it exceeded the configured max depth.
var errMaxDepth = errors.New("max depth exceeded")

const (
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

	// maxResponseBodySize caps the bytes read from any HTTP response.
	// Prevents memory exhaustion from misconfigured or malicious servers.
	// 10 MB is generous for HTML; non-HTML responses are not parsed anyway.
	maxResponseBodySize int64 = 10 << 20 // 10 MB

	// maxRedirects caps the HTTP redirect chain length per request.
	maxRedirects = 10

	// HTTP request headers. Accept-Encoding is intentionally omitted -
	// Go's transport handles gzip/deflate transparently when unset.
	defaultUserAgent     = "Anansi Weaver; Go 1.26;"
	acceptHeader         = "text/html, */*;q=0.8"
	acceptLanguageHeader = "en"
)
