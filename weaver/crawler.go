package weaver

import (
	"context"
	"fmt"
	"io"
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

// readCloser pairs a limited Reader with the original Closer
// so io.LimitReader doesn't prevent body cleanup.
type readCloser struct {
	io.Reader
	io.Closer
}

// Crawler is a single worker that fetches and processes pages.
// Each Crawler runs in its own goroutine, managed by the Weaver.
// Has its own http.Client backed by the Weaver's shared Transport.
type Crawler struct {
	id     uuid.UUID
	weaver *Weaver
	client *http.Client
}

// crawl is the main loop - dequeue URLs, process them, repeat until context is cancelled.
//
// Each processed URL calls frontier.Done() to decrement the pending counter.
// When pending reaches 0, the monitor knows all discovered URLs have been
// fully processed - deterministic completion without polling races.
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
		w.logger.Warn("fetch failed", logKeyURL, pageURL, logKeyError, err)
		w.recordPage(PageResult{URL: pageURL, Depth: fu.Depth, Duration: time.Since(start), Error: err})
		return
	}
	resp.Body = readCloser{io.LimitReader(resp.Body, maxResponseBodySize), resp.Body}
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
		w.logger.Debug("non-HTML content, skipping parse", logKeyURL, pageURL, "content_type", ct)
		w.recordPage(PageResult{
			URL: pageURL, Depth: fu.Depth, Status: resp.StatusCode,
			ContentType: ct, Duration: time.Since(start),
		})
		return
	}

	if !robots.ParseXRobotsTag(resp.Header).ShouldFollow() {
		w.logger.Info("X-Robots-Tag nofollow, skipping link extraction", logKeyURL, pageURL)
		w.recordPage(PageResult{
			URL: pageURL, Depth: fu.Depth, Status: resp.StatusCode,
			ContentType: ct, Duration: time.Since(start),
		})
		return
	}

	c.extractAndEnqueue(ctx, fu, resp, pageURL, ct, start)
}

// extractAndEnqueue parses links from an HTML response, records the page
// result, prints discovered links (if configured), and enqueues same-domain
// URLs for further crawling.
func (c *Crawler) extractAndEnqueue(
	ctx context.Context, fu *frontier.FrontierURL, resp *http.Response,
	pageURL string, ct string, start time.Time) {

	w := c.weaver

	hrefs, err := parser.ExtractLinks(ctx, resp.Body)
	if err != nil {
		w.logger.Warn("link extraction failed", logKeyURL, pageURL, logKeyError, err)
	}

	resolved := c.resolveLinks(fu.URL, hrefs)
	foundLinks := urlStrings(resolved)

	if w.cfg.LogLinks {
		w.logger.Info("crawled",
			logKeyURL, pageURL,
			logKeyDepth, fu.Depth,
			logKeyLinks, len(hrefs),
			"duration", time.Since(start),
		)
	}

	pr := PageResult{
		URL: pageURL, Links: len(foundLinks), FoundLinks: foundLinks, Depth: fu.Depth,
		Status: resp.StatusCode, ContentType: ct, Duration: time.Since(start),
		Error: err,
	}
	w.recordPage(pr)
	w.printPage(pr)

	c.enqueueLinks(ctx, fu, resolved)
}

// resolveLinks normalizes raw hrefs to deduplicated absolute URLs.
// Unparseable and duplicate hrefs are silently skipped.
func (*Crawler) resolveLinks(base *url.URL, hrefs []string) []*url.URL {

	seen := make(map[string]struct{}, len(hrefs))
	result := make([]*url.URL, 0, len(hrefs))

	for _, raw := range hrefs {
		normalized, err := normalizer.Normalize(base, raw)
		if err != nil {
			continue
		}

		key := normalized.String()
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

// urlStrings converts resolved URLs to strings for display.
func urlStrings(urls []*url.URL) []string {
	s := make([]string, len(urls))
	for i, u := range urls {
		s[i] = u.String()
	}
	return s
}

// enqueueLinks filters and enqueues already-normalized URLs.
func (c *Crawler) enqueueLinks(ctx context.Context, parent *frontier.FrontierURL, links []*url.URL) {
	w := c.weaver

	for _, normalized := range links {
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

		if err := w.front.Enqueue(ctx, &frontier.FrontierURL{URL: normalized, Depth: parent.Depth + 1}); err != nil {
			return
		}
	}
}

// fetchPage performs an HTTP GET with retry on transient errors.
// Transient: connection reset, timeout, 5xx. Non-retryable: 4xx, context cancelled.
func (c *Crawler) fetchPage(ctx context.Context, u *url.URL) (*http.Response, error) {

	return withRetry(ctx, retryConfig{
		maxRetries: c.weaver.cfg.MaxRetries,
		baseDelay:  baseRetryDelay,
		logger:     c.weaver.logger,
		label:      u.String(),
	}, func() (*http.Response, error, bool) {

		resp, err := c.doRequest(ctx, u)
		if err != nil {
			return nil, err, true // network error — retryable
		}

		if resp.StatusCode >= serverErrorThreshold {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, u), true
		}

		return resp, nil, false
	})
}

// doRequest executes a single HTTP GET.
func (c *Crawler) doRequest(ctx context.Context, u *url.URL) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", u, err)
	}

	req.Header.Set("User-Agent", c.weaver.cfg.UserAgent)
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("Accept-Language", acceptLanguageHeader)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", u, err)
	}

	return resp, nil
}

// isHTML checks if the response Content-Type is text/html.
func isHTML(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "text/html")
}
