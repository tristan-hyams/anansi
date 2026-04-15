# webutil

Shared HTTP client infrastructure for the Anansi web crawler.

## Usage

```go
// Create once — shared connection pool.
transport := webutil.NewTransport()

// Create per-worker — each gets its own client, shared pool.
client := webutil.NewClient(30*time.Second, transport)
```

## Why

- **One Transport, many Clients.** The `http.Transport` is the connection pool (TCP, TLS, keep-alives). It's safe for concurrent use and should be shared. The `http.Client` holds per-request config (timeout, redirects) and should be per-worker.
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
| Response header timeout | 30s | Don't wait forever for slow servers |
| HTTP/2 | Enabled | Better multiplexing if server supports it |
