# weaver

Orchestration layer for the Anansi web crawler. The Weaver manages the crawl - it owns the frontier, rate limiter, robots rules, and spawns Crawlers (workers) that fetch and parse pages.

Named after Anansi the spider: the weaver weaves the web, and crawlers venture out to fetch pages.

## Public API

```go
wv, err := weaver.NewWeaver(ctx, cfg, originURL, logger)
web, err := wv.Weave(ctx)

// web.Visited, web.Skipped, web.Duration, web.Pages - plain data
// Rendering and file output live in the fileutil package.
```

## Architecture

- **Weaver** - orchestrator. Owns config, frontier, rate limiter, robots rules. Pre-creates Crawlers during construction.
- **Crawler** - worker. Each has a UUID ID and logs start/stop events. Fetches pages, checks headers, parses links, enqueues new URLs. Runs in its own goroutine with its own `http.Client` backed by the shared `webutil.Transport()` singleton. Parse errors are recorded on `PageResult`.
- **Web** - crawl result data. Contains visited/skipped counts, duration, and per-page results.

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
| `result.go` | `Web` and `PageResult` - crawl result data structs |
| `consts.go` | Defaults, log keys, error sentinels |

## Termination

| Condition | Trigger | Behavior |
|-----------|---------|----------|
| Natural completion | `frontier.IsDone()` - pending counter at 0 and queue empty | Monitor cancels crawl context |
| Signal interrupt | Parent context cancelled (SIGINT/SIGTERM) | Crawlers exit via `ctx.Done()` |
