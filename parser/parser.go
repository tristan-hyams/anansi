// Package parser extracts raw href values from HTML anchor tags.
// It uses the golang.org/x/net/html tokenizer for robust handling
// of malformed HTML. No filtering or normalization is performed —
// that is the caller's responsibility.
package parser

import (
	"bytes"
	"context"
	"io"

	"golang.org/x/net/html"
)

// ExtractLinks scans HTML from r and returns raw href attribute values
// from all <a> tags. Tags without an href attribute are skipped.
// The returned hrefs are unfiltered and unnormalized.
//
// The context is checked each iteration so callers can cancel parsing
// of large or buffered documents. Partial results are returned alongside
// the context error.
//
// The html.Tokenizer is a state machine — avoid concurrency or parallelism
// for token iteration. Parallelism belongs in the crawler (multiple workers
// parsing different pages), not within a single page parse.
//
// For parallelism within a single page parse, we would need to perhaps go
// a different route with a full DOM parser and read through partitions of
// the page.
func ExtractLinks(ctx context.Context, r io.Reader) ([]string, error) {
	tokenizer := html.NewTokenizer(r)

	var links []string

	for {
		if err := ctx.Err(); err != nil {
			return links, err
		}

		tokenType := tokenizer.Next()

		switch tokenType {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				return links, nil
			}
			return links, tokenizer.Err()

		// SelfClosingTagToken handles <a href="/page"/> which the tokenizer
		// emits as a separate type from StartTagToken. The odds of this are
		// low but again, trying to be robust/technically correct.
		case html.StartTagToken, html.SelfClosingTagToken:
			if !isAHrefToken(tokenizer) {
				continue
			}

			if href, ok := getHrefValue(tokenizer); ok {
				links = append(links, href)
			}

		default:
			continue
		}
	}
}

func isAHrefToken(tokenizer *html.Tokenizer) bool {
	name, hasAttr := tokenizer.TagName()
	return len(name) == 1 && name[0] == 'a' && hasAttr
}

// bytes.Equal is used for efficient byte slice comparison without unnecessary string conversions.
var hrefKey = []byte("href")

// getHrefValue iterates attributes of the current token looking for href.
func getHrefValue(tokenizer *html.Tokenizer) (string, bool) {
	for {
		key, val, more := tokenizer.TagAttr()
		if bytes.Equal(key, hrefKey) {
			return string(val), true
		}
		if !more {
			return "", false
		}
	}
}
