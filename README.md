# Anansi

A single-domain web crawler written in Go. Given a starting URL, Anansi visits every reachable page on the same subdomain, printing each URL visited and the links found on that page.

Named after [Anansi](https://en.wikipedia.org/wiki/Anansi), the West African spider of folklore - a weaver of webs and stories.

## Dev/IDE Get Started

Open the project using `anansi.code-workspace` for the best experience - it configures the debugger (F5), revive linter, and lint-on-save out of the box:

```
File → Open Workspace from File → anansi.code-workspace
```

## Quick Start

```bash
# Native (requires Go 1.26+)
make run

# Native run with 10 crawlers, self-rate limited to 5000 req/s, unlimited depth.
make run ARGS="-workers 10 -rate 5000 -max-depth 0 -log-links=false"

# Docker (no Go required)
make docker-run

# Docker run with 50 crawlers, self-rate limited to 10000 req/s, unlimited depth.
make docker-run ARGS="-workers 50 -rate 10000 -max-depth 0 -log-links=false"
```

## Usage

```bash
anansi [flags] <url>

Flags:
  -workers int      Number of concurrent workers (default 1)
  -rate float       Max requests per second (default 1)
  -max-depth int    Maximum crawl depth, 0 for unlimited (default 1)
  -timeout duration HTTP request timeout (default 30s)
  -log-level string Log level: debug, info, warn, error (default "info")
  -log-links        Print each visited URL and its links to stdout (default true)
```

See [cmd/anansi/README.md](cmd/anansi/README.md) for full running examples (Make, Docker, binary).

### Output Files

Every crawl generates output files in the current directory:

| File | Contents |
|------|----------|
| `crawl-results.md` | Spider banner, latency stats (P50/P95/P99), status codes, content-type breakdown, page list, directory tree sitemap |
| `crawl-results.json` | Machine-readable JSON: same data as markdown, pipeable to `jq` |
| `crawl-errors.md` | Errors grouped by reason, each URL timestamped (only created if errors occurred) |

## Architecture

Worker pool with a buffered channel as the work queue. Global token-bucket rate limiter. URLs normalized, deduplicated, and domain-filtered before enqueueing.

| Package | Responsibility | Details |
|---|---|---|
| [`cmd/anansi`](cmd/anansi/README.md) | CLI entry point, flag parsing | Wires weaver and fileutil |
| [`weaver`](weaver/README.md) | Orchestrator | Spawns Crawlers, owns frontier and rate limiter |
| [`fileutil`](fileutil/README.md) | Rendering and file output | Markdown, JSON, error logs, stats |
| [`frontier`](frontier/README.md) | URL queue + visited tracking | Interface-based, swappable backend |
| [`parser`](parser/README.md) | HTML link extraction | `x/net/html` tokenizer |
| [`normalizer`](normalizer/README.md) | URL canonicalization | Pure functions, zero dependencies |
| [`robots`](robots/README.md) | robots.txt + X-Robots-Tag | Fetched once at crawl start |
| [`webutil`](webutil/README.md) | HTTP transport singleton | Per-worker clients, shared connection pool |

For design decisions, trade-offs, and rationale, see [DESIGN.md](DESIGN.md).

## Development

```bash
make build       # Build binary to bin\anansi.exe
make test        # Run tests with coverage
make lint        # Run revive linter
make tidy        # Run go mod tidy
make clean       # Remove build artifacts
```

Open with `anansi.code-workspace` for debugger (F5), revive linter, and lint-on-save.

## Testing

See [TESTING.md](TESTING.md) for test strategy, integration tests, race detector, and test categories.

```bash
make test
```

## License

MIT
