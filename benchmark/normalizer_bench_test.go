package benchmark

import (
	"net/url"
	"testing"

	"github.com/tristan-hyams/anansi/normalizer"
)

var benchBase, _ = url.Parse("https://crawlme.monzo.com/blog/")

func BenchmarkNormalize_Relative(b *testing.B) {
	for b.Loop() {
		normalizer.Normalize(benchBase, "../products/123")
	}
}

func BenchmarkNormalize_Absolute(b *testing.B) {
	for b.Loop() {
		normalizer.Normalize(benchBase, "https://crawlme.monzo.com/about")
	}
}

func BenchmarkNormalize_Fragment(b *testing.B) {
	for b.Loop() {
		normalizer.Normalize(benchBase, "/page#section")
	}
}

func BenchmarkNormalize_QueryParams(b *testing.B) {
	for b.Loop() {
		normalizer.Normalize(benchBase, "/search?q=test&page=2&lang=en")
	}
}

func BenchmarkIsSameHost_Match(b *testing.B) {
	origin, _ := url.Parse("https://crawlme.monzo.com/")
	candidate, _ := url.Parse("https://crawlme.monzo.com/about")
	for b.Loop() {
		normalizer.IsSameHost(origin, candidate)
	}
}

func BenchmarkIsSameHost_NoMatch(b *testing.B) {
	origin, _ := url.Parse("https://crawlme.monzo.com/")
	candidate, _ := url.Parse("https://community.monzo.com/forum")
	for b.Loop() {
		normalizer.IsSameHost(origin, candidate)
	}
}
