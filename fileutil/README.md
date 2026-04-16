# fileutil

Rendering and file output for the Anansi web crawler. Converts `weaver.Web` crawl results into markdown, JSON, and error reports. Writes output files to disk.

Extracted from the weaver package to keep the orchestrator focused on crawling. All functions are pure transforms on `*weaver.Web` data - no I/O except `WriteOutputFiles`.

## Public API

```go
// Rendering - pure functions, no side effects.
md := fileutil.RenderMarkdown(web)     // Markdown with banner, stats, page list, sitemap tree
js, err := fileutil.RenderJSON(web)    // Indented JSON, pipeable to jq
errLog := fileutil.RenderErrorLog(web) // Errors grouped by reason (empty if no errors)
stats := fileutil.ComputeStats(web)    // Latency P50/P95/P99, status codes, content types

// File output - writes to current directory.
err := fileutil.WriteOutputFiles(web, os.Stderr)
```

## Output Files

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
| `writer.go` | `WriteOutputFiles()` - writes all output files to disk |
| `consts.go` | Banner, formatting constants, output filenames |
