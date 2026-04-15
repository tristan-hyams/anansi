package weaver

import "time"

// Web is the crawl summary returned by Weaver.Weave().
// Represents the web that was woven — all pages discovered and their metadata.
type Web struct {
	Visited   int
	Skipped   int
	Duration  time.Duration
	OriginURL string
	Pages     []PageResult
}

// PageResult holds per-page crawl data.
type PageResult struct {
	URL      string
	Links    int
	Depth    int
	Duration time.Duration
	Status   int
	Error    error
}
