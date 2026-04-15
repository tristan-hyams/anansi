package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
)

// ErrUsage indicates the CLI was invoked with incorrect arguments.
var ErrUsage = errors.New("usage: anansi [flags] <url>")

// ParseFlags registers CLI flags, parses them, and returns a validated config.
func ParseFlags() (*AnansiConfig, error) {

	workers := flag.Int("workers", defaultWorkers, "number of concurrent workers")
	rate := flag.Float64("rate", defaultRate, "max requests per second")
	maxDepth := flag.Int("max-depth", defaultMaxDepth, "maximum crawl depth (0 = unlimited)")
	timeout := flag.Duration("timeout", defaultTimeout, "HTTP request timeout")
	logLevel := flag.String("log-level", defaultLogLevel, "log level (debug, info, warn, error)")

	flag.Usage = func() {
		_, _ = fmt.Fprint(os.Stderr, "Usage: anansi [flags] <url>\n\nFlags:\n")
		flag.PrintDefaults()
	}

	//revive:disable-next-line:deep-exit Sole CLI entry point, flag.Parse may os.Exit(2)
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
		Origin:   flag.Arg(0),
		LogLevel: *logLevel,
	}

	if _, err := cfg.OriginURL(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SetupSignalContext returns a context that is cancelled on SIGINT (Ctrl+C).
func SetupSignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

// SetupLogger configures slog with a JSON handler and the specified log level.
func SetupLogger(cfg *AnansiConfig) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: cfg.SlogLevel(),
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
