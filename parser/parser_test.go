package parser_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tristan-hyams/anansi/parser"
)

func TestExtractLinks_WellFormed(t *testing.T) {
	t.Parallel()

	f, err := os.Open("testdata/well_formed.html")
	require.NoError(t, err)
	defer f.Close()

	links, err := parser.ExtractLinks(context.Background(), f)
	require.NoError(t, err)

	want := []string{
		"/about",
		"/contact",
		"https://example.com/external",
		"/page?q=test&lang=en",
		"#section",
		"mailto:user@example.com",
		"javascript:void(0)",
		"../relative",
	}
	assert.Equal(t, want, links)
}

func TestExtractLinks_Malformed(t *testing.T) {
	t.Parallel()

	f, err := os.Open("testdata/malformed.html")
	require.NoError(t, err)
	defer f.Close()

	links, err := parser.ExtractLinks(context.Background(), f)
	require.NoError(t, err)

	// Tokenizer should recover and extract what it can.
	assert.NotEmpty(t, links)
}

func TestExtractLinks_NoLinks(t *testing.T) {
	t.Parallel()

	f, err := os.Open("testdata/no_links.html")
	require.NoError(t, err)
	defer f.Close()

	links, err := parser.ExtractLinks(context.Background(), f)
	require.NoError(t, err)
	assert.Empty(t, links)
}

func TestExtractLinks_MixedTags(t *testing.T) {
	t.Parallel()

	f, err := os.Open("testdata/mixed_tags.html")
	require.NoError(t, err)
	defer f.Close()

	links, err := parser.ExtractLinks(context.Background(), f)
	require.NoError(t, err)

	// Only <a href> tags, not <link>, <script>, <img>, or <a> without href.
	assert.Equal(t, []string{"/real-link", "/another-link"}, links)
}

func TestExtractLinks_EmptyReader(t *testing.T) {
	t.Parallel()

	links, err := parser.ExtractLinks(context.Background(), strings.NewReader(""))
	require.NoError(t, err)
	assert.Empty(t, links)
}

func TestExtractLinks_NoHrefAttribute(t *testing.T) {
	t.Parallel()

	html := `<a name="anchor">Named</a><a class="btn">Button</a>`
	links, err := parser.ExtractLinks(context.Background(), strings.NewReader(html))
	require.NoError(t, err)
	assert.Empty(t, links)
}

func TestExtractLinks_HTMLEntities(t *testing.T) {
	t.Parallel()

	html := `<a href="/search?q=a&amp;b=c">Link</a>`
	links, err := parser.ExtractLinks(context.Background(), strings.NewReader(html))
	require.NoError(t, err)

	// Tokenizer decodes &amp; to &
	require.Len(t, links, 1)
	assert.Equal(t, "/search?q=a&b=c", links[0])
}

func TestExtractLinks_VariousHrefTypes(t *testing.T) {
	t.Parallel()

	html := `
		<a href="https://example.com/absolute">Absolute</a>
		<a href="/root-relative">Root</a>
		<a href="relative/path">Relative</a>
		<a href="../parent">Parent</a>
		<a href="//cdn.example.com/proto">Protocol Relative</a>
		<a href="#fragment">Fragment</a>
		<a href="">Empty</a>
		<a href="tel:+1234567890">Phone</a>
	`

	links, err := parser.ExtractLinks(context.Background(), strings.NewReader(html))
	require.NoError(t, err)

	want := []string{
		"https://example.com/absolute",
		"/root-relative",
		"relative/path",
		"../parent",
		"//cdn.example.com/proto",
		"#fragment",
		"",
		"tel:+1234567890",
	}
	assert.Equal(t, want, links)
}

func TestExtractLinks_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	html := `<a href="/one">One</a><a href="/two">Two</a>`
	links, err := parser.ExtractLinks(ctx, strings.NewReader(html))

	assert.ErrorIs(t, err, context.Canceled)
	// No tokens processed - context was already cancelled before iteration.
	assert.Empty(t, links)
}
