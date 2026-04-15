# cmd/anansi

CLI entry point for the Anansi web crawler.

Thin wiring layer — all logic lives in packages. This is the only place that calls `os.Exit`.

## Running

```bash
# Build and run with defaults (1 worker, 1 req/s, depth 1)
make run

# Custom flags via ARGS
make run ARGS="-workers 5 -rate 10 -max-depth 3"

# Custom URL
make run URL=https://example.com/

# Both
make run ARGS="-workers 5 -rate 10" URL=https://example.com/

# Binary directly
make build
bin\anansi.exe https://crawlme.monzo.com/
bin\anansi.exe -workers 5 -rate 10 -max-depth 3 https://example.com/

# Docker (no Go required)
docker run --rm anansi
docker run --rm anansi https://example.com/
docker run --rm anansi -workers 5 -rate 10 https://example.com/

# Docker via Make
make docker-run
make docker-run ARGS="-workers 5 -max-depth 3"
```

## Flags

```
anansi [flags] <url>

  -workers int      Number of concurrent workers (default 1)
  -rate float       Max requests per second (default 1)
  -max-depth int    Maximum crawl depth, 0 for unlimited (default 1)
  -timeout duration HTTP request timeout (default 30s)
  -log-level string Log level: debug, info, warn, error (default "info")
```

## Output

Every crawl generates two files in the current directory:

| File | Contents |
|------|----------|
| `crawl-results.md` | Spider banner, latency stats (P50/P95/P99), status codes, content-type breakdown, page list, sitemap tree |
| `crawl-results.json` | Machine-readable JSON: same data as markdown, pipeable to `jq` |
| `crawl-errors.md` | Errors grouped by reason, each URL timestamped (only if errors occurred) |

The terminal shows:
- **stderr** — structured JSON logs (crawl progress, warnings, debug)
- **stderr** — short completion summary: `crawl complete: 57 pages crawled, 84 skipped, 11.2s`

To capture logs: `bin\anansi.exe https://crawlme.monzo.com/ 2>crawl.log`

## Responsibilities

- Flag parsing (`ParseFlags`)
- Config loading from JSON (`LoadConfigFromFile`)
- Structured logger setup (`SetupLogger`)
- Signal handling for graceful shutdown (`SetupSignalContext`)
- Wiring `weaver.NewWeaver()` → `weaver.Weave()`
- Printing crawl summary via `web.String()`

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry point — wires config, logger, signal context, weaver, prints summary |
| `config.go` | `AnansiConfig` struct with JSON serialization, `OriginURL` |
| `consts.go` | Default flag values, exit codes, error format |
| `logger.go` | `SetupLogger` — slog JSON handler to stderr |
| `startup.go` | `ParseFlags`, `SetupSignalContext` |
| `config_test.go` | Config unit tests |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Crawl completed successfully |
| 1 | Error (invalid config, crawl failure) |
| 130 | Interrupted (SIGINT/Ctrl+C) |
