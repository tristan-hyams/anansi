package reporting

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/tristan-hyams/anansi/weaver"
)

// CreateOutputDir creates a unique output directory under ./output/.
// Uses UUIDv7 for chronological sorting and uniqueness.
func CreateOutputDir() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generating output dir ID: %w", err)
	}

	dir := filepath.Join("output", id.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating output dir: %w", err)
	}

	return dir, nil
}

// WriteOutputFiles writes the crawl results to dir:
//   - crawl-results.md  - markdown summary with stats and sitemap
//   - crawl-results.json - machine-readable JSON
//   - crawl-errors.md   - error report (only if errors occurred)
//
// Status messages are written to w (typically os.Stderr).
func WriteOutputFiles(web *weaver.Web, dir string, w io.Writer) error {
	resultsPath := filepath.Join(dir, outputResultsFile)
	if err := os.WriteFile(resultsPath, []byte(RenderMarkdown(web)), 0o644); err != nil {
		return fmt.Errorf("writing results: %w", err)
	}

	_, _ = fmt.Fprintf(w, "results written to %s\n", resultsPath)

	jsonPath := filepath.Join(dir, outputJSONFile)
	jsonData, err := RenderJSON(web)
	if err != nil {
		return fmt.Errorf("generating JSON: %w", err)
	}

	if err := os.WriteFile(jsonPath, jsonData, 0o644); err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}

	_, _ = fmt.Fprintf(w, "json written to %s\n", jsonPath)

	errorLog := RenderErrorLog(web)
	if errorLog != "" {
		errorsPath := filepath.Join(dir, outputErrorsFile)
		if err := os.WriteFile(errorsPath, []byte(errorLog), 0o644); err != nil {
			return fmt.Errorf("writing errors: %w", err)
		}

		_, _ = fmt.Fprintf(w, "errors written to %s\n", errorsPath)
	}

	return nil
}
