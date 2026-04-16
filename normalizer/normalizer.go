// Package normalizer provides URL canonicalization for the Anansi web crawler.
// All functions are pure - no I/O, no shared state, no side effects.
package normalizer

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Normalize resolves raw against base, then canonicalizes the result.
// Strips fragments, lowercases scheme and host, resolves relative URLs,
// and removes default ports (:80 for HTTP, :443 for HTTPS).
//
// Trailing slashes are intentionally NOT normalized. Per RFC 3986 Section 6,
// /path and /path/ are distinct URIs. While most servers return identical
// content for both, the rendered response is entirely up to the origin server.
// We treat them as distinct for technical accuracy while fully acknowledging
// they may be semantically equivalent. Future enhancements may include optional
// trailing slash normalization with configurable heuristics.
func Normalize(base *url.URL, raw string) (*url.URL, error) {
	if raw == "" {
		return nil, errors.New("normalizing URL: empty href")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("normalizing URL %q: %w", raw, err)
	}

	resolved := base.ResolveReference(parsed)

	resolved.Scheme = strings.ToLower(resolved.Scheme)
	resolved.Host = strings.ToLower(resolved.Host)
	resolved.Fragment = ""

	resolved.Host = stripDefaultPort(resolved.Scheme, resolved.Host)

	return resolved, nil
}

// IsSameHost returns true if candidate has the exact same hostname as origin.
// Strict match - crawlme.monzo.com does not match monzo.com or community.monzo.com.
func IsSameHost(origin, candidate *url.URL) bool {
	return stripDefaultPort(origin.Scheme, strings.ToLower(origin.Host)) ==
		stripDefaultPort(candidate.Scheme, strings.ToLower(candidate.Host))
}

// IsFollowableScheme returns true if u uses http or https.
func IsFollowableScheme(u *url.URL) bool {
	s := strings.ToLower(u.Scheme)
	return s == "http" || s == "https"
}

// stripDefaultPort removes :80 from HTTP hosts and :443 from HTTPS hosts.
// Uses net.SplitHostPort for safe parsing - no string suffix ambiguity.
func stripDefaultPort(scheme, host string) string {
	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		return host // no port present
	}

	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		return hostname
	}

	return host
}
