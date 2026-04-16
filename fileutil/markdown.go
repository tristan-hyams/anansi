package fileutil

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/tristan-hyams/anansi/weaver"
)

// RenderMarkdown returns a formatted summary of the crawl results, including
// a flat page list sorted by depth and a directory tree sitemap.
func RenderMarkdown(web *weaver.Web) string {
	var sb strings.Builder

	_, _ = fmt.Fprint(&sb, banner)
	_, _ = fmt.Fprintf(&sb, "  A N A N S I - %s\n\n", web.OriginURL)
	_, _ = fmt.Fprintln(&sb, strings.Repeat("=", summaryWidth))
	_, _ = fmt.Fprintf(&sb, "Pages crawled: %d\n", web.Visited)
	_, _ = fmt.Fprintf(&sb, "Pages skipped: %d\n", web.Skipped)
	_, _ = fmt.Fprintf(&sb, "Duration:      %s\n\n", web.Duration.Round(summaryDurationRound))

	stats := ComputeStats(web)
	writeStats(&sb, stats)

	// Sort by depth first, then alphabetically by URL within each depth.
	slices.SortFunc(web.Pages, func(a, b weaver.PageResult) int {
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

	// Flat page list - only crawled pages, not skipped.
	for _, p := range web.Pages {
		if p.Error != nil {
			continue
		}
		_, _ = fmt.Fprintf(&sb, "  %-50s  → %d links\n", p.URL, p.Links)
	}

	_, _ = fmt.Fprint(&sb, "\nSitemap:\n")
	_, _ = fmt.Fprintln(&sb, strings.Repeat("-", summaryWidth))
	writeTree(&sb, web)

	return sb.String()
}

// writeStats writes the statistics section to the summary.
func writeStats(sb *strings.Builder, stats *Stats) {
	_, _ = fmt.Fprint(sb, "Response Latency:\n")
	_, _ = fmt.Fprintf(sb, "  Avg: %-10s P50: %-10s P95: %-10s P99: %s\n",
		stats.Latency.Avg.Round(time.Millisecond),
		stats.Latency.P50.Round(time.Millisecond),
		stats.Latency.P95.Round(time.Millisecond),
		stats.Latency.P99.Round(time.Millisecond),
	)
	_, _ = fmt.Fprintf(sb, "  Min: %-10s Max: %s\n\n",
		stats.Latency.Min.Round(time.Millisecond),
		stats.Latency.Max.Round(time.Millisecond),
	)

	if len(stats.StatusCodes) > 0 {
		_, _ = fmt.Fprint(sb, "Status Codes:\n")
		codes := make([]int, 0, len(stats.StatusCodes))
		for code := range stats.StatusCodes {
			codes = append(codes, code)
		}
		slices.Sort(codes)
		for _, code := range codes {
			_, _ = fmt.Fprintf(sb, "  %d: %d\n", code, stats.StatusCodes[code])
		}
		_, _ = fmt.Fprint(sb, "\n")
	}

	if len(stats.ContentTypes) > 0 {
		_, _ = fmt.Fprint(sb, "Content Types:\n")
		types := make([]string, 0, len(stats.ContentTypes))
		for ct := range stats.ContentTypes {
			types = append(types, ct)
		}
		slices.Sort(types)
		for _, ct := range types {
			_, _ = fmt.Fprintf(sb, "  %-40s %d\n", ct, stats.ContentTypes[ct])
		}
		_, _ = fmt.Fprint(sb, "\n")
	}
}

// writeTree builds a directory tree from crawled page URLs and writes
// it to the string builder. Skipped pages are excluded.
func writeTree(sb *strings.Builder, web *weaver.Web) {
	root := &treeNode{name: pathSeparator, children: make(map[string]*treeNode)}

	for _, p := range web.Pages {
		if p.Error != nil {
			continue
		}

		parsed, err := url.Parse(p.URL)
		if err != nil {
			continue
		}

		path := strings.TrimPrefix(parsed.Path, pathSeparator)
		if path == "" {
			root.isPage = true
			continue
		}

		segments := strings.Split(path, pathSeparator)
		node := root
		for _, seg := range segments {
			if seg == "" {
				continue
			}
			child, exists := node.children[seg]
			if !exists {
				child = &treeNode{name: seg, children: make(map[string]*treeNode)}
				node.children[seg] = child
			}
			node = child
		}
		node.isPage = true
	}

	printTree(sb, root, "", "")
}

// treeNode represents a directory or page in the sitemap tree.
type treeNode struct {
	name     string
	children map[string]*treeNode
	isPage   bool
}

// printTree recursively writes the tree with box-drawing characters.
func printTree(sb *strings.Builder, node *treeNode, prefix string, connector string) {
	label := node.name
	if !node.isPage && len(node.children) > 0 {
		label = node.name + pathSeparator
	}

	_, _ = fmt.Fprintf(sb, "%s%s\n", connector, label)

	keys := make([]string, 0, len(node.children))
	for k := range node.children {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for i, key := range keys {
		child := node.children[key]
		isLast := i == len(keys)-1

		var childConnector, childPrefix string
		if isLast {
			childConnector = prefix + "└── "
			childPrefix = prefix + "    "
		} else {
			childConnector = prefix + "├── "
			childPrefix = prefix + "│   "
		}

		printTree(sb, child, childPrefix, childConnector)
	}
}

// RenderErrorLog returns a formatted error report of all failed URLs,
// grouped by error reason. Returns empty string if no errors occurred.
func RenderErrorLog(web *weaver.Web) string {
	type errorEntry struct {
		url       string
		timestamp time.Time
	}

	// Group by error reason.
	groups := make(map[string][]errorEntry)

	for _, p := range web.Pages {
		if p.Error == nil {
			continue
		}
		reason := p.Error.Error()
		groups[reason] = append(groups[reason], errorEntry{url: p.URL, timestamp: p.Timestamp})
	}

	if len(groups) == 0 {
		return ""
	}

	var sb strings.Builder

	_, _ = fmt.Fprintf(&sb, "Crawl Errors: %s\n", web.OriginURL)
	_, _ = fmt.Fprintln(&sb, strings.Repeat("=", summaryWidth))
	_, _ = fmt.Fprintf(&sb, "Total errors: %d\n\n", web.Skipped)

	// Sort error reasons for consistent output.
	reasons := make([]string, 0, len(groups))
	for reason := range groups {
		reasons = append(reasons, reason)
	}
	slices.Sort(reasons)

	for _, reason := range reasons {
		entries := groups[reason]
		slices.SortFunc(entries, func(a, b errorEntry) int {
			if a.url < b.url {
				return -1
			}
			if a.url > b.url {
				return 1
			}
			return 0
		})

		_, _ = fmt.Fprintf(&sb, "%s (%d)\n", reason, len(entries))
		for _, e := range entries {
			_, _ = fmt.Fprintf(&sb, "  [%s] %s\n", e.timestamp.Format(time.RFC3339), e.url)
		}
		_, _ = fmt.Fprint(&sb, "\n")
	}

	return sb.String()
}
