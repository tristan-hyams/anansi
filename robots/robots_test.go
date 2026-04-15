package robots_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tristan-hyams/anansi/robots"
)

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func testLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestFetch_StandardDisallow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /admin\nDisallow: /private\n"))
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.False(t, rules.IsAllowed("/admin"))
	assert.False(t, rules.IsAllowed("/admin/settings"))
	assert.False(t, rules.IsAllowed("/private"))
	assert.True(t, rules.IsAllowed("/public"))
	assert.True(t, rules.IsAllowed("/"))
}

func TestFetch_EmptyBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.True(t, rules.IsAllowed("/anything"))
}

func TestFetch_404(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.True(t, rules.IsAllowed("/anything"))
	assert.Contains(t, logBuf.String(), "robots.txt not found")
}

func TestFetch_403_TreatedAsNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.True(t, rules.IsAllowed("/anything"))
	assert.Contains(t, logBuf.String(), "403")
}

func TestFetch_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	assert.Error(t, err)
	assert.Nil(t, rules)
	assert.Contains(t, err.Error(), "500")
}

func TestFetch_NetworkError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.True(t, rules.IsAllowed("/anything"))
	assert.Contains(t, logBuf.String(), "fetch failed")
}

func TestFetch_ContextCancellation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(ctx, mustParseURL(t, srv.URL), testLogger(&logBuf))
	assert.Error(t, err)
	assert.Nil(t, rules)
}

func TestFetch_MultipleUserAgents(t *testing.T) {
	t.Parallel()

	body := "User-agent: Googlebot\nDisallow: /google-only\n\nUser-agent: *\nDisallow: /blocked\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.False(t, rules.IsAllowed("/blocked"))
	assert.True(t, rules.IsAllowed("/google-only"))
}

func TestFetch_AllowAndDisallow(t *testing.T) {
	t.Parallel()

	body := "User-agent: *\nDisallow: /\nAllow: /public\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.True(t, rules.IsAllowed("/public"))
	assert.True(t, rules.IsAllowed("/public/page"))
	assert.False(t, rules.IsAllowed("/private"))
	assert.False(t, rules.IsAllowed("/"))
}

func TestFetch_CrawlDelay(t *testing.T) {
	t.Parallel()

	body := "User-agent: *\nCrawl-delay: 10\nDisallow: /slow\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.Equal(t, 10*time.Second, rules.CrawlDelay())
}

func TestCrawlDelay_NotSpecified(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /nope\n"))
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.Equal(t, time.Duration(0), rules.CrawlDelay())
}

func TestCrawlDelay_NilRules(t *testing.T) {
	t.Parallel()

	rules := &robots.Rules{}
	assert.Equal(t, time.Duration(0), rules.CrawlDelay())
}

func TestIsAllowed_NilRules(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.True(t, rules.IsAllowed("/anything"))
	assert.True(t, rules.IsAllowed("/admin"))
	assert.Equal(t, time.Duration(0), rules.CrawlDelay())
}

func TestFetch_NoStarUserAgent(t *testing.T) {
	t.Parallel()

	body := "User-agent: Googlebot\nDisallow: /blocked\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	rules, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.True(t, rules.IsAllowed("/blocked"))
	assert.True(t, rules.IsAllowed("/anything"))
	assert.Equal(t, time.Duration(0), rules.CrawlDelay())
}

func TestFetch_SetsUserAgentHeader(t *testing.T) {
	t.Parallel()

	var receivedUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte("User-agent: *\nDisallow:\n"))
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	_, err := robots.Fetch(context.Background(), mustParseURL(t, srv.URL), testLogger(&logBuf))
	require.NoError(t, err)

	assert.Equal(t, "Anansi", receivedUA)
}
