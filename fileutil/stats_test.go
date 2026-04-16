package fileutil

import (
	"errors"
	"testing"
	"time"

	"github.com/tristan-hyams/anansi/weaver"
)

func TestComputeStats_Empty(t *testing.T) {
	web := &weaver.Web{}
	stats := ComputeStats(web)

	if len(stats.StatusCodes) != 0 {
		t.Fatalf("expected empty status codes, got %v", stats.StatusCodes)
	}
	if len(stats.ContentTypes) != 0 {
		t.Fatalf("expected empty content types, got %v", stats.ContentTypes)
	}
	if stats.Latency != (LatencyStats{}) {
		t.Fatalf("expected zero latency, got %+v", stats.Latency)
	}
}

func TestComputeStats_CountsStatusCodes(t *testing.T) {
	web := &weaver.Web{
		Pages: []weaver.PageResult{
			{URL: "/a", Status: 200, Duration: 10 * time.Millisecond},
			{URL: "/b", Status: 200, Duration: 20 * time.Millisecond},
			{URL: "/c", Status: 404, Duration: 5 * time.Millisecond},
		},
	}

	stats := ComputeStats(web)

	if stats.StatusCodes[200] != 2 {
		t.Fatalf("expected 200 count=2, got %d", stats.StatusCodes[200])
	}
	if stats.StatusCodes[404] != 1 {
		t.Fatalf("expected 404 count=1, got %d", stats.StatusCodes[404])
	}
}

func TestComputeStats_CountsContentTypes(t *testing.T) {
	web := &weaver.Web{
		Pages: []weaver.PageResult{
			{URL: "/a", ContentType: "text/html", Status: 200, Duration: time.Millisecond},
			{URL: "/b", ContentType: "text/html", Status: 200, Duration: time.Millisecond},
			{URL: "/c", ContentType: "application/json", Status: 200, Duration: time.Millisecond},
			{URL: "/d"}, // no content type — skipped/errored
		},
	}

	stats := ComputeStats(web)

	if stats.ContentTypes["text/html"] != 2 {
		t.Fatalf("expected text/html count=2, got %d", stats.ContentTypes["text/html"])
	}
	if stats.ContentTypes["application/json"] != 1 {
		t.Fatalf("expected application/json count=1, got %d", stats.ContentTypes["application/json"])
	}
	if len(stats.ContentTypes) != 2 {
		t.Fatalf("expected 2 content types, got %d", len(stats.ContentTypes))
	}
}

func TestComputeStats_SkipsErroredPagesForLatency(t *testing.T) {
	web := &weaver.Web{
		Pages: []weaver.PageResult{
			{URL: "/ok", Status: 200, Duration: 100 * time.Millisecond},
			{URL: "/err", Error: errors.New("fail"), Duration: 50 * time.Millisecond}, // Status=0
		},
	}

	stats := ComputeStats(web)

	// Only 1 page has Status > 0 and Duration > 0.
	if stats.Latency.Min != 100*time.Millisecond {
		t.Fatalf("expected min=100ms, got %v", stats.Latency.Min)
	}
	if stats.Latency.Max != 100*time.Millisecond {
		t.Fatalf("expected max=100ms, got %v", stats.Latency.Max)
	}
}

func TestComputeLatency_Percentiles(t *testing.T) {
	// 100 durations: 1ms, 2ms, ..., 100ms
	durations := make([]time.Duration, 100)
	for i := range durations {
		durations[i] = time.Duration(i+1) * time.Millisecond
	}

	lat := computeLatency(durations)

	if lat.Min != 1*time.Millisecond {
		t.Fatalf("expected min=1ms, got %v", lat.Min)
	}
	if lat.Max != 100*time.Millisecond {
		t.Fatalf("expected max=100ms, got %v", lat.Max)
	}
	if lat.P50 != 51*time.Millisecond {
		t.Fatalf("expected P50=51ms, got %v", lat.P50)
	}
	if lat.P95 != 96*time.Millisecond {
		t.Fatalf("expected P95=96ms, got %v", lat.P95)
	}
	if lat.P99 != 100*time.Millisecond {
		t.Fatalf("expected P99=100ms, got %v", lat.P99)
	}
	if lat.Avg != 50500*time.Microsecond { // (1+2+...+100)/100 = 50.5ms
		t.Fatalf("expected avg=50.5ms, got %v", lat.Avg)
	}
}

func TestComputeLatency_SingleDuration(t *testing.T) {
	durations := []time.Duration{42 * time.Millisecond}
	lat := computeLatency(durations)

	if lat.Min != 42*time.Millisecond || lat.Max != 42*time.Millisecond {
		t.Fatalf("expected min=max=42ms, got min=%v max=%v", lat.Min, lat.Max)
	}
	if lat.P50 != 42*time.Millisecond {
		t.Fatalf("expected P50=42ms, got %v", lat.P50)
	}
	if lat.Avg != 42*time.Millisecond {
		t.Fatalf("expected avg=42ms, got %v", lat.Avg)
	}
}

func TestComputeLatency_Empty(t *testing.T) {
	lat := computeLatency(nil)
	if lat != (LatencyStats{}) {
		t.Fatalf("expected zero LatencyStats, got %+v", lat)
	}
}
