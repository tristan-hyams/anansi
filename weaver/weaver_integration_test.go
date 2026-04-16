package weaver_test

import (
	"context"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tristan-hyams/anansi/testutil"
	"github.com/tristan-hyams/anansi/weaver"
)

const origin = "https://crawlme.monzo.com/"

var knownPages = []string{
	"https://crawlme.monzo.com/",
	"https://crawlme.monzo.com/about.html",
	"https://crawlme.monzo.com/products.html",
	"https://crawlme.monzo.com/services.html",
	"https://crawlme.monzo.com/blog.html",
	"https://crawlme.monzo.com/contact.html",
	"https://crawlme.monzo.com/help.html",
	"https://crawlme.monzo.com/faq.html",
	"https://crawlme.monzo.com/terms.html",
	"https://crawlme.monzo.com/privacy.html",
}

// TestWeave_CrawlMe is a live integration test against crawlme.monzo.com.
// The site is a static S3/CloudFront site with known structure:
//   - 10 top-level pages (index, about, products, services, blog, contact, help, faq, terms, privacy)
//   - /products/ - paginated (7+ pages), each listing UUID-named product pages (100+ total)
//   - /blog/ - paginated (5+ pages), each listing numbered post pages (77+ total)
//   - contact.html has external links (instagram, facebook, twitter), mailto:, tel:
//   - X-Robots-Tag: noindex,follow on all pages (follow = crawl allowed)
//   - robots.txt returns 403 (S3 missing file behavior, treated as allow-all)
//
// We use MaxDepth=2 and limited workers to keep the test fast while still
// verifying the core crawl pipeline works end-to-end.
func TestWeave_CrawlMe(t *testing.T) {
	testutil.SkipIfNoIntegration(t)
	t.Parallel()

	origin, err := url.Parse(origin)
	require.NoError(t, err)

	cfg := &weaver.WeaverConfig{
		Workers:   2,
		Rate:      5,
		MaxDepth:  2,
		Timeout:   30 * time.Second,
		UserAgent: "Anansi/test",
	}

	logger := testLogger()
	wv, err := weaver.NewWeaver(context.Background(), cfg, origin, logger, io.Discard)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	web, err := wv.Weave(ctx)
	require.NoError(t, err)
	require.NotNil(t, web)

	t.Logf("visited: %d, skipped: %d, duration: %v", web.Visited, web.Skipped, web.Duration)

	// --- Known structure assertions ---

	// Must have visited at least the 10 top-level pages.
	assert.GreaterOrEqual(t, web.Visited, 10, "should visit at least 10 top-level pages")

	// Collect visited URLs for specific checks.
	visitedURLs := make(map[string]bool)
	for _, p := range web.Pages {
		visitedURLs[p.URL] = true
	}

	for _, page := range knownPages {
		assert.True(t, visitedURLs[page], "expected page to be visited: %s", page)
	}

	// External links must NOT be visited.
	externalDomains := []string{
		"instagram.com",
		"facebook.com",
		"twitter.com",
	}
	for _, p := range web.Pages {
		for _, domain := range externalDomains {
			assert.NotContains(t, p.URL, domain, "external URL should not be visited: %s", p.URL)
		}
	}

	// No mailto: or tel: URLs should appear in results.
	for _, p := range web.Pages {
		assert.NotContains(t, p.URL, "mailto:", "mailto URL should not be visited: %s", p.URL)
		assert.NotContains(t, p.URL, "tel:", "tel URL should not be visited: %s", p.URL)
	}

	// All pages should be on crawlme.monzo.com.
	for _, p := range web.Pages {
		assert.Contains(t, p.URL, "crawlme.monzo.com", "all visited URLs should be on crawlme.monzo.com: %s", p.URL)
	}

	// Duration should be reasonable (not hung).
	assert.Less(t, web.Duration, 60*time.Second)

	// Log some results for manual inspection.
	for _, p := range web.Pages {
		t.Logf("  depth=%d links=%d status=%d %s", p.Depth, p.Links, p.Status, p.URL)
	}
}
