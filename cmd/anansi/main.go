package main

import (
	"fmt"
	"io"
	"os"

	"github.com/tristan-hyams/anansi/fileutil"
	"github.com/tristan-hyams/anansi/weaver"
)

func main() {

	cfg, err := ParseFlags()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, errFmt, err)
		os.Exit(exitCodeError)
	}

	logger := SetupLogger(cfg)
	StartPprofServer(logger)

	ctx, cancel := SetupSignalContext()
	defer cancel()

	origin, err := cfg.OriginURL()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, errFmt, err)
		os.Exit(exitCodeError)
	}

	weaverCfg := weaver.NewWeaverConfig(weaver.WeaverConfig{
		Workers:     cfg.Workers,
		Rate:        cfg.Rate,
		MaxDepth:    cfg.MaxDepth,
		Timeout:     cfg.Timeout,
		LogLinks:    cfg.LogLinks,
		MaxRetries:  cfg.MaxRetries,
		MaxDuration: cfg.MaxDuration,
		BufferSize:  cfg.BufferSize,
	})

	var output io.Writer = os.Stdout
	if !cfg.LogLinks {
		output = io.Discard
	}

	logger.Info(
		fmt.Sprintf("crawl starting for [%s]", cfg.Origin),
		"origin", cfg.Origin,
		"workers", cfg.Workers,
		"rate", cfg.Rate,
		"max_depth", cfg.MaxDepth,
		"timeout", cfg.Timeout,
	)

	wv, err := weaver.NewWeaver(ctx, weaverCfg, origin, logger, output)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, errFmt, err)
		os.Exit(exitCodeError)
	}

	web, err := wv.Weave(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, errFmt, err)
		os.Exit(exitCodeError)
	}

	if err := fileutil.WriteOutputFiles(web, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, errFmt, err)
		os.Exit(exitCodeError)
	}

	// Short terminal summary - full report is in the files.
	_, _ = fmt.Fprintf(os.Stderr,
		"\ncrawl complete: %d pages crawled, %d skipped, %s\n",
		web.Visited, web.Skipped, web.Duration.Round(summaryDurationRound),
	)

	if ctx.Err() != nil {
		os.Exit(exitCodeSIGINT)
	}
}
