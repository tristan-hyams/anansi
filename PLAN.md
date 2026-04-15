# Implementation Plan - Anansi

Build order follows dependency direction: leaf packages first, orchestrator last, CLI on top.
Each phase produces tested, working code before the next begins.

---

## Phase 0 - Project Init

- [ ] `go mod init github.com/tristan-hyams/anansi`
- [ ] Create `Makefile` (build, test, lint, run, clean, docker, docker-run)
- [ ] Create `Dockerfile` (multi-stage: golang:1.24-alpine → alpine)
- [ ] Verify `make build` and `make docker` work with a stub `cmd/anansi/main.go`

**Exit criteria:** `make build` produces a binary. `make docker` produces an image. `make test` passes (no tests yet, but no errors).

---

## Phase 1 - normalizer

URL canonicalization. Pure functions, zero dependencies, highest test density.

- [ ] `normalizer/normalizer.go`
  - `Normalize(base *url.URL, raw string) (*url.URL, error)`
  - Strip fragments (`#section`)
  - Lowercase scheme and host
  - Resolve relative URLs against base (`net/url.ResolveReference`)
  - Strip default ports (`:80` for HTTP, `:443` for HTTPS)
  - Consistent trailing slash handling
  - `IsSameHost(origin, candidate *url.URL) bool` - strict hostname match
  - `IsFollowableScheme(u *url.URL) bool` - only `http` and `https`
- [ ] `normalizer/normalizer_test.go`
  - Table-driven tests: 15+ cases covering every transform
  - Edge cases: empty href, `#`-only, `javascript:void(0)`, `mailto:`, protocol-relative `//cdn.example.com`, query params, encoded characters

**Exit criteria:** `make test` passes. Full coverage of normalizer. Every URL edge case from our design discussion has a test.

---

## Phase 2 - parser

HTML link extraction. Depends on nothing internal. Uses `golang.org/x/net/html` tokenizer.

- [ ] `parser/parser.go`
  - `ExtractLinks(r io.Reader) ([]string, error)`
  - Tokenizer loop: scan for `<a>` start tags, extract `href` attribute
  - Return raw href strings - no filtering, no normalization (that's the caller's job)
- [ ] `parser/parser_test.go`
  - Well-formed HTML with multiple links
  - Malformed / unclosed tags (tokenizer should handle gracefully)
  - No `<a>` tags (empty result)
  - `<a>` without `href` attribute (skip)
  - Mixed content: `<a>`, `<link>`, `<script>` (only extract `<a>`)
  - Inline HTML entities in href values
- [ ] `testdata/` HTML fixtures for parser tests

**Exit criteria:** `make test` passes. Parser handles malformed HTML without panicking. Returns raw hrefs only.

---

## Phase 3 - frontier

URL queue + visited tracking. Interface-based for swappability.

- [ ] `frontier/frontier.go`
  - `Frontier` interface:
    ```go
    type Frontier interface {
        Enqueue(ctx context.Context, u *url.URL) error
        Dequeue(ctx context.Context) (*url.URL, error)
        MarkVisited(u *url.URL)
        IsVisited(u *url.URL) bool
    }
    ```
  - `InMemory` implementation:
    - Buffered channel as queue (configurable buffer size)
    - `sync.Map` for visited set
    - Key is normalized URL string (scheme + host + path + query, no fragment)
- [ ] `frontier/frontier_test.go`
  - Enqueue/Dequeue ordering
  - Dedup: enqueue same URL twice, only dequeue once
  - IsVisited returns true after MarkVisited
  - Context cancellation: Dequeue unblocks when context is cancelled
  - Concurrent access: multiple goroutines enqueuing simultaneously

**Exit criteria:** `make test -race` passes. No data races. Interface is clean enough that a Redis impl could slot in.

---

## Phase 4 - robots

robots.txt compliance. Fetches and parses rules.

- [ ] `robots/robots.go`
  - `Fetch(ctx context.Context, client *http.Client, baseURL *url.URL) (*Rules, error)`
  - `Rules.IsAllowed(path string) bool`
  - Uses `github.com/temoto/robotstxt` for parsing
  - Graceful handling: 404 means allow all, network error means allow all (with warning log)
- [ ] `robots/robots_test.go`
  - Standard robots.txt with Disallow rules
  - Empty robots.txt (allow all)
  - 404 response (allow all)
  - Wildcard patterns
  - Multiple User-agent blocks (only respect `*`)
  - `httptest.NewServer` for all tests

**Exit criteria:** `make test` passes. robots.txt fetch failures don't crash the crawler - they degrade to allow-all with a warning.

---

## Phase 5 — weaver

Orchestrator. Wires everything together. This is the core.

- [x] `weaver/weaver.go`
  - `WeaverConfig` struct: Workers, Rate, MaxDepth, Timeout, BufferSize, UserAgent
  - `Weaver` struct: owns frontier, rate limiter, robots rules, pre-created Crawlers
  - `NewWeaver(ctx context.Context, cfg *WeaverConfig, origin *url.URL, logger *slog.Logger) (*Weaver, error)`
    - Fetch robots.txt during construction (via `robots.Fetch`)
    - Initialize rate limiter, respecting `CrawlDelay` from robots.txt
    - Initialize frontier with origin URL
    - Pre-create Crawlers with per-worker HTTP clients via `webutil.NewClient`
  - `Weave(ctx context.Context) (*Web, error)`
    - Launch N Crawler goroutines
    - Each Crawler: Dequeue → max depth check → rate limit → HTTP GET → Content-Type → X-Robots-Tag → parse → normalize/filter → enqueue
    - Monitor goroutine: polls `frontier.IsDone()` for natural completion
    - Context cancellation path: drain workers via ctx.Done()
  - `Web` struct: Visited, Skipped, Duration, Pages, OriginURL
    - `String()` — markdown summary with spider banner, stats, sitemap tree
    - `JSON()` — machine-readable JSON output
    - `ErrorLog()` — errors grouped by reason with timestamps
    - `ComputeStats()` — latency P50/P95/P99, status codes, content-type breakdown
- [x] `weaver/weaver_test.go` + `weaver/weaver_integration_test.go`
  - Happy path, cycle detection, external link filtering, non-HTML skip
  - robots.txt respected, X-Robots-Tag nofollow, max depth
  - Context cancellation, natural completion (single page, all-external, small site)
  - Live integration test against crawlme.monzo.com

**Exit criteria:** `make test` passes. Weaver completes on test sites and crawlme.monzo.com, respects all filters, terminates cleanly.

---

## Phase 6 - cmd/anansi

CLI entry point. Thin - all logic lives in packages.

- [ ] `cmd/anansi/main.go`
  - Flag parsing: `-workers`, `-rate`, `-max-depth`, `-timeout`
  - URL argument validation (positional arg, must parse as valid URL)
  - `slog` setup: JSON handler to stderr for structured logs, summary to stdout
  - `signal.NotifyContext` for SIGINT/SIGTERM
  - Wire `crawler.New()` → `crawler.Run()`
  - Print summary report on completion:
    ```
    Crawl Results: crawlme.monzo.com
    ================================
    Pages crawled: 47
    Pages skipped: 3 (non-HTML)
    Duration: 12.4s

    /                          → 8 links
    /about                     → 5 links
    /about/team                → 12 links
    ...
    ```
  - Exit codes: 0 success, 1 error, 130 interrupt (SIGINT convention)

**Exit criteria:** `make run` crawls the target site and prints output. `make docker-run` does the same. Ctrl+C produces a clean shutdown with summary of work done.

---

## Phase 7 - Polish

- [ ] Run `golangci-lint run`, fix all findings
- [ ] Review test coverage: `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out`
- [ ] Verify `make docker-run` works end-to-end
- [ ] Final pass on README.md
- [ ] Update .context/journal with completion entry
- [ ] Git: clean history, meaningful commits per phase

**Exit criteria:** Submission-ready. Clean lint, good coverage, working Docker, clear README, documented decisions.

---

## Dependency Graph (Build Order)

```
Phase 0: init
    │
    ├── Phase 1: normalizer (no deps)
    ├── Phase 2: parser (no deps)
    ├── Phase 3: frontier (no deps)
    ├── Phase 4: robots (no deps)
    │
    └── Phase 5: crawler (depends on 1-4)
            │
            └── Phase 6: cmd/anansi (depends on 5)
                    │
                    └── Phase 7: polish
```

Phases 1-4 are independent - can be built in any order or in parallel. Phase 5 integrates them. Phase 6 wraps Phase 5. Phase 7 is final review.

---

## Key Dependencies (go.mod)

```
golang.org/x/net       # HTML tokenizer (parser)
golang.org/x/time      # rate.Limiter (crawler)
github.com/temoto/robotstxt  # robots.txt parsing (robots)
```

No other external dependencies. Standard library for everything else.
