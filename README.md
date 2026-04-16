# Anansi

A single-domain web crawler written in Go. Given a starting URL, Anansi visits every reachable page on the same subdomain, printing each URL visited and the links found on that page.

Named after [Anansi](https://en.wikipedia.org/wiki/Anansi), the West African spider of folklore - a weaver of webs and stories.

## Getting Started

Open the project using `anansi.code-workspace` for the best experience - it configures the debugger (F5), revive linter, and lint-on-save out of the box:

```
File → Open Workspace from File → anansi.code-workspace
```

## Quick Start

```bash
# Native (requires Go 1.26+)
make run

# Docker (no Go required)
make docker-run
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
```

### Running with Make

```bash
# Default URL (crawlme.monzo.com) with default flags (1 worker, 1 req/s, depth 1)
make run

# Custom URL
make run URL=https://example.com/

# Custom flags
make run ARGS="-workers 5 -rate 10 -max-depth 3"

# Custom flags + custom URL
make run ARGS="-workers 5 -rate 10" URL=https://example.com/
```

### Running with Docker

```bash
# Default — crawls crawlme.monzo.com with default flags
docker run --rm anansi

# Custom URL
docker run --rm anansi https://example.com/

# Custom flags + URL
docker run --rm anansi -workers 5 -rate 10 https://example.com/

# Via Make
make docker-run
make docker-run ARGS="-workers 5 -max-depth 3"
make docker-run ARGS="-workers 5" URL=https://example.com/
```

### Running the Binary Directly

```bash
make build
bin\anansi.exe https://crawlme.monzo.com/
bin\anansi.exe -workers 5 -rate 10 -max-depth 3 https://example.com/
```

### Output Files

Every crawl automatically generates two files in the current directory:

| File | Contents |
|------|----------|
| `crawl-results.md` | Spider banner, latency stats (P50/P95/P99), status codes, content-type breakdown, page list, directory tree sitemap |
| `crawl-results.json` | Machine-readable JSON: same data as markdown, pipeable to `jq` |
| `crawl-errors.md` | Errors grouped by reason, each URL timestamped (only created if errors occurred) |

The terminal shows structured JSON logs (stderr) and a short one-liner on completion:

```
crawl complete: 57 pages crawled, 84 skipped, 11.2s
```

To capture logs separately: `bin\anansi.exe https://crawlme.monzo.com/ 2>crawl.log`

## Development

```bash
make build       # Build binary to bin\anansi.exe
make test        # Run tests with coverage
make lint        # Run revive linter
make tidy        # Run go mod tidy
make update      # Update all deps + tidy
make setup       # Install gopls, dlv, revive
make clean       # Remove build artifacts
```

### Integration Tests

Live network tests are gated by `.env.test`. To run them:

```bash
# Edit .env.test → set GO_RUN_INTEGRATIONS=true
make test
```

## Architecture

Anansi uses a **worker pool** with a buffered channel as a work queue. A global token-bucket rate limiter controls request throughput. URLs are normalized, deduplicated, and filtered by domain before enqueueing.

| Package | Responsibility |
|---|---|
| `cmd/anansi` | CLI entry point, flag parsing, wires weaver and fileutil |
| `weaver` | Orchestrator: Weaver spawns Crawlers, owns frontier/rate limiter |
| `fileutil` | Rendering and file output: markdown, JSON, error logs, stats |
| `frontier` | URL queue + visited tracking (interface-based) |
| `parser` | HTML link extraction via tokenizer |
| `normalizer` | URL canonicalization |
| `robots` | robots.txt + X-Robots-Tag compliance |
| `webutil` | Shared HTTP transport singleton, per-worker clients |

For detailed design decisions, trade-offs, and the URL processing pipeline, see [.context/ARCH.md](.context/ARCH.md).

### Design Trade-offs

- **Worker pool over goroutine-per-URL** - predictable resource usage, configurable concurrency.
- **Interface-based frontier** - in-memory for this scope. A production system would swap in Redis/RabbitMQ for restart durability and distributed crawling.
- **Strict subdomain matching** - `crawlme.monzo.com` does not match `www.crawlme.monzo.com`. The spec says "limited to one subdomain"; strict matching is the safe interpretation.
- **robots.txt respected** - fetched once at crawl start. `User-agent: *` Disallow rules honoured. 403 treated as not found (CDN/static host behavior).
- **X-Robots-Tag respected** - per-page header check. `nofollow` and `none` skip link extraction.
- **Content-Type checked** - only `text/html` responses are parsed. Non-HTML resources are logged but not followed.

### Known Limitations

- **JavaScript-rendered content (SPAs):** This crawler processes server-rendered HTML. JS-rendered content would require a headless browser (e.g., `chromedp`). This is a deliberate scope boundary.
- **No sitemap.xml parsing:** Would be a complementary URL discovery source but is not required by the spec.
- **No distributed crawling:** Single-process design. The `Frontier` interface supports future sharding via swappable backend implementations.

## Testing

Tests use `httptest.NewServer` with canned HTML fixtures in `testdata/`. No external network calls during tests unless `GO_RUN_INTEGRATIONS=true` is set in `.env.test`.

```bash
make test
```

## License

MIT
