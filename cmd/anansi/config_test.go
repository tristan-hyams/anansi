package main_test

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cli "github.com/tristan-hyams/anansi/cmd/anansi"
)

func TestSeedURL_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		seed   string
		wantH  string
		wantS  string
	}{
		{"https", "https://crawlme.monzo.com/", "crawlme.monzo.com", "https"},
		{"http", "http://example.com/path", "example.com", "http"},
		{"with port", "https://localhost:8080/", "localhost:8080", "https"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &cli.AnansiConfig{Seed: tt.seed}
			u, err := cfg.SeedURL()
			require.NoError(t, err)
			assert.Equal(t, tt.wantH, u.Host)
			assert.Equal(t, tt.wantS, u.Scheme)
		})
	}
}

func TestSeedURL_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		seed string
	}{
		{"empty", ""},
		{"no scheme", "crawlme.monzo.com"},
		{"no host", "https://"},
		{"just path", "/some/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &cli.AnansiConfig{Seed: tt.seed}
			u, err := cfg.SeedURL()
			assert.Error(t, err)
			assert.Nil(t, u)
		})
	}
}

func TestSlogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
		{"trace", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			cfg := &cli.AnansiConfig{LogLevel: tt.input}
			assert.Equal(t, tt.want, cfg.SlogLevel())
		})
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	t.Parallel()

	original := &cli.AnansiConfig{
		Workers:  10,
		Rate:     2.5,
		MaxDepth: 3,
		Timeout:  15 * time.Second,
		Seed:     "https://example.com/",
		LogLevel: "debug",
	}

	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, original.SaveToFile(path))

	loaded, err := cli.LoadConfigFromFile(path)
	require.NoError(t, err)

	assert.Equal(t, original.Workers, loaded.Workers)
	assert.Equal(t, original.Rate, loaded.Rate)
	assert.Equal(t, original.MaxDepth, loaded.MaxDepth)
	assert.Equal(t, original.Timeout, loaded.Timeout)
	assert.Equal(t, original.Seed, loaded.Seed)
	assert.Equal(t, original.LogLevel, loaded.LogLevel)
}

func TestLoadConfigFromFile_NotFound(t *testing.T) {
	t.Parallel()

	cfg, err := cli.LoadConfigFromFile("/nonexistent/config.json")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoadConfigFromFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("{not json}"), 0o644))

	cfg, err := cli.LoadConfigFromFile(path)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestSaveToFile_WritesValidJSON(t *testing.T) {
	t.Parallel()

	cfg := &cli.AnansiConfig{
		Workers:  5,
		Rate:     5,
		Seed:     "https://example.com/",
		LogLevel: "info",
	}

	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, cfg.SaveToFile(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, json.Valid(data), "SaveToFile produced invalid JSON: %s", data)
}
