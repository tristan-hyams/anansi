# cmd/anansi

CLI entry point for the Anansi web crawler.

Thin wiring layer — all logic lives in packages. This is the only place that calls `os.Exit`.

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

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Crawl completed successfully |
| 1 | Error (invalid config, crawl failure) |
| 130 | Interrupted (SIGINT/Ctrl+C) |
