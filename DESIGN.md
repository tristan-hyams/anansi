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

Disable with `-log-links=false` for a much quieter operation during development.

### Result Files (written after crawl)

| File | Contents |
|------|----------|
| `crawl-results.md` | Latency stats (P50/P95/P99), status codes, content-type breakdown, page list, sitemap tree |
| `crawl-results.json` | Machine-readable JSON with per-page found links, pipeable to `jq` |
| `crawl-errors.md` | Errors grouped by reason, each URL timestamped (only if errors occurred) |

Rendering is separated from the crawl orchestrator — the `fileutil` package converts plain `Web` result data into all output formats.

## Design Trade-offs

| Decision | Trade-off |
|----------|-----------|
| Worker pool over goroutine-per-URL | Predictable resource usage, but requires tuning `-workers` for throughput |
| Interface-based frontier | Swappable backend (Redis/RabbitMQ), but adds indirection for an in-memory-only implementation |
| Strict subdomain matching | Safe interpretation of spec, but aware of sub-sub domain *.*.*.company.com etc. issues. `www.` equivalence requires explicit opt-in |
| Single shared rate limiter | Simple global throttle, but doesn't allow per-path or adaptive rate limiting |
| Buffered channel as queue | Simple, fast, in-process — but no restart durability or distributed fanout |
| robots.txt 403 as not found | Pragmatic for CDN hosts, but could mask genuine access restrictions |
| Normalize before dedup | Prevents visiting `/About` and `/about` twice, but normalizer must be correct or the crawler misses pages |
| Retry/Exponential Backoff | Definitely easily configurable and should be added for transient network issues. | 

## Rejected Patterns

| Pattern | Why |
|---------|-----|
| `go-colly` / `scrapy` | Spec requires own implementation |
| Headless browser (chromedp) | Out of scope — server-rendered HTML only |
| Unbounded goroutines | Unpredictable resource usage |
| `fmt.Println` for output | Structured logging (`slog`) with key-value pairs |
| `internal/` packages | Packages are public by design — reusable as library imports |
| Docker Compose / multi-service | Single static binary. Dockerfile is for reviewer convenience |

## Profiling & Benchmarks

With more time, the following profiling and benchmarking infrastructure would be added. This section documents the approach and what each technique would target.

### pprof

Go's `net/http/pprof` package provides runtime profiling with zero application code changes beyond registering the HTTP handlers. In a production crawler, a debug HTTP server would run alongside the crawl on a separate port:

```go
go func() {
    // Debug server — pprof endpoints at /debug/pprof/
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

This exposes CPU profiles, heap snapshots, goroutine dumps, and block/mutex contention profiles — all queryable live during a crawl.

**What to profile and why:**

| Profile | Target | What it reveals |
|---------|--------|-----------------|
| CPU (`/debug/pprof/profile`) | `normalizer.Normalize`, `parser.ExtractLinks` | These run on every page. URL normalization involves `net/url.Parse`, `strings.ToLower`, `net.SplitHostPort` — many small allocations. The HTML tokenizer walks every byte. CPU profiling reveals whether the hot path is in our code or the standard library. |
| Heap (`/debug/pprof/heap`) | `frontier.InMemory`, `weaver.pages` | The visited `sync.Map` and the `[]PageResult` slice grow monotonically with crawl size. At 42k pages, each `PageResult` holds a URL string, found links slice, content-type string, and timestamps. Heap profiling quantifies actual memory per page and identifies whether the `FoundLinks []string` field (normalized URLs for display) is the dominant allocation. |
| Goroutine (`/debug/pprof/goroutine`) | Worker pool lifecycle | With N workers, we expect exactly N+2 goroutines during crawl (N crawlers + 1 monitor + 1 main). A goroutine dump during a stall would immediately reveal whether crawlers are blocked on the rate limiter, the frontier channel, or HTTP I/O. |
| Mutex (`/debug/pprof/mutex`) | `weaver.mu`, `weaver.outputMu` | Two mutexes protect shared state — one for the pages slice, one for stdout printing. Under high worker counts (50+), mutex contention could become measurable. Mutex profiling reveals whether the lock hold time justifies the separate-mutex design or whether a lock-free approach (e.g., per-worker result channels) would help. |
| Block (`/debug/pprof/block`) | `frontier.Dequeue`, `rate.Limiter.Wait` | Block profiling shows where goroutines spend time waiting. Expected: most blocking time in `Dequeue` (waiting for work) and `limiter.Wait` (rate limiting). Unexpected blocking in `recordPage` or `printPage` would signal contention. |

**Investigation workflow:**

```bash
# Start a crawl with pprof enabled
bin/anansi -workers 20 -rate 5000 -max-depth 0 https://crawlme.monzo.com/

# In another terminal — capture a 30-second CPU profile mid-crawl
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Heap snapshot — shows live allocations
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine dump — verify worker count, find stuck goroutines
curl http://localhost:6060/debug/pprof/goroutine?debug=2
```

The CPU profile would generate a flame graph showing time distribution across the page fetch pipeline: HTTP I/O, HTML tokenizing, URL normalization, frontier operations, and mutex contention.

### Benchmarks

Benchmarks would target the hot-path functions — the code that runs per-page or per-link during a crawl. With 42,000 pages averaging 10+ links each, these functions execute hundreds of thousands of times.

**Pure function benchmarks (no I/O):**

| Benchmark | Function | Why it matters |
|-----------|----------|----------------|
| `BenchmarkNormalize` | `normalizer.Normalize` | Called once per discovered link. Involves `url.Parse`, `ResolveReference`, `strings.ToLower`, `net.SplitHostPort`. At 10 links/page across 42k pages, this runs ~420k times per crawl. |
| `BenchmarkIsSameHost` | `normalizer.IsSameHost` | Called once per normalized link. String comparison after lowercasing and port stripping. |
| `BenchmarkExtractLinks` | `parser.ExtractLinks` | HTML tokenizer loop — sequential I/O bound. Benchmark with varying HTML sizes (1KB, 10KB, 100KB) to establish the relationship between page size and extraction time. |
| `BenchmarkParseXRobotsTag` | `robots.ParseXRobotsTag` | Called once per HTTP response. String splitting and directive parsing. Should be sub-microsecond. |
| `BenchmarkComputeStats` | `fileutil.ComputeStats` | Runs once at crawl end. With 42k PageResults, this sorts durations and iterates the full slice. Benchmark with 1k, 10k, 100k results to verify linear scaling. |

**Example benchmark implementation:**

```go
func BenchmarkNormalize(b *testing.B) {
    base, _ := url.Parse("https://crawlme.monzo.com/blog/")
    hrefs := []string{
        "/about",
        "../products/123",
        "https://crawlme.monzo.com/contact",
        "#section",
        "page?q=search&page=2",
    }
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        normalizer.Normalize(base, hrefs[i%len(hrefs)])
    }
}

func BenchmarkExtractLinks(b *testing.B) {
    html := loadTestHTML(b, "testdata/large_page.html") // 50KB page with 100 links
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        parser.ExtractLinks(context.Background(), bytes.NewReader(html))
    }
}
```

**Concurrency benchmarks:**

| Benchmark | Target | What it measures |
|-----------|--------|------------------|
| `BenchmarkFrontierEnqueue` | `frontier.Enqueue` with contention | N goroutines enqueueing simultaneously. Measures `sync.Map.LoadOrStore` and channel send throughput under contention. |
| `BenchmarkFrontierDequeueEnqueue` | Producer-consumer throughput | Simulates the crawl loop — M producers enqueueing, N consumers dequeuing. Measures overall frontier throughput independent of network I/O. |
| `BenchmarkRecordPage` | `weaver.recordPage` with contention | N goroutines appending PageResults concurrently. Measures mutex overhead on the shared pages slice. |

**Integration benchmark (httptest):**

A `BenchmarkWeave` using `httptest.NewServer` with a canned site of 100-1000 pages would measure end-to-end throughput without network variability. This isolates the crawler's overhead from server latency:

```go
func BenchmarkWeave_100Pages(b *testing.B) {
    srv := newCannedSite(100) // httptest server with 100 interlinked pages
    defer srv.Close()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        wv, _ := weaver.NewWeaver(ctx, cfg, origin, logger, io.Discard)
        wv.Weave(ctx)
    }
}
```

**What the numbers would tell us:**

From the live crawl data (42,011 pages, 20 workers, 51.6s):
- **Actual throughput:** ~814 pages/sec (server-limited, not crawler-limited)
- **Per-page overhead estimate:** If the server responded instantly, how fast could the crawler process? The benchmark suite would answer this by removing network latency from the equation.
- **Scaling characteristics:** Does doubling workers double throughput? At what worker count does mutex contention become the bottleneck? The concurrency benchmarks would establish these inflection points.

### Memory profiling considerations

The current design stores all `PageResult` data in memory until the crawl completes. At 42k pages with `FoundLinks` populated, this is the dominant memory consumer. Profiling would quantify:

- **Per-page memory cost:** `PageResult` struct + URL string + `FoundLinks` slice + content-type string
- **Total working set:** For a 42k page crawl, expected to be in the tens of MB range
- **Growth curve:** Linear with page count. A streaming output approach (writing results as they arrive rather than buffering) would cap memory at O(workers) instead of O(pages), at the cost of more complex I/O coordination.

## Known Limitations

- **JavaScript-rendered content (SPAs):** Processes server-rendered HTML only. JS-rendered content would require a headless browser.
- **No sitemap.xml parsing:** Would be a complementary URL discovery source alongside link extraction.
- **No distributed crawling:** Single-process design. The `Frontier` interface supports future sharding via swappable backend.
