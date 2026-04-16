# Project Structure - Anansi

Repository layout and package responsibilities.

---

## Module

Single Go module: `github.com/tristan-hyams/anansi`

Top-level packages are public - importable as libraries by external consumers.

---

## Dependency Direction

```
cmd/anansi (main) → reporting → weaver → (frontier, parser, normalizer, robots, webutil)
cmd/anansi (main) → weaver (for WeaverConfig, NewWeaver, Weave)
```

`weaver` is the orchestrator. All other packages are leaf dependencies with no cross-imports between them (except `robots` → `webutil` for HTTP client creation).

---

## Directory Tree

```
anansi/
├── cmd/
│   └── anansi/
│       ├── main.go              # CLI entry: wiring, fatal(), summary output
│       ├── config.go            # AnansiConfig struct, JSON serialization
│       ├── config_test.go       # Config unit tests (package main_test)
│       ├── startup.go           # ParseFlags, SetupSignalContext, SetupLogger, StartPprofServer
│       ├── startup_test.go      # Logger, signal context, pprof server tests
│       └── consts.go            # Default constants, exit codes, error format
├── weaver/
│   ├── weaver.go                # Weaver struct, NewWeaver(), Weave(), monitor, redirectPolicy
│   ├── crawler.go               # Crawler struct, page fetch pipeline, readCloser
│   ├── retry.go                 # Generic withRetry[T] with exponential backoff
│   ├── retry_test.go            # Retry unit tests (7 cases)
│   ├── config.go                # WeaverConfig, NewWeaverConfig(), Validate(), CrawlRate()
│   ├── config_test.go           # Validation + constructor defaults tests
│   ├── result.go                # Web and PageResult - crawl result data structs
│   ├── consts.go                # Defaults, log keys, retry, body limits, redirect constants
│   ├── weaver_test.go           # httptest integration tests
│   └── weaver_integration_test.go # Live test against crawlme.monzo.com
├── reporting/
│   ├── markdown.go              # RenderMarkdown(), RenderErrorLog(), sitemap tree
│   ├── json.go                  # RenderJSON() - machine-readable JSON output
│   ├── stats.go                 # ComputeStats(), Stats, LatencyStats, percentiles
│   ├── writer.go                # CreateOutputDir(), WriteOutputFiles() - UUIDv7 dirs + file I/O
│   ├── consts.go                # Banner, summary formatting, output filenames
│   ├── stats_test.go            # ComputeStats, computeLatency tests
│   ├── markdown_test.go         # RenderMarkdown, RenderErrorLog tests
│   ├── json_test.go             # RenderJSON tests
│   └── writer_test.go           # WriteOutputFiles, CreateOutputDir tests
├── frontier/
│   ├── frontier.go              # Frontier interface (7 methods), InMemory impl
│   ├── frontierurl.go           # FrontierURL struct (URL + Depth)
│   ├── frontier_test.go         # Queue, dedup, select behavior, pending, concurrency
│   └── consts.go                # defaultBufferSize
├── normalizer/
│   ├── normalizer.go            # Normalize, IsSameHost, IsFollowableScheme
│   └── normalizer_test.go       # Table-driven: URLs, ports, schemes, hosts
├── parser/
│   ├── parser.go                # ExtractLinks via x/net/html tokenizer
│   ├── parser_test.go           # Fixtures + inline HTML tests
│   ├── parser_integration_test.go # Live test against crawlme.monzo.com
│   └── testdata/                # HTML fixtures
├── robots/
│   ├── robots.go                # Fetch() + Rules wrapper for robots.txt
│   ├── xrobotstag.go            # ParseXRobotsTag() for per-page X-Robots-Tag
│   ├── robots_test.go           # httptest-based unit tests
│   ├── robots_integration_test.go # Live tests against crawlme.monzo.com
│   ├── xrobotstag_test.go       # Directive parsing tests
│   └── consts.go                # userAgent, fetchTimeout, xRobotsTagHeader
├── webutil/
│   ├── transport.go             # Singleton Transport(), NewClient()
│   ├── transport_test.go        # Singleton and client tests
│   └── consts.go                # Dial, TLS, pool, timeout settings
├── benchmark/
│   ├── normalizer_bench_test.go # Normalize, IsSameHost benchmarks
│   ├── parser_bench_test.go     # ExtractLinks small/large benchmarks
│   ├── frontier_bench_test.go   # Enqueue, Dequeue throughput benchmarks
│   └── stats_bench_test.go      # ComputeStats scaling benchmarks
├── testutil/
│   └── integration.go           # SkipIfNoIntegration helper, .env.test loader
├── output/                      # Crawl output (gitignored), one UUIDv7 dir per run
├── .context/                    # AI agent context (rules, architecture, journal)
│   ├── RULES.md
│   ├── STRUCTURE.md             # ← you are here
│   ├── ARCH.md
│   └── journal/
├── .claude/
│   ├── CLAUDE.md                # Claude Code shim → points to .context/
│   └── memory/                  # Claude persistent memory (feedback, project notes)
├── .github/
│   └── copilot-instructions.md  # GitHub Copilot shim → points to .context/
├── AGENTS.md                    # Codex/agent shim → points to .context/
├── BENCHMARK.md                 # Benchmark suite, pprof usage, profiling targets
├── DESIGN.md                    # Design decisions, trade-offs, rationale
├── TESTING.md                   # Test strategy, integration tests, race detector
├── PLAN.md                      # Phased implementation plan
├── Dockerfile                   # Multi-stage: golang:1.26-alpine → alpine:3.23
├── Makefile                     # build, test, lint, run, clean, tidy, update, docker
├── anansi.code-workspace        # VS Code workspace: F5 debug, revive lint-on-save
├── revive.toml                  # Revive linter config (enableAllRules + overrides)
├── .env.test                    # Integration test config (GO_RUN_INTEGRATIONS)
├── README.md
├── .gitignore
├── go.mod
└── go.sum
```

---

## Package Responsibilities

| Package | Responsibility | Key Types |
|---|---|---|
| `cmd/anansi` | CLI entry point. Parses flags, wires weaver, delegates output to reporting. Only place that calls `os.Exit`. | `main()`, `AnansiConfig`, `ParseFlags()`, `SetupLogger()` |
| `weaver` | Orchestrates the crawl. Owns frontier, rate limiter, robots rules, redirect policy. Pre-creates Crawlers with body size limits. | `Weaver`, `Crawler`, `NewWeaverConfig()`, `WeaverConfig`, `Web`, `PageResult` |
| `reporting` | Crawl report rendering and output. Creates per-run output directories (UUIDv7). Converts `Web` results to markdown, JSON, error logs. | `CreateOutputDir()`, `RenderMarkdown()`, `RenderJSON()`, `RenderErrorLog()`, `ComputeStats()`, `WriteOutputFiles()` |
| `frontier` | URL queue + visited tracking + pending counter. Interface-based for swappability. | `Frontier` (7 methods), `InMemory`, `FrontierURL` |
| `parser` | Extracts `<a href>` links from HTML using tokenizer. No URL filtering - returns raw hrefs. | `ExtractLinks(ctx, r io.Reader) ([]string, error)` |
| `normalizer` | Canonicalizes URLs: strips fragments, lowercases host, resolves relative paths. Pure functions. | `Normalize(base, raw)`, `IsSameHost(origin, candidate)`, `IsFollowableScheme(u)` |
| `robots` | robots.txt + X-Robots-Tag compliance. Creates own HTTP client via webutil. | `Fetch(ctx, baseURL, logger)`, `Rules`, `IsAllowed()`, `Directives`, `ParseXRobotsTag()` |
| `webutil` | Shared HTTP transport singleton. Per-worker client creation. | `Transport()`, `NewClient(timeout)` |
| `testutil` | Shared test helpers. Integration test gating via `.env.test`. | `SkipIfNoIntegration(t)` |
