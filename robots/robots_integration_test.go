package robots_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tristan-hyams/anansi/robots"
	"github.com/tristan-hyams/anansi/testutil"
)

// TestFetch_LiveSite fetches robots.txt from crawlme.monzo.com.
// The site is hosted on S3/CloudFront which returns 403 for missing files
// instead of 404. Our Fetch treats 403 as not found (allow all) since
// CDN/static hosts commonly use 403 for missing objects.
// Skipped unless GO_RUN_INTEGRATIONS=true is set in .env.test.
func TestFetch_LiveSite(t *testing.T) {
	testutil.SkipIfNoIntegration(t)
	t.Parallel()

	client := &http.Client{Timeout: 10 * time.Second}

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(
		context.Background(),
		client,
		mustParseURL(t, "https://crawlme.monzo.com/"),
		testLogger(&logBuf),
	)

	require.NoError(t, err)
	require.NotNil(t, rules)

	assert.True(t, rules.IsAllowed("/"), "root path should be allowed")
	assert.Contains(t, logBuf.String(), "403")

	t.Logf("logger output:\n%s", logBuf.String())
}

// TestParseXRobotsTag_LiveSite fetches the crawlme.monzo.com homepage
// and parses its X-Robots-Tag header. The site returns "noindex,follow"
// which means: don't index in search engines, but do follow links.
// Our crawler should respect the follow directive and continue crawling.
func TestParseXRobotsTag_LiveSite(t *testing.T) {
	testutil.SkipIfNoIntegration(t)
	t.Parallel()

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://crawlme.monzo.com/")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	rawTag := resp.Header.Get("X-Robots-Tag")
	t.Logf("X-Robots-Tag: %q", rawTag)

	directives := robots.ParseXRobotsTag(resp.Header)
	t.Logf("directives: NoIndex=%v, NoFollow=%v", directives.NoIndex, directives.NoFollow)

	// crawlme.monzo.com returns "noindex,follow"
	assert.True(t, directives.NoIndex, "should be noindex")
	assert.True(t, directives.ShouldFollow(), "should allow following links")
}
