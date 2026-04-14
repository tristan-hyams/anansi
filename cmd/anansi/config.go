package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"
)

// AnansiConfig holds all crawler configuration. Serializable to/from JSON
// for file-based config or debugging.
type AnansiConfig struct {
	Workers  int           `json:"workers"`
	Rate     float64       `json:"rate"`
	MaxDepth int           `json:"max_depth"`
	Timeout  time.Duration `json:"timeout"`
	Seed     string        `json:"seed"`
	LogLevel string        `json:"log_level"`
}

// SeedURL parses and validates the Seed field as a *url.URL.
func (c *AnansiConfig) SeedURL() (*url.URL, error) {
	u, err := url.Parse(c.Seed)
	if err != nil {
		return nil, fmt.Errorf("parsing seed URL: %w", err)
	}

	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("seed URL %q missing scheme or host", c.Seed)
	}
	return u, nil
}

// SlogLevel converts the LogLevel string to a slog.Level.
// Defaults to Info for empty or unrecognised values.
func (c *AnansiConfig) SlogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// SaveToFile serializes the config as indented JSON to the given path.
func (c *AnansiConfig) SaveToFile(path string) error {

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config to %s: %w", path, err)
	}
	return nil
}

// LoadConfigFromFile reads a JSON config file and returns an AnansiConfig.
func LoadConfigFromFile(path string) (*AnansiConfig, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config from %s: %w", path, err)
	}

	var cfg AnansiConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config from %s: %w", path, err)
	}
	return &cfg, nil
}

// ErrUsage indicates the CLI was invoked with incorrect arguments.
var ErrUsage = errors.New("usage: anansi [flags] <url>")

// ParseFlags registers CLI flags, parses them, and returns a validated config.
func ParseFlags() (*AnansiConfig, error) {

	workers := flag.Int("workers", defaultWorkers, "number of concurrent workers")
	rate := flag.Float64("rate", defaultRate, "max requests per second")
	maxDepth := flag.Int("max-depth", 0, "maximum crawl depth (0 = unlimited)")
	timeout := flag.Duration("timeout", defaultTimeout, "HTTP request timeout")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")

	flag.Usage = func() {
		_, _ = fmt.Fprint(os.Stderr, "Usage: anansi [flags] <url>\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		return nil, ErrUsage
	}

	cfg := AnansiConfig{
		Workers:  *workers,
		Rate:     *rate,
		MaxDepth: *maxDepth,
		Timeout:  *timeout,
		Seed:     flag.Arg(0),
		LogLevel: *logLevel,
	}

	if _, err := cfg.SeedURL(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
