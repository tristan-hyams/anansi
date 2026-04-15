package weaver

import (
	"slices"
	"time"
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
func (w *Web) ComputeStats() *Stats {
	statusCodes := make(map[int]int)
	contentTypes := make(map[string]int)
	var durations []time.Duration

	for _, p := range w.Pages {
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

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return LatencyStats{
		Avg: total / time.Duration(len(durations)),
		P50: percentile(durations, pct50),
		P95: percentile(durations, pct95),
		P99: percentile(durations, pct99),
		Min: durations[0],
		Max: durations[len(durations)-1],
	}
}

func percentile(sorted []time.Duration, pct int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	idx := (pct * len(sorted)) / pct100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}
