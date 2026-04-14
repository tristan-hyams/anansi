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

func SetupSignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

func SetupLogger(cfg *AnansiConfig) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: cfg.SlogLevel(),
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
