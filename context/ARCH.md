# Architecture - Anansi

Design decisions, trade-offs, and rationale for the web crawler.

---

## Overview

Anansi is a single-domain web crawler. Given a starting URL, it visits every reachable page on the same subdomain, printing each URL visited and the links found on that page.

Built as a take-home challenge demonstrating: concurrency design, software structure, testing strategy, and production awareness.

---

## Concurrency Model

**Worker pool** with configurable concurrency (default: 5 workers).

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

- **Buffered channel** acts as the work queue. Buffer size configurable (default: 1000). Provides natural backpressure — workers discovering new URLs block on send if the buffer is full.
- **`sync.WaitGroup`** tracks in-flight work. `Add(1)` on enqueue, `Done()` on completion. A monitor goroutine calls `wg.Wait()` then closes the channel to signal natural completion.
- **`golang.org/x/time/rate` token bucket** enforces global rate limiting (default: 5 req/s). All workers draw a token before making HTTP requests.
- **`signal.NotifyContext`** for graceful shutdown on SIGINT/SIGTERM. Cancels context, workers drain, logger flushes.

### Production note

The buffered channel is functionally equivalent to a message queue. In a production system, replace with RabbitMQ/Redis streams for restart durability. The `Frontier` interface enables this swap without changing crawler logic.

---

## Termination

Two distinct code paths:

| Condition | Trigger | Behaviour |
|---|---|---|
| Natural completion | WaitGroup counter reaches zero (all workers idle + empty queue) | Clean exit, print summary |
| Signal interrupt | SIGINT/SIGTERM via `signal.NotifyContext` | Cancel context, drain workers, log discarded URLs, flush logger |

---

## URL Processing Pipeline

```
Raw href from HTML
  │
  ├─ Parse with net/url ──► reject if error
  │
  ├─ Normalize ──► strip fragment, lowercase host, resolve relative, strip default port
  │
  ├─ Filter scheme ──► only http/https (reject mailto:, javascript:, tel:, etc.)
  │
  ├─ Filter domain ──► strict hostname match against seed URL
  │
  ├─ Check robots.txt ──► reject if Disallowed
  │
  ├─ Check seen-set ──► reject if already visited
  │
  └─ Enqueue to frontier
```

---

## Domain Scoping

**Strict hostname match.** `crawlme.monzo.com` does not match `www.crawlme.monzo.com` or `monzo.com`.

Rationale: the spec says "limited to one subdomain." Strict matching is the safe interpretation. A comment in code notes that `www.` handling could be a configurable option.

---

## Content-Type Handling

Check `Content-Type` response header before parsing. Only parse `text/html` (with charset suffix tolerance via `strings.HasPrefix`). Non-HTML resources (PDF, images, JSON) are logged as visited but not parsed for links.

---

## robots.txt

Fetched once at crawl start from `{scheme}://{host}/robots.txt`. Parsed for `User-agent: *` Disallow rules. URLs checked against rules before enqueuing.

Uses `github.com/temoto/robotstxt` for parsing — robots.txt parsing is not a crawler framework.

Optional: `Crawl-delay` directive fed into rate limiter if present.

---

## Scope Boundaries (Explicit Non-Goals)

| Feature | Status | Rationale |
|---|---|---|
| JavaScript-rendered content (SPAs) | Out of scope | Would require headless browser (chromedp). Documented in README. |
| Persistent queue (RabbitMQ/Redis) | Interface only | In-memory impl for take-home. Interface enables swap. |
| Distributed crawling | Not implemented | Single-process design. Architecture supports future sharding via frontier partitioning. |
| Sitemap.xml parsing | Not implemented | Spec doesn't require it. Would be a complementary URL discovery source. |
| Depth limiting | Configurable | Default uncapped. Flag `--max-depth` available. Dedup via seen-set prevents infinite loops regardless. |
