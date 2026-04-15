# frontier

URL queue and visited tracking for the Anansi web crawler. Interface-based for swappability.

The term "frontier" comes from graph traversal — it's the boundary between explored and unexplored nodes. Standard terminology in web crawling literature (Mercator, IRLbot).

## Interface

```go
type Frontier interface {
    Enqueue(ctx context.Context, fu *FrontierURL) error
    Dequeue(ctx context.Context) (*FrontierURL, error)
    Clear()
}
```

## Key Types

| Type | Purpose |
|------|---------|
| `Frontier` | Interface — queue + dedup contract |
| `InMemory` | Implementation — buffered channel + `sync.Map` |
| `FrontierURL` | URL wrapper with `Depth`, `Status`, `Err` metadata |
| `Status` | Enum: `StatusPending`, `StatusVisited`, `StatusError` |

## Design Notes

- **Dedup built into Enqueue** — `sync.Map.LoadOrStore` checks and stores atomically. Duplicates logged at debug level.
- **Buffer defaults to 100,000** — the visited set is the real bound on growth, not the channel. A single subdomain has finite pages.
- **No WaitGroup** — lifecycle management belongs in the crawler, not the queue.
- **`select` multiplexing** — both Enqueue and Dequeue use `select` with `ctx.Done()`. Tests prove modern Go's `select` correctly picks up whichever case fires first without a `default` + sleep loop.
- **Production swap** — replace `InMemory` with a Redis/RabbitMQ implementation via the `Frontier` interface.
