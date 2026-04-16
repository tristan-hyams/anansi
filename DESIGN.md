# Design

Design decisions, trade-offs, and rationale for the Anansi web crawler.

## Concurrency Model

**Worker pool** with configurable concurrency (default: 1 worker, configurable via `-workers`).

```
                    ┌──────────┐
   Enqueue ───────► │ Buffered │ ──────► Crawler 1 ──► HTTP GET ──► Parse ──► Enqueue new URLs
                    │ Channel  │ ──────► Crawler 2 ──► ...
                    │ (queue)  │ ──────► Crawler N ──► ...
                    └──────────┘
                         ▲
                         │
                   Rate Limiter
                  (token bucket)
```

- **Buffered channel** acts as the work queue. Buffer defaults to 100,000 — the visited set (not the channel) is the real bound on growth.
- **Pending counter** (`atomic.Int32` in frontier) tracks in-flight work. Incremented on `Enqueue`, decremented via `Done()` after processing. `IsDone()` checks `pending <= 0 AND queue empty` for deterministic completion.
- **Pre-created Crawlers** — each has a UUID ID, its own `http.Client` backed by a shared `Transport` singleton, and logs start/stop/progress events for lifecycle visibility.
- **Token bucket rate limiter** (`golang.org/x/time/rate`) enforces global request throughput. Respects `Crawl-delay` from robots.txt if stricter.
- **Graceful shutdown** via `signal.NotifyContext` — SIGINT/SIGTERM cancels the context, Crawlers drain via `ctx.Done()`.

### Why a worker pool, not goroutine-per-URL

Predictable resource usage. With goroutine-per-URL, a site with 50,000 pages spawns 50,000 goroutines and potentially 50,000 open connections. A worker pool with N workers means N goroutines, N connections, bounded memory. The rate limiter further constrains throughput regardless of worker count.

### Production swap

The buffered channel is functionally a message queue. In production, replace `InMemory` with Redis/RabbitMQ via the `Frontier` interface — no crawler changes needed.

## Page Fetch Pipeline

```
Dequeue URL from frontier
  │
  ├─ Rate limiter wait
  │
  ├─ HTTP GET
  │
  ├─ Check Content-Type ──► only parse text/html
  │
  ├─ Check X-Robots-Tag header ──► if nofollow or none, skip link extraction
  │
  ├─ Parse HTML for <a href> links (parse errors recorded on PageResult)
  │
  └─ For each discovered href:
       │
       ├─ Parse with net/url ──► reject if error
       │
       ├─ Normalize ──► strip fragment, lowercase host, resolve relative, strip default port
       │
       ├─ Filter scheme ──► only http/https (reject mailto:, javascript:, tel:, etc.)
       │
       ├─ Filter domain ──► strict hostname match against origin URL
       │
       ├─ Check robots.txt ──► reject if Disallowed
       │
       └─ Enqueue to frontier (dedup built in)
```

## Domain Scoping

**Strict hostname match.** `crawlme.monzo.com` does not match `www.crawlme.monzo.com` or `monzo.com`.

The spec says "limited to one subdomain." Strict matching is the safe interpretation — `www.` equivalence could be a configurable option but is not assumed.

## Termination

| Condition | Trigger | Behaviour |
|---|---|---|
| Natural completion | `frontier.IsDone()` returns true (pending counter at 0, queue empty) | Monitor cancels crawl context, Crawlers exit, summary written |
| Signal interrupt | SIGINT/SIGTERM via `signal.NotifyContext` | Cancel context, Crawlers exit via `ctx.Done()`, partial results written |

## Robot Compliance

### robots.txt (once at crawl start)

Fetched from `{scheme}://{host}/robots.txt`. URLs checked against `User-agent: *` Disallow rules before enqueuing. `Crawl-delay` directive feeds into the rate limiter.

Graceful degradation:
- **404:** no rules, allow all.
- **403:** treated as not found — CDN/static hosts (S3, CloudFront, Azure Blob, GCS) return 403 for missing files because the bucket doesn't allow public listing.
- **5xx / other 4xx:** returned as error.
- **Network error:** allow all with warning.

### X-Robots-Tag (per page response)

Checked on every HTTP response. `nofollow` or `none` skips link extraction. No additional request — the header is on the GET response the crawler already makes.

## Output

### Real-time (stdout)

When `-log-links` is enabled (default), each visited page and its discovered links are printed to stdout as they are crawled:

```
https://crawlme.monzo.com/
  https://crawlme.monzo.com/about
  https://crawlme.monzo.com/blog
  https://crawlme.monzo.com/products
```

Disable with `-log-links=false` for quieter operation during development.

### Files (written after crawl)

| File | Contents |
|------|----------|
| `crawl-results.md` | Spider banner, latency stats (P50/P95/P99), status codes, content-type breakdown, page list, sitemap tree |
| `crawl-results.json` | Machine-readable JSON with per-page found links, pipeable to `jq` |
| `crawl-errors.md` | Errors grouped by reason, each URL timestamped (only if errors occurred) |

Rendering is separated from the crawl orchestrator — the `fileutil` package converts plain `Web` result data into all output formats.

## Design Trade-offs

| Decision | Trade-off |
|----------|-----------|
| Worker pool over goroutine-per-URL | Predictable resource usage, but requires tuning `-workers` for throughput |
| Interface-based frontier | Swappable backend (Redis/RabbitMQ), but adds indirection for an in-memory-only implementation |
| Strict subdomain matching | Safe interpretation of spec, but `www.` equivalence requires explicit opt-in |
| Single shared rate limiter | Simple global throttle, but doesn't allow per-path or adaptive rate limiting |
| Buffered channel as queue | Simple, fast, in-process — but no restart durability or distributed fanout |
| robots.txt 403 as not found | Pragmatic for CDN hosts, but could mask genuine access restrictions |
| Normalize before dedup | Prevents visiting `/About` and `/about` twice, but normalizer must be correct or the crawler misses pages |

## Rejected Patterns

| Pattern | Why |
|---------|-----|
| `go-colly` / `scrapy` | Spec requires own implementation |
| Headless browser (chromedp) | Out of scope — server-rendered HTML only |
| Unbounded goroutines | Unpredictable resource usage |
| `fmt.Println` for output | Structured logging (`slog`) with key-value pairs |
| `internal/` packages | Packages are public by design — reusable as library imports |
| Docker Compose / multi-service | Single static binary. Dockerfile is for reviewer convenience |

## Known Limitations

- **JavaScript-rendered content (SPAs):** Processes server-rendered HTML only. JS-rendered content would require a headless browser.
- **No sitemap.xml parsing:** Would be a complementary URL discovery source alongside link extraction.
- **No distributed crawling:** Single-process design. The `Frontier` interface supports future sharding via swappable backend.
