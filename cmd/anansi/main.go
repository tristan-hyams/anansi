package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

func setupSignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

func main() {

	cfg, err := ParseFlags()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "anansi: %v\n", err)
		os.Exit(1)
	}

	logger := setupLogger(cfg)
	ctx, stop := setupSignalContext()
	defer stop()

	// TODO: wire crawler.New(cfg, logger) → crawler.Run(ctx) once crawler package exists
	logger.Info("crawl starting",
		"seed", cfg.Seed,
		"workers", cfg.Workers,
		"rate", cfg.Rate,
		"max_depth", cfg.MaxDepth,
		"timeout", cfg.Timeout,
		"log_level", cfg.LogLevel,
	)

	_ = ctx
	_, _ = fmt.Fprintln(os.Stderr, "anansi: crawler not yet implemented")
	os.Exit(1)
}
