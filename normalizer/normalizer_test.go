package normalizer_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tristan-hyams/anansi/normalizer"
)

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func TestNormalize(t *testing.T) {
	t.Parallel()

	base := mustParse(t, "https://crawlme.monzo.com/")

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		// Relative URL resolution
		{"relative path", "/about", "https://crawlme.monzo.com/about", false},
		{"relative child", "./child", "https://crawlme.monzo.com/child", false},
		{"relative parent", "../sibling", "https://crawlme.monzo.com/sibling", false},

		// Fragment stripping
		{"strip fragment", "https://crawlme.monzo.com/page#section", "https://crawlme.monzo.com/page", false},
		{"fragment-only href", "#top", "https://crawlme.monzo.com/", false},

		// Scheme and host lowercasing
		{"uppercase scheme", "HTTPS://crawlme.monzo.com/page", "https://crawlme.monzo.com/page", false},
		{"uppercase host", "https://CRAWLME.MONZO.COM/page", "https://crawlme.monzo.com/page", false},
		{"mixed case", "HTTP://CrawlMe.Monzo.COM/About", "http://crawlme.monzo.com/About", false},

		// Default port stripping
		{"strip http :80", "http://example.com:80/path", "http://example.com/path", false},
		{"strip https :443", "https://example.com:443/path", "https://example.com/path", false},
		{"keep non-default port", "https://example.com:8080/path", "https://example.com:8080/path", false},
		{"keep port 8080 not confused with 80", "http://example.com:8080/path", "http://example.com:8080/path", false},
		{"keep port 180", "http://example.com:180/path", "http://example.com:180/path", false},
		{"keep port 4431", "https://example.com:4431/path", "https://example.com:4431/path", false},

		// Absolute URLs
		{"absolute same domain", "https://crawlme.monzo.com/other", "https://crawlme.monzo.com/other", false},
		{"absolute external", "https://external.com/page", "https://external.com/page", false},

		// Protocol-relative
		{"protocol-relative", "//cdn.example.com/script.js", "https://cdn.example.com/script.js", false},

		// Query params preserved
		{"query params", "/search?q=test&page=1", "https://crawlme.monzo.com/search?q=test&page=1", false},
		{"query with fragment", "/page?q=1#anchor", "https://crawlme.monzo.com/page?q=1", false},

		// Encoded characters
		{"encoded spaces", "/path%20with%20spaces", "https://crawlme.monzo.com/path%20with%20spaces", false},

		// Error cases
		{"empty href", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := normalizer.Normalize(base, tt.raw)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestIsSameHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		origin    string
		candidate string
		want      bool
	}{
		{"exact match", "https://crawlme.monzo.com/", "https://crawlme.monzo.com/about", true},
		{"different path", "https://crawlme.monzo.com/", "https://crawlme.monzo.com/a/b/c", true},
		{"different scheme same host", "https://crawlme.monzo.com/", "http://crawlme.monzo.com/page", true},
		{"case insensitive", "https://crawlme.monzo.com/", "https://CrawlMe.Monzo.COM/page", true},
		{"default port match", "https://crawlme.monzo.com/", "https://crawlme.monzo.com:443/page", true},

		{"different subdomain", "https://crawlme.monzo.com/", "https://www.crawlme.monzo.com/", false},
		{"parent domain", "https://crawlme.monzo.com/", "https://monzo.com/", false},
		{"different domain", "https://crawlme.monzo.com/", "https://example.com/", false},
		{"different port", "https://crawlme.monzo.com/", "https://crawlme.monzo.com:8080/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			origin := mustParse(t, tt.origin)
			candidate := mustParse(t, tt.candidate)
			assert.Equal(t, tt.want, normalizer.IsSameHost(origin, candidate))
		})
	}
}

func TestIsFollowableScheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{"https", "https://example.com/", true},
		{"http", "http://example.com/", true},
		{"HTTP uppercase", "HTTP://example.com/", true},

		{"mailto", "mailto:user@example.com", false},
		{"javascript", "javascript:void(0)", false},
		{"tel", "tel:+1234567890", false},
		{"ftp", "ftp://files.example.com/", false},
		{"data", "data:text/html,<h1>hi</h1>", false},
		{"empty scheme", "//example.com/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u := mustParse(t, tt.raw)
			assert.Equal(t, tt.want, normalizer.IsFollowableScheme(u))
		})
	}
}
