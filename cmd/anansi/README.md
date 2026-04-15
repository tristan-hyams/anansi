# cmd/anansi

CLI entry point for the Anansi web crawler.

Thin wiring layer — all logic lives in packages. This is the only place that calls `os.Exit`.

## Responsibilities

- Flag parsing (`ParseFlags`)
- Config loading from JSON (`LoadConfigFromFile`)
- Structured logger setup (`SetupLogger`)
- Signal handling for graceful shutdown (`SetupSignalContext`)
- Wiring crawler dependencies and running the crawl

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry point — wires config, logger, signal context, crawler |
| `config.go` | `AnansiConfig` struct with JSON serialization, `ParseFlags`, `OriginURL` |
| `consts.go` | Default flag values (workers, rate, timeout, log level) |
| `logger.go` | `SetupLogger` — slog JSON handler to stderr |
| `startup.go` | `SetupSignalContext` — signal notification wiring |
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
