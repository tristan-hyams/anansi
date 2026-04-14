package parser_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tristan-hyams/anansi/parser"
	"github.com/tristan-hyams/anansi/testutil"
)

// TestExtractLinks_LiveSite hits a real URL and verifies the parser
// extracts links from a live HTML page. Skipped unless GO_RUN_INTEGRATIONS=true
// is set in .env.test or the environment.
func TestExtractLinks_LiveSite(t *testing.T) {
	testutil.SkipIfNoIntegration(t)
	t.Parallel()

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://crawlme.monzo.com/")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	links, err := parser.ExtractLinks(context.Background(), resp.Body)
	require.NoError(t, err)

	assert.NotEmpty(t, links, "expected at least one link on crawlme.monzo.com")

	t.Logf("extracted %d links from crawlme.monzo.com", len(links))
	for _, link := range links {
		t.Logf("  %s", link)
	}
}
