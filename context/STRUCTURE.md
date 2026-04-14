# Project Structure - Anansi

Repository layout and package responsibilities.

---

## Module

Single Go module: `github.com/tristan-hyams/anansi`

Top-level packages are public — importable as libraries by external consumers.

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
│       └── main.go              # CLI entry: flag parsing, signal handling, slog setup
├── crawler/
│   ├── crawler.go               # Orchestrator: worker pool, WaitGroup, rate limiter
│   └── crawler_test.go          # Integration tests with httptest servers
├── frontier/
│   ├── frontier.go              # Frontier interface + in-memory implementation
│   └── frontier_test.go         #   (seen-set, buffered channel queue)
├── parser/
│   ├── parser.go                # HTML tokenizer → link extraction
│   └── parser_test.go           #   Uses golang.org/x/net/html tokenizer
├── normalizer/
│   ├── normalizer.go            # URL canonicalization (fragments, scheme, trailing slash)
│   └── normalizer_test.go       #   Pure functions, high test density
├── robots/
│   ├── robots.go                # robots.txt fetch + Disallow rule checking
│   └── robots_test.go
├── testdata/                    # Canned HTML fixtures for httptest servers
│   ├── simple/
│   ├── cycle/
│   ├── external_links/
│   ├── fragments/
│   ├── non_html/
│   ├── malformed/
│   ├── relative_urls/
│   └── schemes/
├── context/                     # AI agent context (rules, architecture, journal)
│   ├── RULES.md
│   ├── STRUCTURE.md             # ← you are here
│   ├── ARCH.md
│   └── journal/
├── .claude/
│   └── CLAUDE.md                # Claude Code shim → points to context/
├── .github/
│   └── copilot-instructions.md  # GitHub Copilot shim → points to context/
├── AGENTS.md                    # Codex/agent shim → points to context/
├── Dockerfile                   # Multi-stage: golang:1.24-alpine → alpine runtime
├── Makefile                     # build, test, lint, run, docker, docker-run
├── README.md
├── .gitignore
├── go.mod
└── go.sum
```

---

## Package Responsibilities

| Package | Responsibility | Key Types |
|---|---|---|
| `cmd/anansi` | CLI entry point. Parses flags, wires dependencies, handles SIGINT/SIGTERM. | `main()` |
| `crawler` | Orchestrates the crawl. Owns worker pool, rate limiter, WaitGroup. Consumes from frontier, delegates to parser. | `Crawler`, `Config`, `Result` |
| `frontier` | URL queue + visited tracking. Interface-based for swappability. | `Frontier` (interface), `InMemory` (impl) |
| `parser` | Extracts `<a href>` links from HTML using tokenizer. No URL filtering — returns raw hrefs. | `ExtractLinks(io.Reader, *url.URL) []string` |
| `normalizer` | Canonicalizes URLs: strips fragments, lowercases host, resolves relative paths. Pure functions. | `Normalize(base *url.URL, raw string) (*url.URL, error)` |
| `robots` | Fetches and parses `robots.txt`. Checks URLs against Disallow rules. | `Rules`, `IsAllowed(path string) bool` |
