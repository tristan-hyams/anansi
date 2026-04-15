# weaver

Orchestration layer for the Anansi web crawler. The Weaver manages the crawl — it owns the frontier, rate limiter, robots rules, and spawns Crawlers (workers) that fetch and parse pages.

Named after Anansi the spider: the weaver weaves the web, and crawlers venture out to fetch pages.

## Public API

```go
wv, err := weaver.NewWeaver(ctx, cfg, originURL, logger)
result, err := wv.Weave(ctx)
```

## Architecture

- **Weaver** — orchestrator. Owns config, frontier, rate limiter, robots rules, HTTP client. Spawns and monitors crawlers.
- **Crawler** — worker. Fetches pages, checks headers, parses links, enqueues new URLs. Each runs in its own goroutine.

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
| `config.go` | `Config` struct with validation |
| `result.go` | `Result` and `PageResult` structs |
| `consts.go` | Defaults and log key constants |

## Termination

| Condition | Trigger | Behavior |
|-----------|---------|----------|
| Natural completion | Active counter reaches 0, queue empty | Monitor cancels crawl context |
| Signal interrupt | Parent context cancelled (SIGINT/SIGTERM) | Crawlers exit via `ctx.Done()` |
