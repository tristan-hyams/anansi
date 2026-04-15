package weaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tristan-hyams/anansi/frontier"
	"github.com/tristan-hyams/anansi/normalizer"
	"github.com/tristan-hyams/anansi/parser"
	"github.com/tristan-hyams/anansi/robots"
)

// Crawler is a single worker that fetches and processes pages.
// Each Crawler runs in its own goroutine, managed by the Weaver.
// Has its own http.Client backed by the Weaver's shared Transport.
type Crawler struct {
	weaver *Weaver
	client *http.Client
}

// crawl is the main loop — dequeue URLs, process them, repeat until context is cancelled.
func (c *Crawler) crawl(ctx context.Context) {
	for {
		fu, err := c.weaver.front.Dequeue(ctx)
		if err != nil {
			return // context cancelled
		}

		c.weaver.active.Add(1)
		c.processURL(ctx, fu)
		c.weaver.active.Add(-1)
	}
}

// processURL handles a single URL: fetch, check headers, parse, enqueue new links.
func (c *Crawler) processURL(ctx context.Context, fu *frontier.FrontierURL) {

	w := c.weaver
	start := time.Now()
	pageURL := fu.URL.String()

	if w.cfg.MaxDepth > 0 && fu.Depth >= w.cfg.MaxDepth {
		w.logger.Debug("max depth reached, skipping", logKeyURL, pageURL, logKeyDepth, fu.Depth)
		return
	}

	if err := w.limiter.Wait(ctx); err != nil {
		return
	}

	resp, err := c.fetchPage(ctx, fu.URL)
	if err != nil {
		w.logger.Warn("fetch failed", logKeyURL, pageURL, "error", err)
		w.recordPage(PageResult{URL: pageURL, Depth: fu.Depth, Duration: time.Since(start), Error: err})
		return
	}
	defer resp.Body.Close()

	c.handleResponse(ctx, fu, resp, start)
}

// handleResponse processes a successful HTTP response.
func (c *Crawler) handleResponse(ctx context.Context, fu *frontier.FrontierURL, resp *http.Response, start time.Time) {

	w := c.weaver
	pageURL := fu.URL.String()

	w.logger.Debug("fetched", logKeyURL, pageURL, "status", resp.StatusCode)

	if !isHTML(resp) {
		w.logger.Debug("non-HTML content, skipping parse",
			logKeyURL, pageURL,
			"content_type", resp.Header.Get("Content-Type"),
		)
		w.recordPage(PageResult{URL: pageURL, Depth: fu.Depth, Status: resp.StatusCode, Duration: time.Since(start)})
		return
	}

	directives := robots.ParseXRobotsTag(resp.Header)
	if !directives.ShouldFollow() {
		w.logger.Info("X-Robots-Tag nofollow, skipping link extraction", logKeyURL, pageURL)
		w.recordPage(PageResult{URL: pageURL, Depth: fu.Depth, Status: resp.StatusCode, Duration: time.Since(start)})
		return
	}

	links, err := parser.ExtractLinks(ctx, resp.Body)
	if err != nil {
		w.logger.Warn("link extraction failed", logKeyURL, pageURL, "error", err)
	}

	w.logger.Info("crawled",
		logKeyURL, pageURL,
		logKeyDepth, fu.Depth,
		logKeyLinks, len(links),
		"duration", time.Since(start),
	)

	w.recordPage(PageResult{
		URL: pageURL, Links: len(links), Depth: fu.Depth,
		Status: resp.StatusCode, Duration: time.Since(start),
	})

	c.enqueueLinks(ctx, fu, links)
}

// enqueueLinks normalizes, filters, and enqueues discovered hrefs.
func (c *Crawler) enqueueLinks(ctx context.Context, parent *frontier.FrontierURL, hrefs []string) {
	w := c.weaver

	for _, raw := range hrefs {
		normalized, err := normalizer.Normalize(parent.URL, raw)
		if err != nil {
			w.logger.Debug("normalize failed, skipping", "href", raw, "error", err)
			continue
		}

		if !normalizer.IsFollowableScheme(normalized) {
			continue
		}

		if !normalizer.IsSameHost(w.origin, normalized) {
			continue
		}

		if !w.rules.IsAllowed(normalized.Path) {
			w.logger.Debug("robots.txt disallowed", logKeyURL, normalized.String())
			continue
		}

		if err = w.front.Enqueue(ctx, &frontier.FrontierURL{URL: normalized, Depth: parent.Depth + 1}); err != nil {
			return
		}
	}
}

// fetchPage performs an HTTP GET with the Weaver's User-Agent.
func (c *Crawler) fetchPage(ctx context.Context, u *url.URL) (*http.Response, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", u.String(), err)
	}

	req.Header.Set("User-Agent", c.weaver.cfg.UserAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", u.String(), err)
	}

	return resp, nil
}

// isHTML checks if the response Content-Type is text/html.
func isHTML(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "text/html")
}
