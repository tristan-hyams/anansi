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

# Custom URL with tuned settings
make run URL=https://crawler-test.com/ ARGS="-workers 2 -rate 20 -max-depth 3 -max-retries 1 -log-links=false"

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
  -workers int         Number of concurrent workers (default 1)
  -rate float          Max requests per second (default 1)
  -max-depth int       Maximum crawl depth, 0 for unlimited (default 1)
  -max-duration dur    Maximum crawl duration, 0 for unlimited (e.g. 60s, 5m)
  -max-retries int     Max retry attempts for transient errors (default 2, -1 = disabled)
  -timeout duration    HTTP request timeout (default 30s)
  -log-level string    Log level: debug, info, warn, error (default "info")
  -log-links           Print each visited URL and its links to stdout (default true)
```

See [cmd/anansi/README.md](cmd/anansi/README.md) for full running examples (Make, Docker, binary).

### Output Files

Each crawl writes to a unique directory under `./output/<uuidv7>/`:

| File | Contents |
|------|----------|
| `crawl-results.md` | Spider banner, latency stats (P50/P95/P99), status codes, content-type breakdown, page list, directory tree sitemap |
| `crawl-results.json` | Machine-readable JSON: same data as markdown, pipeable to `jq` |
| `crawl-errors.md` | Errors grouped by reason, each URL timestamped (only created if errors occurred) |

The output directory path is printed to stderr before the crawl starts.

## Architecture

Worker pool with a buffered channel as the work queue. Global token-bucket rate limiter. URLs normalized, deduplicated, and domain-filtered before enqueueing.

| Package | Responsibility | Details |
|---|---|---|
| [`cmd/anansi`](cmd/anansi/README.md) | CLI entry point, flag parsing | Wires weaver and reporting |
| [`weaver`](weaver/README.md) | Orchestrator | Spawns Crawlers, owns frontier and rate limiter |
| [`reporting`](reporting/README.md) | Rendering and file output | Markdown, JSON, error logs, stats |
| [`frontier`](frontier/README.md) | URL queue + visited tracking | Interface-based, swappable backend |
| [`parser`](parser/README.md) | HTML link extraction | `x/net/html` tokenizer |
| [`normalizer`](normalizer/README.md) | URL canonicalization | Pure functions, zero dependencies |
| [`robots`](robots/README.md) | robots.txt + X-Robots-Tag | Fetched once at crawl start |
| [`webutil`](webutil/README.md) | HTTP transport singleton | Per-worker clients, shared connection pool |

For design decisions, trade-offs, and rationale, see [DESIGN.md](DESIGN.md).

### Sample Output

When running with `-log-links` (default), stdout shows each visited URL and its discovered links:

```
https://crawlme.monzo.com/
  https://crawlme.monzo.com/blog.html
  https://crawlme.monzo.com/products.html
  https://crawlme.monzo.com/about.html
  https://crawlme.monzo.com/contact.html

https://crawlme.monzo.com/blog.html
  https://crawlme.monzo.com/
  https://crawlme.monzo.com/blog/1
  https://crawlme.monzo.com/blog/2
```

Stderr shows structured JSON logs and a completion summary:

```
crawl complete: 42010 pages crawled, 1 skipped, 1m38.2s
```

## Development

```bash
make build       # Build binary to bin\anansi.exe
make test        # Run tests with coverage
make lint        # Run revive linter
make bench       # Run benchmarks
make tidy        # Run go mod tidy
make clean       # Remove build artifacts
```

### Profiling

pprof is available via the `ANANSI_DEBUG` environment variable:

```bash
ANANSI_DEBUG=1 make run ARGS="-workers 20 -rate 5000 -max-depth 0 -log-links=false"

# In another terminal:
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
go tool pprof http://localhost:6060/debug/pprof/heap
```

See [DESIGN.md](DESIGN.md) for profiling targets and benchmark methodology.

## Testing

See [TESTING.md](TESTING.md) for test strategy, integration tests, race detector, and test categories.

```bash
make test
make bench
```

## License

MIT
