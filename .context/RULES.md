# Coding Rules - Anansi

Shared rules for all AI agents and tools. Referenced by `.claude/CLAUDE.md`, `AGENTS.md`, and `.github/copilot-instructions.md`.

---

## Identity

Anansi is a single-domain web crawler written in Go.

Given a starting URL, it visits every reachable page on the same subdomain, printing each URL visited and links found.

Built as a take-home challenge demonstrating concurrency design, software structure, testing strategy, and production awareness.

Clean interfaces.

Production-quality code.

---

## Design Principles

Apply judgment. These describe *why* - use them to reason about novel situations.

- **Interfaces for swappable boundaries.** Define interfaces at infrastructure seams (frontier, HTTP client). In-memory implementations for now; comment where a production system would use Redis/Postgres/RabbitMQ.
- **Concurrency is explicit.** Worker pool with bounded goroutines. No unbounded goroutine spawning. All shared state protected by `sync.Mutex` or `sync.Map` - document which and why.
- **Errors are values, not panics.** Return errors up the stack. `panic` only for programmer bugs (nil dereference in init). Wrap errors with `fmt.Errorf("context: %w", err)` for stack context.
- **No global state.** Pass dependencies via struct fields or function parameters. No `init()` side effects beyond flag registration.
- **Pure functions are testable functions.** URL normalization, domain matching, link extraction - keep these as pure functions with table-driven tests.
- **Log structure, not just strings.** Use `log/slog` with key-value pairs. No `fmt.Println` for operational output.
- **Respect the network.** Rate limit all outbound HTTP. Honour `robots.txt`. Check `Content-Type` before parsing. Handle 429 with backoff.

---

## Go Conventions

| Convention | Rule |
|---|---|
| Go version | 1.26+ (latest stable) |
| Module path | `github.com/tristan-hyams/anansi` |
| Package layout | Top-level packages (`weaver/`, `reporting/`, `frontier/`, `parser/`, `normalizer/`, `robots/`, `webutil/`, `testutil/`). No `internal/`. |
| Testing | `go test -cover ./...` - race detector enabled in CI/Docker where CGO is available |
| Linting | `revive -config revive.toml ./...` |
| Build | `CGO_ENABLED=0 go build -o bin/anansi ./cmd/anansi` |
| Error wrapping | `fmt.Errorf("operation: %w", err)` - always wrap with context |
| Exit control | Only `main()` may call `os.Exit` or `log.Fatal`. All other functions return errors to the caller. |
| Error returns | Return `(nil, err)` with pointer receivers, not zero-value structs. Prevents callers from using an uninitialised config. |
| Context propagation | Pass `context.Context` as first parameter. Respect cancellation. |
| Naming | Exported types: `PascalCase`. Unexported: `camelCase`. Acronyms: `URL`, `HTTP`, `HTML` (all caps). |

---

## Rejected Patterns

| Pattern | Why Rejected |
|---|---|
| `internal/` packages | Packages are public by design - reusable as library imports. README notes where `go.work` multi-module would apply at scale. |
| `go-colly` / `scrapy` | Spec prohibits crawl frameworks. Own implementation required. |
| Headless browser (chromedp) | Out of scope. Crawler handles server-rendered HTML only. SPA limitation documented in README. |
| Unbounded goroutines | Worker pool with configurable concurrency. Predictable resource usage. |
| `fmt.Println` for output | Structured logging (`slog`) for crawl events. Summary report to stdout on completion. |
| Docker Compose / multi-service | Single static binary. Dockerfile is multi-stage for reviewer convenience, not architecture. |

---

## Testing

- **Unit tests per package.** Table-driven tests for `normalizer`, `parser`, `robots`.
- **Integration tests with `httptest.NewServer`.** Canned HTML in `testdata/` directories.
- **Race detector in CI.** `-race` requires CGO; enabled in CI/Docker, not in local Windows builds.
- **Test categories in `testdata/`:**
  - `simple/` - happy path, 3 pages linking to each other
  - `cycle/` - A → B → A cycle detection
  - `external_links/` - mix of internal and external links
  - `fragments/` - `#section` and `/page#section` handling
  - `non_html/` - endpoints returning `application/json`, images
  - `malformed/` - broken HTML, unclosed tags
  - `relative_urls/` - `../sibling`, `./child`, `//protocol-relative`
  - `schemes/` - `mailto:`, `javascript:void(0)`, `tel:`

---

## Commands

```bash
# Build
make build

# Run (default: https://crawlme.monzo.com/)
make run
make run URL=https://example.com/

# Test (coverage)
make test

# Lint (revive)
make lint

# Dependencies
make tidy           # go mod tidy
make update         # go get -u ./... + tidy

# Docker (no Go install required)
make docker-run
make docker-run URL=https://example.com/

# Clean
make clean
```

---

## Agent Workflow

- **Journal:** Read the latest entry in `.context/journal/` at session start for continuity.
- **Naming:** `YYYY-MM-DD_NN.md` (date + sequence number per day).
- **Content:** Context, decisions made, work done, next steps.
- **Package source:** Read the relevant package source before modifying it.
- **Tests alongside code:** Write tests in the same commit as the feature. Never skip `-race`.
- **Context files are source of truth.** Do not duplicate rules or architecture in shim files. Update `.context/` files directly.
