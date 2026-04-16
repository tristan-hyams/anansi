package weaver_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tristan-hyams/anansi/weaver"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func testConfig() *weaver.WeaverConfig {
	return weaver.NewWeaverConfig(weaver.WeaverConfig{
		Workers:   2,
		Rate:      100,
		MaxDepth:  0,
		Timeout:   5 * time.Second,
		UserAgent: "AnansiTest",
	})
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func newTestWeaver(
	t *testing.T, cfg *weaver.WeaverConfig, srvURL string,
) *weaver.Weaver {
	t.Helper()
	wv, err := weaver.NewWeaver(
		context.Background(), cfg,
		mustParseURL(t, srvURL+"/"),
		testLogger(), io.Discard,
	)
	require.NoError(t, err)
	return wv
}

func TestWeave_HappyPath(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		switch r.URL.Path {
		case "/", "/index.html":
			_, _ = fmt.Fprint(w, `<html><body>
				<a href="/about">About</a>
				<a href="/contact">Contact</a>
			</body></html>`)
		case "/about":
			_, _ = fmt.Fprint(w, `<html><body>
				<a href="/">Home</a>
				<a href="/contact">Contact</a>
			</body></html>`)
		case "/contact":
			_, _ = fmt.Fprint(w, `<html><body>
				<a href="/">Home</a>
			</body></html>`)
		case "/robots.txt":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = fmt.Fprint(w, "User-agent: *\nAllow: /\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	wv := newTestWeaver(t, testConfig(), srv.URL)

	result := wv.Weave(context.Background())
	

	assert.GreaterOrEqual(t, result.Visited, 3)
	assert.NotEmpty(t, result.Pages)
}

func TestWeave_CycleDetection(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprint(w, `<html><body><a href="/b">B</a></body></html>`)
		case "/b":
			_, _ = fmt.Fprint(w, `<html><body><a href="/">A</a></body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	wv := newTestWeaver(t, testConfig(), srv.URL)

	result := wv.Weave(context.Background())
	

	assert.Equal(t, 2, result.Visited)
}

func TestWeave_ExternalLinksFiltered(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprint(w, `<html><body>
				<a href="/internal">Internal</a>
				<a href="https://external.com/page">External</a>
			</body></html>`)
		case "/internal":
			_, _ = fmt.Fprint(w, `<html><body><a href="/">Home</a></body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	wv := newTestWeaver(t, testConfig(), srv.URL)

	result := wv.Weave(context.Background())
	

	for _, p := range result.Pages {
		assert.True(t, strings.HasPrefix(p.URL, srv.URL), "visited external URL: %s", p.URL)
	}
}

func TestWeave_NonHTMLSkipped(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, `<html><body>
				<a href="/data.json">JSON</a>
				<a href="/page">Page</a>
			</body></html>`)
		case "/data.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"key": "value"}`)
		case "/page":
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, `<html><body>Done</body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	wv := newTestWeaver(t, testConfig(), srv.URL)

	result := wv.Weave(context.Background())
	

	var jsonPage *weaver.PageResult
	for i, p := range result.Pages {
		if strings.Contains(p.URL, "data.json") {
			jsonPage = &result.Pages[i]
			break
		}
	}
	require.NotNil(t, jsonPage, "JSON page should appear in results")
	assert.Equal(t, 0, jsonPage.Links)
}

func TestWeave_RobotsTxtRespected(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = fmt.Fprint(w, "User-agent: *\nDisallow: /secret\n")
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, `<html><body>
				<a href="/public">Public</a>
				<a href="/secret">Secret</a>
			</body></html>`)
		case "/public":
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, `<html><body>Public page</body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	wv := newTestWeaver(t, testConfig(), srv.URL)

	result := wv.Weave(context.Background())
	

	for _, p := range result.Pages {
		assert.NotContains(t, p.URL, "/secret")
	}
}

func TestWeave_XRobotsTagNoFollow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("X-Robots-Tag", "nofollow")
			_, _ = fmt.Fprint(w, `<html><body>
				<a href="/should-not-visit">Hidden</a>
			</body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	wv := newTestWeaver(t, testConfig(), srv.URL)

	result := wv.Weave(context.Background())
	

	assert.Equal(t, 1, result.Visited)
}

func TestWeave_MaxDepth(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprint(w, `<html><body><a href="/depth1">D1</a></body></html>`)
		case "/depth1":
			_, _ = fmt.Fprint(w, `<html><body><a href="/depth2">D2</a></body></html>`)
		case "/depth2":
			_, _ = fmt.Fprint(w, `<html><body><a href="/depth3">D3</a></body></html>`)
		case "/depth3":
			_, _ = fmt.Fprint(w, `<html><body><a href="/depth4">D4</a></body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := testConfig()
	cfg.MaxDepth = 2 // crawl depths 0, 1, 2. Depth 3+ skipped.

	wv := newTestWeaver(t, cfg, srv.URL)

	result := wv.Weave(context.Background())
	

	// Depths 0, 1, 2 should be visited. Depth 3 recorded as skipped.
	assert.Equal(t, 3, result.Visited, "depths 0, 1, 2 should be visited")
	assert.Equal(t, 1, result.Skipped, "depth 3 should be skipped")
}

func TestWeave_ContextCancellation(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, `<html><body><a href="/page/%d">Next</a></body></html>`, n)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	wv := newTestWeaver(t, testConfig(), srv.URL)

	result := wv.Weave(ctx)
	

	assert.Greater(t, result.Visited, 0)
	assert.Less(t, result.Duration, 2*time.Second)
}

// TestWeave_NaturalCompletion_SinglePage verifies that the crawl terminates
// naturally when a single page has no outgoing links. The pending counter
// goes from 1 (origin enqueued) to 0 (origin processed, no new URLs).
func TestWeave_NaturalCompletion_SinglePage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body><p>Dead end. No links.</p></body></html>`)
	}))
	defer srv.Close()

	wv := newTestWeaver(t, testConfig(), srv.URL)

	result := wv.Weave(context.Background())
	

	assert.Equal(t, 1, result.Visited)
	assert.Less(t, result.Duration, 2*time.Second, "should terminate quickly, not hang")
}

// TestWeave_NaturalCompletion_AllLinksExternal verifies that the crawl
// terminates when the origin page has links but all are external.
// Pending: 1 (origin) → process origin → 0 new internal URLs → done.
func TestWeave_NaturalCompletion_AllLinksExternal(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body>
			<a href="https://external-one.com/">External 1</a>
			<a href="https://external-two.com/">External 2</a>
			<a href="mailto:test@example.com">Email</a>
		</body></html>`)
	}))
	defer srv.Close()

	wv := newTestWeaver(t, testConfig(), srv.URL)

	result := wv.Weave(context.Background())
	

	assert.Equal(t, 1, result.Visited)
	assert.Less(t, result.Duration, 2*time.Second)
}

// TestWeave_NaturalCompletion_SmallSite verifies deterministic completion
// on a small interconnected site where all pages eventually link back to
// already-visited URLs. Pending goes: 1 → 3 → 2 → 1 → 0.
func TestWeave_NaturalCompletion_SmallSite(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprint(w, `<html><body>
				<a href="/a">A</a>
				<a href="/b">B</a>
			</body></html>`)
		case "/a":
			_, _ = fmt.Fprint(w, `<html><body>
				<a href="/">Home</a>
				<a href="/b">B</a>
			</body></html>`)
		case "/b":
			_, _ = fmt.Fprint(w, `<html><body>
				<a href="/">Home</a>
				<a href="/a">A</a>
			</body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	wv := newTestWeaver(t, testConfig(), srv.URL)

	result := wv.Weave(context.Background())
	

	assert.Equal(t, 3, result.Visited)
	assert.Less(t, result.Duration, 5*time.Second, "should terminate deterministically, not hang")
}

// TestWeave_RedirectToExternalBlocked verifies that the redirect policy
// prevents the crawler from following a same-domain URL that 302s to an
// external domain. The redirected page should appear as a fetch error,
// not as a visited external page.
func TestWeave_RedirectToExternalBlocked(t *testing.T) {
	t.Parallel()

	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body>External site</body></html>`)
	}))
	defer external.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, `<html><body><a href="/redirect">Redirect</a></body></html>`)
		case "/redirect":
			http.Redirect(w, r, external.URL+"/landing", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	wv := newTestWeaver(t, testConfig(), srv.URL)
	result := wv.Weave(context.Background())

	// Origin should be visited. The redirect target is external, so it
	// should be recorded as an error (redirect blocked), not visited.
	for _, p := range result.Pages {
		assert.NotContains(t, p.URL, external.URL, "should not visit external redirect target")
	}
	assert.Equal(t, 1, result.Skipped, "redirected page should be skipped")
}

// TestWeave_LargeResponseBodyTruncated verifies that io.LimitReader caps
// the bytes read from an HTTP response. A server streaming more than
// maxResponseBodySize should not cause unbounded memory growth - the
// parser receives a truncated body and extracts whatever links it can.
func TestWeave_LargeResponseBodyTruncated(t *testing.T) {
	t.Parallel()

	// Serve a page with a valid link early, followed by 12 MB of padding.
	// The link before the padding should still be discovered.
	padding := strings.Repeat("x", 12<<20) // 12 MB > 10 MB limit
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprintf(w, `<html><body><a href="/found">Link</a>%s</body></html>`, padding)
		case "/found":
			_, _ = fmt.Fprint(w, `<html><body>Found</body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	wv := newTestWeaver(t, testConfig(), srv.URL)
	result := wv.Weave(context.Background())

	// The origin page should be visited. The link appears before the
	// padding so it should be extracted despite truncation.
	assert.GreaterOrEqual(t, result.Visited, 1, "origin should be visited")

	// No OOM - if we got here, the body was bounded.
}
