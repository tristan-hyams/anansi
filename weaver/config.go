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
	ProgressInterval int           // URLs between progress checkpoints. NewWeaverConfig defaults to 100.
	LogLinks         bool          // Print visited URLs and links to stdout.
	MaxRetries       int           // Retry attempts for transient errors. NewWeaverConfig defaults 0→2, -1=off.
	MaxRedirects     int           // Max redirect chain length. NewWeaverConfig defaults 0→10.
	MaxDuration      time.Duration // Max crawl duration. 0 = unlimited.
}

// NewWeaverConfig creates a WeaverConfig with defaults applied for
// zero-value optional fields. Call Validate() after to check for errors.
func NewWeaverConfig(c WeaverConfig) *WeaverConfig {
	if c.UserAgent == "" {
		c.UserAgent = defaultUserAgent
	}

	if c.ProgressInterval <= 0 {
		c.ProgressInterval = defaultProgressInterval
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = defaultMaxRetries
	}

	if c.MaxRedirects == 0 {
		c.MaxRedirects = defaultMaxRedirects
	}

	return &c
}

// Validate checks that Config values are sane. Does not mutate the receiver.
func (c *WeaverConfig) Validate() error {
	if c.Workers < 1 {
		return errors.New("workers must be at least 1")
	}

	if c.Rate <= 0 {
		return errors.New("rate must be greater than 0")
	}

	if c.MaxDepth < 0 {
		return errors.New("max-depth cannot be negative (0 = unlimited)")
	}

	if c.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}

	if c.UserAgent == "" {
		return errors.New("user agent must not be empty")
	}

	if c.ProgressInterval <= 0 {
		return errors.New("progress interval must be greater than 0")
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
