package fileutil

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tristan-hyams/anansi/weaver"
)

func TestRenderMarkdown_ContainsBanner(t *testing.T) {
	web := &weaver.Web{OriginURL: "https://example.com/"}
	out := RenderMarkdown(web)

	if !strings.Contains(out, "A N A N S I") {
		t.Fatal("expected banner in output")
	}
	if !strings.Contains(out, "https://example.com/") {
		t.Fatal("expected origin URL in output")
	}
}

func TestRenderMarkdown_ShowsCounts(t *testing.T) {
	web := &weaver.Web{
		Visited:   5,
		Skipped:   2,
		Duration:  3500 * time.Millisecond,
		OriginURL: "https://example.com/",
	}
	out := RenderMarkdown(web)

	if !strings.Contains(out, "Pages crawled: 5") {
		t.Fatal("expected visited count")
	}
	if !strings.Contains(out, "Pages skipped: 2") {
		t.Fatal("expected skipped count")
	}
	if !strings.Contains(out, "3.5s") {
		t.Fatal("expected rounded duration")
	}
}

func TestRenderMarkdown_ListsCrawledPages(t *testing.T) {
	web := &weaver.Web{
		Visited:   2,
		OriginURL: "https://example.com/",
		Pages: []weaver.PageResult{
			{URL: "https://example.com/b", Links: 3, Depth: 1, Status: 200, Duration: time.Millisecond},
			{URL: "https://example.com/a", Links: 5, Depth: 0, Status: 200, Duration: time.Millisecond},
		},
	}
	out := RenderMarkdown(web)

	// Depth-sorted: /a (depth 0) before /b (depth 1).
	idxA := strings.Index(out, "example.com/a")
	idxB := strings.Index(out, "example.com/b")
	if idxA < 0 || idxB < 0 {
		t.Fatal("expected both URLs in output")
	}
	if idxA > idxB {
		t.Fatal("expected /a before /b (sorted by depth)")
	}
}

func TestRenderMarkdown_ExcludesErroredPages(t *testing.T) {
	web := &weaver.Web{
		Visited:   1,
		Skipped:   1,
		OriginURL: "https://example.com/",
		Pages: []weaver.PageResult{
			{URL: "https://example.com/ok", Links: 2, Status: 200, Duration: time.Millisecond},
			{URL: "https://example.com/fail", Error: errors.New("timeout"), Duration: time.Millisecond},
		},
	}
	out := RenderMarkdown(web)

	if !strings.Contains(out, "example.com/ok") {
		t.Fatal("expected successful page in output")
	}
	// Errored pages should not appear in the flat page list.
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "example.com/fail") && strings.Contains(line, "→") {
			t.Fatal("errored page should not appear in page list")
		}
	}
}

func TestRenderMarkdown_SitemapTree(t *testing.T) {
	web := &weaver.Web{
		Visited:   3,
		OriginURL: "https://example.com/",
		Pages: []weaver.PageResult{
			{URL: "https://example.com/", Status: 200, Duration: time.Millisecond},
			{URL: "https://example.com/about", Status: 200, Duration: time.Millisecond},
			{URL: "https://example.com/blog/post-1", Status: 200, Duration: time.Millisecond},
		},
	}
	out := RenderMarkdown(web)

	if !strings.Contains(out, "Sitemap:") {
		t.Fatal("expected sitemap section")
	}
	if !strings.Contains(out, "about") {
		t.Fatal("expected 'about' in sitemap tree")
	}
	if !strings.Contains(out, "post-1") {
		t.Fatal("expected 'post-1' in sitemap tree")
	}
}

func TestRenderErrorLog_Empty(t *testing.T) {
	web := &weaver.Web{
		Pages: []weaver.PageResult{
			{URL: "https://example.com/", Status: 200},
		},
	}
	out := RenderErrorLog(web)
	if out != "" {
		t.Fatalf("expected empty error log, got %q", out)
	}
}

func TestRenderErrorLog_GroupsByReason(t *testing.T) {
	ts := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	web := &weaver.Web{
		Skipped:   3,
		OriginURL: "https://example.com/",
		Pages: []weaver.PageResult{
			{URL: "https://example.com/a", Error: errors.New("timeout"), Timestamp: ts},
			{URL: "https://example.com/b", Error: errors.New("timeout"), Timestamp: ts},
			{URL: "https://example.com/c", Error: errors.New("max depth exceeded"), Timestamp: ts},
		},
	}
	out := RenderErrorLog(web)

	if !strings.Contains(out, "timeout (2)") {
		t.Fatal("expected 'timeout (2)' group")
	}
	if !strings.Contains(out, "max depth exceeded (1)") {
		t.Fatal("expected 'max depth exceeded (1)' group")
	}
	if !strings.Contains(out, "Total errors: 3") {
		t.Fatal("expected total error count")
	}
}

func TestRenderErrorLog_SortsURLsWithinGroup(t *testing.T) {
	ts := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	web := &weaver.Web{
		Skipped:   2,
		OriginURL: "https://example.com/",
		Pages: []weaver.PageResult{
			{URL: "https://example.com/z", Error: errors.New("err"), Timestamp: ts},
			{URL: "https://example.com/a", Error: errors.New("err"), Timestamp: ts},
		},
	}
	out := RenderErrorLog(web)

	idxA := strings.Index(out, "example.com/a")
	idxZ := strings.Index(out, "example.com/z")
	if idxA > idxZ {
		t.Fatal("expected /a before /z (alphabetical sort)")
	}
}
