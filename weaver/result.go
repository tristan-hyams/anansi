package weaver

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// Web is the crawl summary returned by Weaver.Weave().
// Represents the web that was woven — all pages discovered and their metadata.
type Web struct {
	Visited   int
	Skipped   int
	Duration  time.Duration
	OriginURL string
	Pages     []PageResult
}

// String returns a formatted summary of the crawl results.
func (w *Web) String() string {
	var sb strings.Builder

	_, _ = fmt.Fprintf(&sb, "\nCrawl Results: %s\n", w.OriginURL)
	_, _ = fmt.Fprintln(&sb, strings.Repeat("=", summaryWidth))
	_, _ = fmt.Fprintf(&sb, "Pages crawled: %d\n", w.Visited)
	_, _ = fmt.Fprintf(&sb, "Pages skipped: %d\n", w.Skipped)
	_, _ = fmt.Fprintf(&sb, "Duration:      %s\n\n", w.Duration.Round(summaryDurationRound))

	// Sort by depth first, then alphabetically by URL within each depth.
	slices.SortFunc(w.Pages, func(a, b PageResult) int {
		if a.Depth != b.Depth {
			return a.Depth - b.Depth
		}
		if a.URL < b.URL {
			return -1
		}
		if a.URL > b.URL {
			return 1
		}
		return 0
	})

	for _, p := range w.Pages {
		if p.Error != nil {
			_, _ = fmt.Fprintf(&sb, "  %-50s  error: %v\n", p.URL, p.Error)
		} else {
			_, _ = fmt.Fprintf(&sb, "  %-50s  → %d links\n", p.URL, p.Links)
		}
	}

	return sb.String()
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
