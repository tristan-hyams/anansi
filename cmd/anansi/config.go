package main

import (
	"encoding/json"
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
	Origin   string        `json:"origin"`
	LogLevel string        `json:"log_level"`
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

// OriginURL parses and validates the Origin field as a *url.URL.
func (c *AnansiConfig) OriginURL() (*url.URL, error) {
	u, err := url.Parse(c.Origin)
	if err != nil {
		return nil, fmt.Errorf("parsing origin URL: %w", err)
	}

	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("origin URL %q missing scheme or host", c.Origin)
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
