package benchmark

import (
	"fmt"
	"testing"
	"time"

	"github.com/tristan-hyams/anansi/fileutil"
	"github.com/tristan-hyams/anansi/weaver"
)

func generatePages(n int) []weaver.PageResult {
	pages := make([]weaver.PageResult, n)
	for i := range pages {
		pages[i] = weaver.PageResult{
			URL:         fmt.Sprintf("https://example.com/page/%d", i),
			Links:       10,
			Depth:       i % 5,
			Status:      200,
			ContentType: "text/html",
			Duration:    time.Duration(50+i%100) * time.Millisecond,
			Timestamp:   time.Now(),
		}
	}
	return pages
}

func BenchmarkComputeStats_1k(b *testing.B) {
	web := &weaver.Web{Pages: generatePages(1_000)}
	b.ResetTimer()
	for b.Loop() {
		fileutil.ComputeStats(web)
	}
}

func BenchmarkComputeStats_10k(b *testing.B) {
	web := &weaver.Web{Pages: generatePages(10_000)}
	b.ResetTimer()
	for b.Loop() {
		fileutil.ComputeStats(web)
	}
}
