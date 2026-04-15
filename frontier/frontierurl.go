package frontier

import "net/url"

// FrontierURL wraps a URL with crawl metadata.
type FrontierURL struct {
	URL    *url.URL
	Depth  int
	Status Status
	Err    error
}
