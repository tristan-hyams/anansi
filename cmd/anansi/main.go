package main

import (
	"fmt"
	"os"
)

func main() {

	cfg, err := ParseFlags()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "anansi: %v\n", err)
		os.Exit(1)
	}

	logger := SetupLogger(cfg)
	ctx, cancel := SetupSignalContext()
	defer cancel()

	// TODO: wire crawler.New(cfg, logger) → crawler.Run(ctx) once crawler package exists
	logger.Info(
		fmt.Sprintf("crawl starting for [%s]", cfg.Origin),
		"origin", cfg.Origin,
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
