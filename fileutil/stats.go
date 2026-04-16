package fileutil

import (
	"slices"
	"time"

	"github.com/tristan-hyams/anansi/weaver"
)

// Stats holds aggregated crawl statistics computed from PageResults.
type Stats struct {
	StatusCodes  map[int]int    `json:"status_codes"`
	ContentTypes map[string]int `json:"content_types"`
	Latency      LatencyStats   `json:"latency"`
}

// LatencyStats holds response time percentiles.
type LatencyStats struct {
	Avg time.Duration `json:"avg"`
	P50 time.Duration `json:"p50"`
	P95 time.Duration `json:"p95"`
	P99 time.Duration `json:"p99"`
	Min time.Duration `json:"min"`
	Max time.Duration `json:"max"`
}

// ComputeStats aggregates statistics from the Web's page results.
// Only includes pages that were actually fetched (have a status code).
func ComputeStats(web *weaver.Web) *Stats {
	statusCodes := make(map[int]int)
	contentTypes := make(map[string]int)
	var durations []time.Duration

	for _, p := range web.Pages {
		if p.Status > 0 {
			statusCodes[p.Status]++
		}

		if p.ContentType != "" {
			contentTypes[p.ContentType]++
		}

		// Only include pages that were fetched for latency stats.
		if p.Status > 0 && p.Duration > 0 {
			durations = append(durations, p.Duration)
		}
	}

	return &Stats{
		StatusCodes:  statusCodes,
		ContentTypes: contentTypes,
		Latency:      computeLatency(durations),
	}
}

func computeLatency(durations []time.Duration) LatencyStats {
	if len(durations) == 0 {
		return LatencyStats{}
	}

	slices.Sort(durations)

	n := len(durations)
	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return LatencyStats{
		Avg: total / time.Duration(n),
		P50: durations[pct50*n/pct100],
		P95: durations[min(pct95*n/pct100, n-1)],
		P99: durations[min(pct99*n/pct100, n-1)],
		Min: durations[0],
		Max: durations[n-1],
	}
}
