# Project Structure - Anansi

Repository layout and package responsibilities.

---

## Module

Single Go module: `github.com/tristan-hyams/anansi`

Top-level packages are public - importable as libraries by external consumers.

---

## Dependency Direction

```
cmd/anansi (main) → crawler → (frontier, parser, normalizer, robots)
```

`crawler` is the orchestrator. All other packages are leaf dependencies with no cross-imports between them.

---

## Directory Tree

```
anansi/
├── cmd/
│   └── anansi/
│       ├── main.go              # CLI entry: signal handling, wiring
│       ├── config.go            # AnansiConfig struct, ParseFlags, JSON serialization
│       ├── config_test.go       # Config unit tests (package main_test)
│       ├── consts.go            # Default constants (workers, rate, timeout)
│       └── logger.go            # slog JSON handler setup
├── crawler/                     # (Phase 5 — not yet implemented)
├── frontier/
│   ├── frontier.go              # Frontier interface, InMemory impl, FrontierURL struct
│   ├── frontier_test.go         # Queue, dedup, select behavior, concurrency tests
│   └── consts.go                # defaultBufferSize
├── normalizer/
│   ├── normalizer.go            # Normalize, IsSameHost, IsFollowableScheme
│   └── normalizer_test.go       # Table-driven: URLs, ports, schemes, hosts
├── parser/
│   ├── parser.go                # ExtractLinks via x/net/html tokenizer
│   ├── parser_test.go           # Fixtures + inline HTML tests
│   ├── parser_integration_test.go # Live test against crawlme.monzo.com
│   └── testdata/                # HTML fixtures
├── robots/                      # (Phase 4 — not yet implemented)
├── testutil/
│   └── integration.go           # SkipIfNoIntegration helper, .env.test loader
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
├── PLAN.md                      # Phased implementation plan
├── Dockerfile                   # Multi-stage: golang:1.26-alpine → alpine:3.23
├── Makefile                     # build, test, lint, run, clean, tidy, update, docker
├── anansi.code-workspace        # VS Code workspace: F5 debug, revive lint-on-save
├── revive.toml                  # Revive linter config (enableAllRules + overrides)
├── README.md
├── .gitignore
├── go.mod
└── go.sum
```

---

## Package Responsibilities

| Package | Responsibility | Key Types |
|---|---|---|
| `cmd/anansi` | CLI entry point. Parses flags, wires dependencies, handles SIGINT/SIGTERM. | `main()`, `AnansiConfig`, `ParseFlags()` |
| `crawler` | Orchestrates the crawl. Owns worker pool, rate limiter, WaitGroup. Consumes from frontier, delegates to parser. | `Crawler`, `Config`, `Result` |
| `frontier` | URL queue + visited tracking. Interface-based for swappability. Dedup built into Enqueue. | `Frontier` (interface), `InMemory` (impl), `FrontierURL`, `Status` |
| `parser` | Extracts `<a href>` links from HTML using tokenizer. No URL filtering — returns raw hrefs. | `ExtractLinks(ctx context.Context, r io.Reader) ([]string, error)` |
| `normalizer` | Canonicalizes URLs: strips fragments, lowercases host, resolves relative paths. Pure functions. | `Normalize(base *url.URL, raw string) (*url.URL, error)` |
| `robots` | Fetches and parses `robots.txt`. Checks URLs against Disallow rules. | `Rules`, `IsAllowed(path string) bool` |
