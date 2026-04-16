package main

import (
	"fmt"
	"io"
	"os"

	"github.com/tristan-hyams/anansi/reporting"
	"github.com/tristan-hyams/anansi/weaver"
)

func fatal(err error) {
	_, _ = fmt.Fprintf(os.Stderr, errFmt, err)
	//revive:disable-next-line:deep-exit CLI helper in main package.
	os.Exit(exitCodeError)
}

func main() {

	cfg, err := ParseFlags()
	if err != nil {
		fatal(err)
	}

	logger := SetupLogger(cfg)
	StartPprofServer(logger)

	ctx, cancel := SetupSignalContext()
	defer cancel()

	origin, err := cfg.OriginURL()
	if err != nil {
		fatal(err)
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

	outputDir, err := reporting.CreateOutputDir()
	if err != nil {
		fatal(err)
	}

	_, _ = fmt.Fprintf(os.Stderr, "results will be written to %s\n", outputDir)

	logger.Info(
		fmt.Sprintf("crawl starting for [%s]", cfg.Origin),
		"origin", cfg.Origin,
		"workers", cfg.Workers,
		"rate", cfg.Rate,
		"max_depth", cfg.MaxDepth,
		"timeout", cfg.Timeout,
		"output_dir", outputDir,
	)

	wv, err := weaver.NewWeaver(ctx, weaverCfg, origin, logger, output)
	if err != nil {
		fatal(err)
	}

	web := wv.Weave(ctx)

	if err := reporting.WriteOutputFiles(web, outputDir, os.Stderr); err != nil {
		fatal(err)
	}

	_, _ = fmt.Fprintf(os.Stderr,
		"\ncrawl complete: %d pages crawled, %d skipped, %s\n",
		web.Visited, web.Skipped, web.Duration.Round(summaryDurationRound),
	)

	if ctx.Err() != nil {
		os.Exit(exitCodeSIGINT)
	}
}
