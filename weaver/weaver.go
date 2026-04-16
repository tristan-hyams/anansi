// Package weaver is the orchestration layer for the Anansi web crawler.
// The Weaver manages the crawl - it owns the frontier, rate limiter, robots
// rules, and spawns Crawlers (workers) that fetch and parse pages.
//
// Named after Anansi the spider: the weaver weaves the web,
// and crawlers venture out to fetch pages.
package weaver

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"

	"github.com/tristan-hyams/anansi/frontier"
	"github.com/tristan-hyams/anansi/normalizer"
	"github.com/tristan-hyams/anansi/robots"
	"github.com/tristan-hyams/anansi/webutil"
)

// Weaver orchestrates the crawl. Created via NewWeaver(), executed via Weave().
type Weaver struct {
	cfg      *WeaverConfig
	origin   *url.URL
	limiter  *rate.Limiter
	front    frontier.Frontier
	rules    *robots.Rules
	logger   *slog.Logger
	output   io.Writer
	crawlers []*Crawler
	mu       sync.Mutex
	outputMu sync.Mutex
	pages    []PageResult
}

// NewWeaver creates a Weaver. Fetches robots.txt during construction.
// If robots.txt fetch fails, the crawl continues with allow-all rules.
func NewWeaver(
	ctx context.Context, cfg *WeaverConfig, origin *url.URL,
	logger *slog.Logger, output io.Writer,
) (*Weaver, error) {

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Fetch robots.txt - 404/403/network errors degrade to allow-all inside
	// Fetch itself. If Fetch returns an error, it's a real failure (5xx,
	// context cancellation) and we should not proceed.
	rules, err := robots.Fetch(ctx, origin, logger)
	if err != nil {
		return nil, fmt.Errorf("robots.txt: %w", err)
	}

	crawlRate := cfg.CrawlRate(rules)
	logger.Info("effective crawl rate", "rate", float64(crawlRate))

	front := frontier.NewInMemory(cfg.BufferSize, logger)

	originFU := &frontier.FrontierURL{URL: origin, Depth: 0}
	if err := front.Enqueue(ctx, originFU); err != nil {
		return nil, fmt.Errorf("enqueueing origin URL: %w", err)
	}

	wv := &Weaver{
		cfg:     cfg,
		origin:  origin,
		limiter: rate.NewLimiter(crawlRate, 1),
		front:   front,
		rules:   rules,
		logger:  logger,
		output:  output,
	}

	// Pre-create crawlers - each gets its own HTTP client backed by
	// the singleton transport. Created once, reused across Weave calls.
	wv.crawlers = make([]*Crawler, cfg.Workers)
	for i := range cfg.Workers {
		client := webutil.NewClient(cfg.Timeout)
		client.CheckRedirect = wv.redirectPolicy
		wv.crawlers[i] = &Crawler{
			id:     uuid.New(),
			weaver: wv,
			client: client,
		}
	}

	return wv, nil
}

// Weave starts the crawl and blocks until completion or context cancellation.
func (w *Weaver) Weave(ctx context.Context) (*Web, error) {

	start := time.Now()
	w.logger.Info("crawl started", "origin", w.origin.String(), "workers", w.cfg.Workers)

	var crawlCtx context.Context
	var crawlCancel context.CancelFunc
	if w.cfg.MaxDuration > 0 {
		crawlCtx, crawlCancel = context.WithTimeout(ctx, w.cfg.MaxDuration)
	} else {
		crawlCtx, crawlCancel = context.WithCancel(ctx)
	}
	defer crawlCancel()

	var wg sync.WaitGroup
	for _, c := range w.crawlers {
		wg.Go(func() {
			c.crawl(crawlCtx)
		})
	}

	go w.monitorCompletion(crawlCtx, crawlCancel)

	wg.Wait()

	return w.buildResult(start), nil
}

// monitorCompletion polls until the crawl is naturally complete.
// Uses the frontier's pending counter - deterministic, no races.
// Pending is incremented on enqueue and decremented when a crawler
// calls Done() after fully processing a URL. When pending reaches 0,
// every discovered URL has been processed and no new work was generated.
func (w *Weaver) monitorCompletion(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if w.front.IsDone() {
				cancel()
				return
			}
		}
	}
}

// buildResult assembles the crawl summary.
func (w *Weaver) buildResult(start time.Time) *Web {

	w.mu.Lock()
	defer w.mu.Unlock()

	visited := 0
	skipped := 0

	for _, p := range w.pages {
		if p.Error != nil {
			skipped++
		} else {
			visited++
		}
	}

	return &Web{
		Visited:   visited,
		Skipped:   skipped,
		Duration:  time.Since(start),
		OriginURL: w.origin.String(),
		Pages:     w.pages,
	}
}

// recordPage appends a page result with timestamp (thread-safe).
func (w *Weaver) recordPage(pr PageResult) {
	pr.Timestamp = time.Now()
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pages = append(w.pages, pr)
}

// printPage writes a visited URL and its discovered links to the output
// writer. Thread-safe - uses a separate mutex to avoid contention with
// recordPage. Output is buffered per page and written in one call to
// prevent interleaving between crawlers.
func (w *Weaver) printPage(pr PageResult) {
	if !w.cfg.LogLinks || w.output == nil {
		return
	}

	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "%s\n", pr.URL)
	for _, link := range pr.FoundLinks {
		_, _ = fmt.Fprintf(&sb, "  %s\n", link)
	}
	_, _ = fmt.Fprint(&sb, "\n")

	w.outputMu.Lock()
	defer w.outputMu.Unlock()
	_, _ = fmt.Fprint(w.output, sb.String())
}

// redirectPolicy rejects redirects to external domains and caps the
// redirect chain length. Prevents the crawler from silently following
// a same-domain URL that 302s to an off-domain destination.
func (w *Weaver) redirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}

	if !normalizer.IsSameHost(w.origin, req.URL) {
		return fmt.Errorf("redirect to external domain %s blocked", req.URL.Host)
	}

	return nil
}
