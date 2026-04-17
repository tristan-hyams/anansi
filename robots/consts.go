package robots

import "time"

const (
	logKeyURL = "url"

	// fetchTimeout is the HTTP timeout for the robots.txt request.
	fetchTimeout = 10 * time.Second

	// xRobotsTagHeader is the HTTP header for per-page robot directives.
	xRobotsTagHeader = "X-Robots-Tag"

	// maxRobotsBodySize caps the bytes read from a robots.txt response.
	// robots.txt files are typically a few KB; 1 MB is very generous.
	maxRobotsBodySize int64 = 1 << 20 // 1 MB

	// TODO: Duplicate consts, better centralized in a shared package with weaver.
	defaultUserAgent     = "Anansi Weaver; Go 1.26;"
	acceptHeader         = "text/html, */*;q=0.8"
	acceptLanguageHeader = "en"
)
