// Package robots provides robots compliance for the Anansi web crawler.
//
// Two mechanisms are supported:
//   - robots.txt: fetched once at crawl start, checked before enqueueing URLs.
//   - X-Robots-Tag: checked per-response to respect nofollow directives.
//
// Graceful degradation: a missing robots.txt (404/403) or network error
// results in allow-all - the crawl continues with a log message.
package robots

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/temoto/robotstxt"

	"github.com/tristan-hyams/anansi/webutil"
)

// Rules wraps parsed robots.txt directives. A nil inner data
// (from 404, network error, or empty body) means allow all.
type Rules struct {
	data *robotstxt.RobotsData
}

// IsAllowed checks if path is permitted by the robots.txt rules
// for User-agent: *. Returns true if no rules are loaded (allow all).
func (r *Rules) IsAllowed(path string) bool {
	if r.data == nil {
		return true
	}

	group := r.data.FindGroup("*")
	if group == nil {
		return true
	}

	return group.Test(path)
}

// CrawlDelay returns the Crawl-delay directive for User-agent: *,
// or zero if not specified or no rules are loaded.
func (r *Rules) CrawlDelay() time.Duration {
	if r.data == nil {
		return 0
	}

	group := r.data.FindGroup("*")
	if group == nil {
		return 0
	}

	return group.CrawlDelay
}

// Fetch retrieves and parses robots.txt for the given base URL.
// Creates its own HTTP client backed by the shared transport singleton.
// Returns allow-all Rules on 404/403 or network error (with log message).
// Only returns an error on context cancellation or non-404/403 HTTP errors.
func Fetch(ctx context.Context, baseURL *url.URL, logger *slog.Logger) (*Rules, error) {

	client := webutil.NewClient(fetchTimeout)

	robotsURL := fmt.Sprintf("%s://%s/robots.txt", baseURL.Scheme, baseURL.Host)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building robots.txt request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("fetching robots.txt: %w", ctx.Err())
		}
		logger.Warn("robots.txt fetch failed, allowing all", logKeyURL, robotsURL, "error", err)
		return &Rules{}, nil
	}
	defer resp.Body.Close()

	return handleResponse(resp, robotsURL, logger)
}

// handleResponse processes the robots.txt HTTP response.
func handleResponse(resp *http.Response, robotsURL string, logger *slog.Logger) (*Rules, error) {

	if resp.StatusCode == http.StatusNotFound {
		logger.Info("robots.txt not found, no rules applied", logKeyURL, robotsURL)
		return &Rules{}, nil
	}

	// CDN and static hosting providers (e.g. AWS S3 behind CloudFront, Azure
	// Blob Storage, Google Cloud Storage) often return 401 or 403 for files
	// that don't exist in the bucket, rather than the expected 404. This
	// happens because the bucket doesn't allow public listing - the server
	// can't distinguish "file not found" from "you don't have access" without
	// revealing bucket contents, so it defaults to 403 Forbidden.
	//
	// Since we cannot reliably distinguish a genuine 403 (file exists but
	// access denied) from a missing-file 403, we treat it as not found.
	// A robots.txt we can't read shouldn't block the entire crawl.
	if resp.StatusCode == http.StatusForbidden {
		logger.Info("robots.txt returned 403, treating as not found (likely CDN/static host)",
			logKeyURL, robotsURL,
		)
		return &Rules{}, nil
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("robots.txt returned status %d for %s", resp.StatusCode, robotsURL)
	}

	return parseBody(resp, robotsURL, logger)
}

// parseBody reads and parses the robots.txt response body.
func parseBody(resp *http.Response, robotsURL string, logger *slog.Logger) (*Rules, error) {

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRobotsBodySize))
	if err != nil {
		logger.Warn("robots.txt read failed, allowing all", logKeyURL, robotsURL, "error", err)
		return &Rules{}, nil
	}

	data, err := robotstxt.FromBytes(body)
	if err != nil {
		logger.Warn("robots.txt parse failed, allowing all", logKeyURL, robotsURL, "error", err)
		return &Rules{}, nil
	}

	return &Rules{data: data}, nil
}
