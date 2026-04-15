package main

import (
	"fmt"
	"os"

	"github.com/tristan-hyams/anansi/weaver"
)

func main() {

	cfg, err := ParseFlags()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, errFmt, err)
		os.Exit(exitCodeError)
	}

	logger := SetupLogger(cfg)
	ctx, cancel := SetupSignalContext()
	defer cancel()

	origin, err := cfg.OriginURL()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, errFmt, err)
		os.Exit(exitCodeError)
	}

	weaverCfg := &weaver.WeaverConfig{
		Workers:   cfg.Workers,
		Rate:      cfg.Rate,
		MaxDepth:  cfg.MaxDepth,
		Timeout:   cfg.Timeout,
		UserAgent: "Anansi",
	}

	logger.Info(
		fmt.Sprintf("crawl starting for [%s]", cfg.Origin),
		"origin", cfg.Origin,
		"workers", cfg.Workers,
		"rate", cfg.Rate,
		"max_depth", cfg.MaxDepth,
		"timeout", cfg.Timeout,
	)

	wv, err := weaver.NewWeaver(ctx, weaverCfg, origin, logger)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, errFmt, err)
		os.Exit(exitCodeError)
	}

	web, err := wv.Weave(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, errFmt, err)
		os.Exit(exitCodeError)
	}

	if err := writeOutputFiles(web); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, errFmt, err)
		os.Exit(exitCodeError)
	}

	// Short terminal summary — full report is in the files.
	_, _ = fmt.Fprintf(os.Stderr,
		"\ncrawl complete: %d pages crawled, %d skipped, %s\n",
		web.Visited, web.Skipped, web.Duration.Round(summaryDurationRound),
	)

	if ctx.Err() != nil {
		os.Exit(exitCodeSIGINT)
	}
}

func writeOutputFiles(web *weaver.Web) error {
	// Markdown summary.
	if err := os.WriteFile(outputResultsFile, []byte(web.String()), 0o644); err != nil {
		return fmt.Errorf("writing results: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stderr, "results written to %s\n", outputResultsFile)

	// JSON output.
	jsonData, err := web.JSON()
	if err != nil {
		return fmt.Errorf("generating JSON: %w", err)
	}

	if err := os.WriteFile(outputJSONFile, jsonData, 0o644); err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stderr, "json written to %s\n", outputJSONFile)

	// Error log (only if errors occurred).
	errorLog := web.ErrorLog()
	if errorLog != "" {
		if err := os.WriteFile(outputErrorsFile, []byte(errorLog), 0o644); err != nil {
			return fmt.Errorf("writing errors: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stderr, "errors written to %s\n", outputErrorsFile)
	}

	return nil
}
