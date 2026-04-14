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

# Custom target
make run URL=https://example.com/
make docker-run URL=https://example.com/
```

## Usage

```bash
anansi [flags] <url>

Flags:
  -workers int      Number of concurrent workers (default 5)
  -rate float       Max requests per second (default 5)
  -max-depth int    Maximum crawl depth, 0 for unlimited (default 0)
  -timeout duration HTTP request timeout (default 30s)
  -log-level string Log level: debug, info, warn, error (default "info")
```

## Development

```bash
make build       # Build binary to bin\anansi.exe
make test        # Run tests with coverage
make lint        # Run revive linter
make tidy        # Run go mod tidy
make update      # Update all deps + tidy
make clean       # Remove build artifacts
make run         # Build and run against default URL
make docker-run  # Build Docker image and run
```

## Architecture

Anansi uses a **worker pool** with a buffered channel as a work queue. A global token-bucket rate limiter controls request throughput. URLs are normalized, deduplicated, and filtered by domain before enqueueing.

| Package | Responsibility |
|---|---|
| `cmd/anansi` | CLI entry point, flag parsing, signal handling |
| `crawler` | Orchestrator: worker pool, rate limiter, WaitGroup |
| `frontier` | URL queue + visited tracking (interface-based) |
| `parser` | HTML link extraction via tokenizer |
| `normalizer` | URL canonicalization |
| `robots` | robots.txt compliance |

For detailed design decisions, trade-offs, and the URL processing pipeline, see [.context/ARCH.md](.context/ARCH.md).

### Design Trade-offs

- **Worker pool over goroutine-per-URL** - predictable resource usage, configurable concurrency.
- **Interface-based frontier** - in-memory for this scope. A production system would swap in Redis/RabbitMQ for restart durability and distributed crawling.
- **Strict subdomain matching** - `crawlme.monzo.com` does not match `www.crawlme.monzo.com`. The spec says "limited to one subdomain"; strict matching is the safe interpretation.
- **robots.txt respected** - fetched once at crawl start. `User-agent: *` Disallow rules honoured.
- **Content-Type checked** - only `text/html` responses are parsed. Non-HTML resources are logged but not followed.

### Known Limitations

- **JavaScript-rendered content (SPAs):** This crawler processes server-rendered HTML. JS-rendered content would require a headless browser (e.g., `chromedp`). This is a deliberate scope boundary.
- **No sitemap.xml parsing:** Would be a complementary URL discovery source but is not required by the spec.
- **No distributed crawling:** Single-process design. The `Frontier` interface supports future sharding via swappable backend implementations.

## Testing

Tests use `httptest.NewServer` with canned HTML fixtures in `testdata/`. No external network calls during tests.

```bash
make test
```

## License

MIT
