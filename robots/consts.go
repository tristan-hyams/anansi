package robots

import "time"

const (
	logKeyURL = "url"

	// userAgent identifies Anansi in robots.txt requests per RFC 9309.
	userAgent = "Anansi"

	// fetchTimeout is the HTTP timeout for the robots.txt request.
	fetchTimeout = 10 * time.Second

	// xRobotsTagHeader is the HTTP header for per-page robot directives.
	xRobotsTagHeader = "X-Robots-Tag"

	// maxRobotsBodySize caps the bytes read from a robots.txt response.
	// robots.txt files are typically a few KB; 1 MB is very generous.
	maxRobotsBodySize int64 = 1 << 20 // 1 MB
)
