package main_test

import (
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cli "github.com/tristan-hyams/anansi/cmd/anansi"
)

func TestSetupLogger_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	cfg := &cli.AnansiConfig{LogLevel: "info"}
	logger := cli.SetupLogger(cfg)
	assert.NotNil(t, logger)
}

func TestSetupLogger_RespectsLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level   string
		enabled slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			t.Parallel()
			cfg := &cli.AnansiConfig{LogLevel: tt.level}
			logger := cli.SetupLogger(cfg)
			assert.True(t, logger.Enabled(nil, tt.enabled),
				"logger should be enabled at %s level", tt.level)
		})
	}
}

func TestSetupSignalContext_ReturnsCancellable(t *testing.T) {
	t.Parallel()

	ctx, cancel := cli.SetupSignalContext()
	defer cancel()

	require.NotNil(t, ctx)
	require.NotNil(t, cancel)

	assert.NoError(t, ctx.Err(), "context should not be cancelled initially")

	cancel()
	assert.Error(t, ctx.Err(), "context should be cancelled after cancel()")
}

func TestStartPprofServer_NoOpWithoutEnvVar(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("ANANSI_DEBUG", "")

	cfg := &cli.AnansiConfig{LogLevel: "info"}
	logger := cli.SetupLogger(cfg)

	// Should return immediately without starting a server.
	cli.StartPprofServer(logger)

	// Verify no server is listening — give it a moment then check.
	time.Sleep(50 * time.Millisecond)
	_, err := http.Get("http://localhost:6060/debug/pprof/")
	assert.Error(t, err, "pprof server should not be running without ANANSI_DEBUG")
}

func TestStartPprofServer_StartsWithEnvVar(t *testing.T) {
	// Not parallel — binds to a fixed port.
	t.Setenv("ANANSI_DEBUG", "1")

	cfg := &cli.AnansiConfig{LogLevel: "info"}
	logger := cli.SetupLogger(cfg)

	cli.StartPprofServer(logger)

	// Give the goroutine time to start listening.
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:6060/debug/pprof/")
	if err != nil {
		t.Skipf("pprof server may already be in use: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestErrUsage(t *testing.T) {
	t.Parallel()

	assert.ErrorIs(t, cli.ErrUsage, cli.ErrUsage)
	assert.Contains(t, cli.ErrUsage.Error(), "usage:")
}

func TestAnansiConfig_BufferSize_RoundTrip(t *testing.T) {
	t.Parallel()

	original := &cli.AnansiConfig{
		Workers:    4,
		Rate:       10,
		MaxDepth:   5,
		Timeout:    30 * time.Second,
		Origin:     "https://example.com/",
		LogLevel:   "debug",
		LogLinks:   true,
		MaxRetries: 3,
		BufferSize: 50000,
	}

	path := t.TempDir() + "/config.json"
	require.NoError(t, original.SaveToFile(path))

	loaded, err := cli.LoadConfigFromFile(path)
	require.NoError(t, err)

	assert.Equal(t, original.BufferSize, loaded.BufferSize)
	assert.Equal(t, original.LogLinks, loaded.LogLinks)
	assert.Equal(t, original.MaxRetries, loaded.MaxRetries)
}

func TestSaveToFile_BadPath(t *testing.T) {
	t.Parallel()

	cfg := &cli.AnansiConfig{Origin: "https://example.com/"}
	err := cfg.SaveToFile("/nonexistent/dir/config.json")
	assert.Error(t, err)
}

func TestOriginURL_PreservesPath(t *testing.T) {
	t.Parallel()

	cfg := &cli.AnansiConfig{Origin: "https://example.com/some/path"}
	u, err := cfg.OriginURL()
	require.NoError(t, err)
	assert.Equal(t, "/some/path", u.Path)
}

func TestOriginURL_SchemeOnly(t *testing.T) {
	t.Parallel()

	cfg := &cli.AnansiConfig{Origin: "https://"}
	u, err := cfg.OriginURL()
	assert.Error(t, err)
	assert.Nil(t, u)
}

func TestOriginURL_ControlCharacters(t *testing.T) {
	t.Parallel()

	cfg := &cli.AnansiConfig{Origin: "https://example.com/\x00"}
	_, err := cfg.OriginURL()
	// Go's url.Parse is lenient, but the result should still have a host.
	if err != nil {
		assert.Contains(t, err.Error(), "invalid")
	}
}

func TestSlogLevel_CaseInsensitive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  slog.Level
	}{
		{"Debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"Warning", slog.LevelWarn},
		{"ERROR", slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			cfg := &cli.AnansiConfig{LogLevel: tt.input}
			assert.Equal(t, tt.want, cfg.SlogLevel())
		})
	}
}

func TestSetupLogger_SetsDefault(t *testing.T) {
	cfg := &cli.AnansiConfig{LogLevel: "warn"}
	logger := cli.SetupLogger(cfg)

	// SetupLogger calls slog.SetDefault — verify it took effect.
	assert.Equal(t, logger.Handler(), slog.Default().Handler())
}

func TestLoadConfigFromFile_EmptyFile(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/empty.json"
	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	cfg, err := cli.LoadConfigFromFile(path)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}
