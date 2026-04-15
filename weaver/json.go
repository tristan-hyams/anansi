package weaver

import (
	"encoding/json"
	"fmt"
	"time"
)

// jsonOutput is the top-level JSON structure for machine-readable crawl results.
type jsonOutput struct {
	Origin   string            `json:"origin"`
	Visited  int               `json:"visited"`
	Skipped  int               `json:"skipped"`
	Duration string            `json:"duration"`
	Stats    *jsonStats        `json:"stats"`
	Pages    []jsonPageResult  `json:"pages"`
	Errors   []jsonError       `json:"errors,omitempty"`
}

type jsonStats struct {
	StatusCodes  map[int]int    `json:"status_codes"`
	ContentTypes map[string]int `json:"content_types"`
	Latency      jsonLatency    `json:"latency"`
}

type jsonLatency struct {
	Avg string `json:"avg"`
	P50 string `json:"p50"`
	P95 string `json:"p95"`
	P99 string `json:"p99"`
	Min string `json:"min"`
	Max string `json:"max"`
}

type jsonPageResult struct {
	URL         string `json:"url"`
	Links       int    `json:"links"`
	Depth       int    `json:"depth"`
	Status      int    `json:"status"`
	ContentType string `json:"content_type,omitempty"`
	Duration    string `json:"duration"`
	Timestamp   string `json:"timestamp"`
}

type jsonError struct {
	URL       string `json:"url"`
	Depth     int    `json:"depth"`
	Error     string `json:"error"`
	Timestamp string `json:"timestamp"`
}

// JSON returns the crawl results as indented JSON bytes.
func (w *Web) JSON() ([]byte, error) {

	stats := w.ComputeStats()

	var pages []jsonPageResult
	var errors []jsonError

	for _, p := range w.Pages {
		ts := p.Timestamp.Format(time.RFC3339)

		if p.Error != nil {
			errors = append(errors, jsonError{
				URL:       p.URL,
				Depth:     p.Depth,
				Error:     p.Error.Error(),
				Timestamp: ts,
			})
			continue
		}

		pages = append(pages, jsonPageResult{
			URL:         p.URL,
			Links:       p.Links,
			Depth:       p.Depth,
			Status:      p.Status,
			ContentType: p.ContentType,
			Duration:    p.Duration.Round(time.Millisecond).String(),
			Timestamp:   ts,
		})
	}

	output := jsonOutput{
		Origin:   w.OriginURL,
		Visited:  w.Visited,
		Skipped:  w.Skipped,
		Duration: w.Duration.Round(summaryDurationRound).String(),
		Stats: &jsonStats{
			StatusCodes:  stats.StatusCodes,
			ContentTypes: stats.ContentTypes,
			Latency: jsonLatency{
				Avg: stats.Latency.Avg.Round(time.Millisecond).String(),
				P50: stats.Latency.P50.Round(time.Millisecond).String(),
				P95: stats.Latency.P95.Round(time.Millisecond).String(),
				P99: stats.Latency.P99.Round(time.Millisecond).String(),
				Min: stats.Latency.Min.Round(time.Millisecond).String(),
				Max: stats.Latency.Max.Round(time.Millisecond).String(),
			},
		},
		Pages:  pages,
		Errors: errors,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling JSON output: %w", err)
	}

	return data, nil
}
