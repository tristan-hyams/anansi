package fileutil

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/tristan-hyams/anansi/weaver"
)

func TestRenderJSON_ValidJSON(t *testing.T) {
	web := &weaver.Web{
		Visited:   1,
		OriginURL: "https://example.com/",
		Duration:  2 * time.Second,
		Pages: []weaver.PageResult{
			{
				URL: "https://example.com/", Links: 3, Depth: 0,
				Status: 200, ContentType: "text/html",
				Duration: 100 * time.Millisecond, Timestamp: time.Now(),
			},
		},
	}

	data, err := RenderJSON(web)
	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}

	var output jsonOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if output.Origin != "https://example.com/" {
		t.Fatalf("expected origin https://example.com/, got %s", output.Origin)
	}
	if output.Visited != 1 {
		t.Fatalf("expected visited=1, got %d", output.Visited)
	}
	if len(output.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(output.Pages))
	}
	if output.Pages[0].Links != 3 {
		t.Fatalf("expected 3 links, got %d", output.Pages[0].Links)
	}
}

func TestRenderJSON_SeparatesPagesAndErrors(t *testing.T) {
	ts := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	web := &weaver.Web{
		Visited:   1,
		Skipped:   1,
		OriginURL: "https://example.com/",
		Duration:  time.Second,
		Pages: []weaver.PageResult{
			{URL: "https://example.com/ok", Status: 200, Duration: 50 * time.Millisecond, Timestamp: ts},
			{URL: "https://example.com/fail", Error: errors.New("timeout"), Depth: 2, Timestamp: ts},
		},
	}

	data, err := RenderJSON(web)
	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}

	var output jsonOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(output.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(output.Pages))
	}
	if len(output.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(output.Errors))
	}
	if output.Errors[0].Error != "timeout" {
		t.Fatalf("expected error 'timeout', got %q", output.Errors[0].Error)
	}
}

func TestRenderJSON_IncludesStats(t *testing.T) {
	web := &weaver.Web{
		OriginURL: "https://example.com/",
		Duration:  time.Second,
		Pages: []weaver.PageResult{
			{URL: "/a", Status: 200, ContentType: "text/html", Duration: 10 * time.Millisecond, Timestamp: time.Now()},
			{URL: "/b", Status: 200, ContentType: "text/html", Duration: 20 * time.Millisecond, Timestamp: time.Now()},
		},
	}

	data, err := RenderJSON(web)
	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}

	var output jsonOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if output.Stats == nil {
		t.Fatal("expected stats in output")
	}
	if output.Stats.StatusCodes[200] != 2 {
		t.Fatalf("expected status 200 count=2, got %d", output.Stats.StatusCodes[200])
	}
	if output.Stats.ContentTypes["text/html"] != 2 {
		t.Fatalf("expected text/html count=2, got %d", output.Stats.ContentTypes["text/html"])
	}
}

func TestRenderJSON_EmptyWeb(t *testing.T) {
	web := &weaver.Web{OriginURL: "https://example.com/"}

	data, err := RenderJSON(web)
	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}

	var output jsonOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if output.Pages != nil {
		t.Fatalf("expected nil pages, got %v", output.Pages)
	}
	if output.Errors != nil {
		t.Fatalf("expected nil errors, got %v", output.Errors)
	}
}
