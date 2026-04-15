# webutil

Shared HTTP client infrastructure for the Anansi web crawler.

## Usage

```go
// Per-worker client — backed by singleton transport.
client := webutil.NewClient(30 * time.Second)

// Access the singleton transport directly (rarely needed).
transport := webutil.Transport()
```

## Why

- **Singleton Transport, many Clients.** `Transport()` returns a single `*http.Transport` via `sync.Once` — shared connection pool across all HTTP clients. `NewClient(timeout)` creates a per-worker client wrapping it.
- **Tuned for crawling.** Dial timeout, TLS timeout, idle connection limits, and keep-alive are configured for a long-lived single-domain crawl — not Go's conservative defaults.
- **Single-domain optimisation.** `MaxIdleConnsPerHost` equals `MaxIdleConns` since we're crawling one host.

## Configuration

| Setting | Value | Rationale |
|---------|-------|-----------|
| Dial timeout | 10s | Fail fast on unreachable hosts |
| Keep-alive | 30s | Reuse TCP connections across requests |
| Max idle conns | 100 | Connection pool size |
| Max idle per host | 100 | Same as total — single domain |
| Idle conn timeout | 90s | Close stale connections |
| TLS handshake timeout | 10s | Fail fast on TLS issues |
| Response header timeout | 10s | Fail fast if server goes silent |
| HTTP/2 | Enabled | Better multiplexing if server supports it |
