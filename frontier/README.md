# frontier

URL queue and visited tracking for the Anansi web crawler. Interface-based for swappability.

The term "frontier" comes from graph traversal — it's the boundary between explored and unexplored nodes. Standard terminology in web crawling literature (Mercator, IRLbot).

## Interface

```go
type Frontier interface {
    Enqueue(ctx context.Context, fu *FrontierURL) error
    Dequeue(ctx context.Context) (*FrontierURL, error)
    Done()
    Pending() int32
    IsDone() bool
    Len() int
    Clear()
}
```

## Key Types

| Type | Purpose |
|------|---------|
| `Frontier` | Interface — queue + dedup + completion tracking |
| `InMemory` | Implementation — buffered channel + `sync.Map` + `atomic.Int32` pending counter |
| `FrontierURL` | URL wrapper with `Depth` metadata |

## Design Notes

- **Dedup built into Enqueue** — `sync.Map.LoadOrStore` checks and stores atomically. Duplicates logged at debug level.
- **Pending counter** — `atomic.Int32` incremented on `Enqueue`, decremented via `Done()`. `IsDone()` checks `pending <= 0 AND queue empty` — deterministic completion detection, no polling races.
- **Buffer defaults to 100,000** — the visited set is the real bound on growth, not the channel. A single subdomain has finite pages.
- **`select` multiplexing** — both Enqueue and Dequeue use `select` with `ctx.Done()`. Tests prove modern Go's `select` correctly picks up whichever case fires first without a `default` + sleep loop.
- **Production swap** — replace `InMemory` with a Redis/RabbitMQ implementation via the `Frontier` interface.
