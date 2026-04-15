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

- **Buffered channel** acts as the work queue. Buffer size defaults to 100,000 - the visited set (not the channel) is the real bound on growth. Dedup is built into Enqueue via `sync.Map`.
- **`FrontierURL`** wraps each URL with crawl metadata: depth, status (pending/visited/error), and error state. The crawler sets depth at enqueue time.
- **`sync.WaitGroup`** lives in the crawler (not the frontier) to track in-flight workers. The frontier is a pure queue + dedup layer.
- **`golang.org/x/time/rate` token bucket** enforces global rate limiting (default: 5 req/s). All workers draw a token before making HTTP requests.
- **`signal.NotifyContext`** for graceful shutdown on SIGINT/SIGTERM. Cancels context, workers drain, frontier select cases unblock via ctx.Done().

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
  ├─ Parse HTML for <a href> links
  │
  └─ For each discovered href:
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
