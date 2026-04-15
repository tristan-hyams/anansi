package weaver

import "time"

const (
	defaultUserAgent  = "Anansi"
	defaultBufferSize = 0 // use frontier's default

	logKeyURL   = "url"
	logKeyDepth = "depth"
	logKeyLinks = "links"

	summaryWidth         = 40
	summaryDurationRound = 100 * time.Millisecond
)
