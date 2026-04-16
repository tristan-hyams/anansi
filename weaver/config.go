package weaver

import (
	"errors"
	"time"

	"golang.org/x/time/rate"

	"github.com/tristan-hyams/anansi/robots"
)

// WeaverConfig holds crawler-specific configuration, decoupled from CLI flags.
type WeaverConfig struct {
	Workers          int
	Rate             float64
	MaxDepth         int
	Timeout          time.Duration
	BufferSize       int
	UserAgent        string
	ProgressInterval int  // URLs processed before each crawler logs a progress checkpoint. 0 uses default (100).
	LogLinks         bool // Print each visited URL and its discovered links to the output writer.
}

// Validate checks that Config values are sane.
func (c *WeaverConfig) Validate() error {
	if c.Workers < 1 {
		return errors.New("workers must be at least 1")
	}

	if c.Rate < 1 {
		return errors.New("rate must be greater than 0")
	}

	if c.MaxDepth < 0 {
		return errors.New("max-depth cannot be negative (0 = unlimited)")
	}

	if c.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}

	if c.UserAgent == "" {
		c.UserAgent = defaultUserAgent
	}

	if c.ProgressInterval <= 0 {
		c.ProgressInterval = defaultProgressInterval
	}

	return nil
}

// CrawlRate returns the effective rate limit, respecting robots.txt
// Crawl-delay if it's stricter than the configured rate.
func (c *WeaverConfig) CrawlRate(rules *robots.Rules) rate.Limit {
	configuredRate := rate.Limit(c.Rate)

	if rules == nil {
		return configuredRate
	}

	delay := rules.CrawlDelay()
	if delay <= 0 {
		return configuredRate
	}

	delayRate := rate.Every(delay)
	if delayRate < configuredRate {
		return delayRate
	}

	return configuredRate
}
