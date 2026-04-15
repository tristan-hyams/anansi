# weaver

Orchestration layer for the Anansi web crawler. The Weaver manages the crawl — it owns the frontier, rate limiter, robots rules, and spawns Crawlers (workers) that fetch and parse pages.

Named after Anansi the spider: the weaver weaves the web, and crawlers venture out to fetch pages.

## Public API

```go
wv, err := weaver.NewWeaver(ctx, cfg, originURL, logger)
web, err := wv.Weave(ctx)
fmt.Print(web.String())
```

## Architecture

- **Weaver** — orchestrator. Owns config, frontier, rate limiter, robots rules. Pre-creates Crawlers during construction.
- **Crawler** — worker. Fetches pages, checks headers, parses links, enqueues new URLs. Each runs in its own goroutine with its own `http.Client` backed by the shared `webutil.Transport()` singleton.
- **Web** — crawl result with `String()` method for formatted summary output. Sorted by depth, then alphabetically.

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
| `result.go` | `Web` and `PageResult` structs, `Web.String()` summary formatting |
| `consts.go` | Defaults, log keys, summary formatting constants |

## Termination

| Condition | Trigger | Behavior |
|-----------|---------|----------|
| Natural completion | `active == 0 AND queue.Len() == 0` | Monitor cancels crawl context |
| Signal interrupt | Parent context cancelled (SIGINT/SIGTERM) | Crawlers exit via `ctx.Done()` |
