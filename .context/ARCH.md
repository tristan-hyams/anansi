# Architecture - Anansi

Design decisions, trade-offs, and rationale for the web crawler.

---

## Overview

Anansi is a single-domain web crawler. Given a starting URL, it visits every reachable page on the same subdomain, printing each URL visited and the links found on that page.

Built as a take-home challenge demonstrating: concurrency design, software structure, testing strategy, and production awareness.

---

## Concurrency Model

**Worker pool** with configurable concurrency (default: 1 worker, configurable via `-workers`).

```
                    ┌──────────┐
   Enqueue ───────► │ Buffered │ ──────► Worker 1 ──► HTTP GET ──► Parse ──► Enqueue new URLs
                    │ Channel  │ ──────► Worker 2 ──► ...
                    │ (queue)  │ ──────► Worker N ──► ...
                    └──────────┘
                         ▲
                         │
                   Rate Limiter
                  (token bucket)
```

- **Buffered channel** acts as the work queue. Buffer size defaults to 100,000 - the visited set (not the channel) is the real bound on growth. Dedup is built into Enqueue via `sync.Map`.
- **`FrontierURL`** wraps each URL with crawl depth. Depth is set by the Weaver at enqueue time.
- **Pending counter** (`atomic.Int32` in frontier) tracks work: incremented on `Enqueue`, decremented via `Done()` after processing. `IsDone()` checks `pending <= 0 AND queue empty` for deterministic completion detection.
- **`sync.WaitGroup`** in `Weaver.Weave()` tracks Crawler goroutines. Pre-created Crawlers are launched via `wg.Go()`. Each Crawler has a UUID ID and logs start/stop events for lifecycle visibility.
- **`golang.org/x/time/rate` token bucket** enforces global rate limiting (default: 1 req/s). All Crawlers draw a token before making HTTP requests. Respects `Crawl-delay` from robots.txt if stricter.
- **`signal.NotifyContext`** for graceful shutdown on SIGINT/SIGTERM. Cancels context, Crawlers drain, frontier select cases unblock via ctx.Done().

### Production note

The buffered channel is functionally equivalent to a message queue. In a production system, replace with RabbitMQ/Redis streams for restart durability. The `Frontier` interface enables this swap without changing crawler logic.

---

## Termination

Two distinct code paths:

| Condition | Trigger | Behaviour |
|---|---|---|
| Natural completion | `frontier.IsDone()` returns true (pending counter at 0, queue empty) | Monitor cancels crawl context, Crawlers exit, summary written |
| Signal interrupt | SIGINT/SIGTERM via `signal.NotifyContext` | Cancel context, Crawlers exit via `ctx.Done()`, partial results written |

---

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

---

## Domain Scoping

**Strict hostname match.** `crawlme.monzo.com` does not match `www.crawlme.monzo.com` or `monzo.com`.

Rationale: the spec says "limited to one subdomain." Strict matching is the safe interpretation. A comment in code notes that `www.` handling could be a configurable option.

---

## Content-Type Handling

Check `Content-Type` response header before parsing. Only parse `text/html` (with charset suffix tolerance via `strings.HasPrefix`). Non-HTML resources (PDF, images, JSON) are logged as visited but not parsed for links.

---

## Robot Compliance

Two complementary mechanisms:

### robots.txt (once at crawl start)

Fetched from `{scheme}://{host}/robots.txt`. Parsed for `User-agent: *` Disallow rules. URLs checked against rules before enqueuing.

Uses `github.com/temoto/robotstxt` for parsing — not a crawler framework.

`Crawl-delay` directive exposed as `time.Duration` for the rate limiter.

Graceful degradation:
- **404:** no rules, allow all.
- **403:** treated as not found. CDN/static hosts (S3, CloudFront, Azure Blob, GCS) often return 403 for missing files instead of 404 because the bucket doesn't allow public listing.
- **5xx / other 4xx:** returned as error to the caller.
- **Network error:** allow all with warning log.

Request includes `User-Agent: Anansi` and `Accept: text/plain` headers.

### X-Robots-Tag (per page response)

Checked on every HTTP response via the `X-Robots-Tag` header. Relevant directives:
- **`nofollow`** or **`none`**: skip link extraction for this page.
- **`noindex`**: not relevant for crawling (search engine indexing concern).
- **`follow`** or absent: proceed normally.

Example from crawlme.monzo.com: `X-Robots-Tag: noindex,follow` — don't index, but do follow links.

---

## Scope Boundaries (Explicit Non-Goals)

| Feature | Status | Rationale |
|---|---|---|
| JavaScript-rendered content (SPAs) | Out of scope | Would require headless browser (chromedp). Documented in README. |
| Persistent queue (RabbitMQ/Redis) | Interface only | In-memory impl for take-home. Interface enables swap. |
| Distributed crawling | Not implemented | Single-process design. Architecture supports future sharding via frontier partitioning. |
| Sitemap.xml parsing | Not implemented | Spec doesn't require it. Would be a complementary URL discovery source. |
| Depth limiting | Configurable | Default uncapped. Flag `--max-depth` available. Dedup via seen-set prevents infinite loops regardless. |

---

## Output Rendering

Crawl results (`Web` struct) are plain data. All rendering — markdown summary, JSON output, error logs, statistics — lives in the `fileutil` package, not in the weaver. This separation keeps the orchestrator focused on crawling and makes output formats independently testable.

Output files are written by `fileutil.WriteOutputFiles()`, called from the CLI after the crawl completes.
