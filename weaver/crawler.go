package weaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/tristan-hyams/anansi/frontier"
	"github.com/tristan-hyams/anansi/normalizer"
	"github.com/tristan-hyams/anansi/parser"
	"github.com/tristan-hyams/anansi/robots"
)

// Crawler is a single worker that fetches and processes pages.
// Each Crawler runs in its own goroutine, managed by the Weaver.
// Has its own http.Client backed by the Weaver's shared Transport.
type Crawler struct {
	id     uuid.UUID
	weaver *Weaver
	client *http.Client
}

// crawl is the main loop — dequeue URLs, process them, repeat until context is cancelled.
//
// Each processed URL calls frontier.Done() to decrement the pending counter.
// When pending reaches 0, the monitor knows all discovered URLs have been
// fully processed — deterministic completion without polling races.
func (c *Crawler) crawl(ctx context.Context) {
	c.weaver.logger.Info("crawler started", logKeyCrawlerID, c.id)
	defer c.weaver.logger.Info("crawler stopped", logKeyCrawlerID, c.id)

	processed := 0
	interval := c.weaver.cfg.ProgressInterval

	for {
		fu, err := c.weaver.front.Dequeue(ctx)
		if err != nil {
			return // context cancelled
		}

		c.processURL(ctx, fu)
		c.weaver.front.Done()

		processed++
		if processed%interval == 0 {
			c.weaver.logger.Info("crawler progress",
				logKeyCrawlerID, c.id,
				"processed", processed,
			)
		}
	}
}

// processURL handles a single URL: fetch, check headers, parse, enqueue new links.
func (c *Crawler) processURL(ctx context.Context, fu *frontier.FrontierURL) {

	w := c.weaver
	start := time.Now()
	pageURL := fu.URL.String()

	// MaxDepth=3 means crawl depths 0, 1, 2, 3. Skip depth 4+.
	if w.cfg.MaxDepth > 0 && fu.Depth > w.cfg.MaxDepth {
		w.logger.Debug("max depth reached, skipping", logKeyURL, pageURL, logKeyDepth, fu.Depth)
		w.recordPage(PageResult{URL: pageURL, Depth: fu.Depth, Duration: time.Since(start), Error: errMaxDepth})
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

	c.handleResponse(ctx, fu, resp, pageURL, start)
}

// handleResponse processes a successful HTTP response.
func (c *Crawler) handleResponse(
	ctx context.Context, fu *frontier.FrontierURL, resp *http.Response, pageURL string, start time.Time) {

	w := c.weaver
	ct := resp.Header.Get("Content-Type")

	w.logger.Debug("fetched", logKeyURL, pageURL, "status", resp.StatusCode)

	if !isHTML(resp) {
		w.logger.Debug("non-HTML content, skipping parse",
			logKeyURL, pageURL,
			"content_type", ct,
		)
		w.recordPage(PageResult{
			URL: pageURL, Depth: fu.Depth, Status: resp.StatusCode,
			ContentType: ct, Duration: time.Since(start),
		})
		return
	}

	directives := robots.ParseXRobotsTag(resp.Header)
	if !directives.ShouldFollow() {
		w.logger.Info("X-Robots-Tag nofollow, skipping link extraction", logKeyURL, pageURL)
		w.recordPage(PageResult{
			URL: pageURL, Depth: fu.Depth, Status: resp.StatusCode,
			ContentType: ct, Duration: time.Since(start),
		})
		return
	}

	links, err := parser.ExtractLinks(ctx, resp.Body)
	if err != nil {
		w.logger.Warn("link extraction failed", logKeyURL, pageURL, "error", err)
	}

	w.logger.Debug("crawled",
		logKeyURL, pageURL,
		logKeyDepth, fu.Depth,
		logKeyLinks, len(links),
		"duration", time.Since(start),
	)

	w.recordPage(PageResult{
		URL: pageURL, Links: len(links), Depth: fu.Depth,
		Status: resp.StatusCode, ContentType: ct, Duration: time.Since(start),
		Error: err,
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
