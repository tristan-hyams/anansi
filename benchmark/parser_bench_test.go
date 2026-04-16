package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/tristan-hyams/anansi/parser"
)

// smallHTML is a realistic page with 8 links (matches parser/testdata/well_formed.html).
var smallHTML = []byte(`<!DOCTYPE html>
<html>
<head><title>Small Page</title></head>
<body>
  <a href="/about">About</a>
  <a href="/contact">Contact</a>
  <a href="https://example.com/external">External</a>
  <a href="/page?q=test&amp;lang=en">Search</a>
  <a href="#section">Fragment</a>
  <a href="mailto:user@example.com">Email</a>
  <a href="javascript:void(0)">JS Link</a>
  <a href="../relative">Relative</a>
</body>
</html>`)

// largeHTML is generated once — 50KB page with 100 links.
var largeHTML []byte

func init() {
	largeHTML = generateLargeHTML(100)
}

func generateLargeHTML(linkCount int) []byte {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html><html><head><title>Large Page</title></head><body>\n")
	sb.WriteString("<h1>Product Catalog</h1>\n")

	for i := range linkCount {
		sb.WriteString(fmt.Sprintf(
			`<div class="product"><h2>Product %d</h2>`+
				`<p>Description of product %d with enough text to simulate real page content.</p>`+
				`<a href="/products/%d.html">View Product %d</a></div>`+"\n",
			i, i, i, i,
		))
	}

	// Add some non-link tags to simulate realistic HTML density.
	for range 50 {
		sb.WriteString("<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. ")
		sb.WriteString("Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</p>\n")
	}

	sb.WriteString("</body></html>")
	return []byte(sb.String())
}

func BenchmarkExtractLinks_Small(b *testing.B) {
	ctx := context.Background()
	for b.Loop() {
		parser.ExtractLinks(ctx, bytes.NewReader(smallHTML))
	}
}

func BenchmarkExtractLinks_Large100Links(b *testing.B) {
	ctx := context.Background()
	for b.Loop() {
		parser.ExtractLinks(ctx, bytes.NewReader(largeHTML))
	}
}
