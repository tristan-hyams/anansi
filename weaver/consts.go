package weaver

import "errors"

// errMaxDepth indicates a URL was skipped because it exceeded the configured max depth.
var errMaxDepth = errors.New("max depth exceeded")

const (
	defaultUserAgent = "Anansi"

	// defaultProgressInterval is the number of URLs a crawler processes
	// before logging a progress checkpoint. Keeps logs quiet during
	// normal operation but gives visibility on long crawls.
	defaultProgressInterval = 100

	logKeyURL       = "url"
	logKeyDepth     = "depth"
	logKeyLinks     = "links"
	logKeyCrawlerID = "crawler_id"
)
