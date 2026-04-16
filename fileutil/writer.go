package fileutil

import (
	"fmt"
	"io"
	"os"

	"github.com/tristan-hyams/anansi/weaver"
)

// WriteOutputFiles writes the crawl results to disk:
//   - crawl-results.md  — markdown summary with stats and sitemap
//   - crawl-results.json — machine-readable JSON
//   - crawl-errors.md   — error report (only if errors occurred)
//
// Status messages are written to w (typically os.Stderr).
func WriteOutputFiles(web *weaver.Web, w io.Writer) error {
	// Markdown summary.
	if err := os.WriteFile(outputResultsFile, []byte(RenderMarkdown(web)), 0o644); err != nil {
		return fmt.Errorf("writing results: %w", err)
	}

	_, _ = fmt.Fprintf(w, "results written to %s\n", outputResultsFile)

	// JSON output.
	jsonData, err := RenderJSON(web)
	if err != nil {
		return fmt.Errorf("generating JSON: %w", err)
	}

	if err := os.WriteFile(outputJSONFile, jsonData, 0o644); err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}

	_, _ = fmt.Fprintf(w, "json written to %s\n", outputJSONFile)

	// Error log (only if errors occurred).
	errorLog := RenderErrorLog(web)
	if errorLog != "" {
		if err := os.WriteFile(outputErrorsFile, []byte(errorLog), 0o644); err != nil {
			return fmt.Errorf("writing errors: %w", err)
		}

		_, _ = fmt.Fprintf(w, "errors written to %s\n", outputErrorsFile)
	}

	return nil
}
