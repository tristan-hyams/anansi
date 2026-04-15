package robots

import (
	"net/http"
	"strings"
)

// Directives represents parsed X-Robots-Tag directives from an HTTP response.
// These are per-page instructions that complement robots.txt.
//
// Reference: https://developers.google.com/search/docs/crawling-indexing/robots-meta-tag
type Directives struct {
	NoIndex  bool
	NoFollow bool
}

// ShouldFollow returns true if the crawler is allowed to follow links
// from this page. Returns false if nofollow or none is set.
func (d *Directives) ShouldFollow() bool {
	return !d.NoFollow
}

// ParseXRobotsTag extracts directives from the X-Robots-Tag HTTP header.
// The header value is a comma-separated list of directives, e.g.:
//
//	X-Robots-Tag: noindex, follow
//	X-Robots-Tag: none
//	X-Robots-Tag: nofollow
//
// An absent header or unrecognised directives result in permissive defaults
// (index and follow allowed).
func ParseXRobotsTag(header http.Header) *Directives {
	raw := header.Get(xRobotsTagHeader)
	if raw == "" {
		return &Directives{}
	}

	d := &Directives{}

	for _, part := range strings.Split(raw, ",") {
		directive := strings.TrimSpace(strings.ToLower(part))

		switch directive {
		case "noindex":
			d.NoIndex = true
		case "nofollow":
			d.NoFollow = true
		case "none":
			d.NoIndex = true
			d.NoFollow = true
		default:
			continue
		}
	}

	return d
}
