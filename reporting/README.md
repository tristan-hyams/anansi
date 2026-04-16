# reporting

Crawl report rendering and output for the Anansi web crawler. Converts `weaver.Web` crawl results into markdown, JSON, and error reports. Writes output files to a unique per-run directory.

Separated from the weaver package to keep the orchestrator focused on crawling. All rendering functions are pure transforms on `*weaver.Web` data.

## Public API

```go
// Output directory - UUIDv7 for chronological sorting.
dir, err := reporting.CreateOutputDir()  // creates ./output/<uuidv7>/

// Rendering - pure functions, no side effects.
md := reporting.RenderMarkdown(web)      // Markdown with banner, stats, page list, sitemap tree
js, err := reporting.RenderJSON(web)     // Indented JSON, pipeable to jq
errLog := reporting.RenderErrorLog(web)  // Errors grouped by reason (empty if no errors)
stats := reporting.ComputeStats(web)     // Latency P50/P95/P99, status codes, content types

// File output - writes to the given directory.
err := reporting.WriteOutputFiles(web, dir, os.Stderr)
```

## Output Files

Each crawl writes to `./output/<uuidv7>/`:

| File | Contents |
|------|----------|
| `crawl-results.md` | Spider banner, latency stats, status codes, content-type breakdown, page list, sitemap tree |
| `crawl-results.json` | Machine-readable JSON: same data as markdown |
| `crawl-errors.md` | Errors grouped by reason, each URL timestamped (only if errors occurred) |

## Files

| File | Purpose |
|------|---------|
| `markdown.go` | `RenderMarkdown()`, `RenderErrorLog()`, sitemap tree builder |
| `json.go` | `RenderJSON()`, JSON struct types |
| `stats.go` | `ComputeStats()`, `Stats`, `LatencyStats`, percentile math |
| `writer.go` | `CreateOutputDir()`, `WriteOutputFiles()` - output directory and file I/O |
| `consts.go` | Banner, formatting constants, output filenames |
