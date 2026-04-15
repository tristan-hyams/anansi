# weaver

Orchestration layer for the Anansi web crawler. The Weaver manages the crawl — it owns the frontier, rate limiter, robots rules, and spawns Crawlers (workers) that fetch and parse pages.

Named after Anansi the spider: the weaver weaves the web, and crawlers venture out to fetch pages.

## Public API

```go
wv, err := weaver.NewWeaver(ctx, cfg, originURL, logger)
web, err := wv.Weave(ctx)

// Outputs
fmt.Print(web.String())        // Markdown summary with stats + sitemap tree
jsonBytes, _ := web.JSON()     // Machine-readable JSON
errorLog := web.ErrorLog()     // Errors grouped by reason with timestamps
stats := web.ComputeStats()    // Latency P50/P95/P99, status codes, content types
```

## Architecture

- **Weaver** — orchestrator. Owns config, frontier, rate limiter, robots rules. Pre-creates Crawlers during construction.
- **Crawler** — worker. Fetches pages, checks headers, parses links, enqueues new URLs. Each runs in its own goroutine with its own `http.Client` backed by the shared `webutil.Transport()` singleton.
- **Web** — crawl result. `String()` renders markdown with spider banner, latency stats, sitemap tree. `JSON()` returns machine-readable output. `ErrorLog()` groups errors by reason with timestamps.

## Page Fetch Pipeline

```
Dequeue → max depth check → rate limit → HTTP GET → Content-Type check
  → X-Robots-Tag check → parse links → normalize/filter → enqueue
```

## Files

| File | Purpose |
|------|---------|
| `weaver.go` | `Weaver` struct, `NewWeaver()`, `Weave()`, monitor, result building |
| `crawler.go` | `Crawler` struct, page processing, link filtering, HTTP fetch |
| `config.go` | `WeaverConfig` with `Validate()`, `CrawlRate(rules)` |
| `result.go` | `Web` with `String()`, `ErrorLog()`, sitemap tree, `PageResult` |
| `json.go` | `Web.JSON()` — machine-readable JSON output |
| `stats.go` | `ComputeStats()` — latency P50/P95/P99, status codes, content types |
| `consts.go` | Defaults, log keys, summary formatting, spider banner |

## Termination

| Condition | Trigger | Behavior |
|-----------|---------|----------|
| Natural completion | `frontier.IsDone()` — pending counter at 0 and queue empty | Monitor cancels crawl context |
| Signal interrupt | Parent context cancelled (SIGINT/SIGTERM) | Crawlers exit via `ctx.Done()` |
