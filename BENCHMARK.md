# Benchmarks & Profiling

## Running Benchmarks

```bash
# All benchmarks
make bench

# Or individually (faster, avoids large frontier allocations)
go test -bench=BenchmarkNormalize -benchmem ./benchmark/...
go test -bench=BenchmarkExtractLinks -benchmem ./benchmark/...
go test -bench=BenchmarkFrontier -benchmem ./benchmark/...
go test -bench=BenchmarkComputeStats -benchmem ./benchmark/...

# With shorter benchtime for quick checks
go test -bench=. -benchmem -benchtime=500ms ./benchmark/...
```

## Benchmark Suite

Benchmarks live in `benchmark/` and target hot-path functions that run per-page or per-link. With 42,000 pages averaging 10+ links each, these functions execute hundreds of thousands of times per crawl.

### Normalizer (`benchmark/normalizer_bench_test.go`)

| Benchmark | What it measures |
|-----------|------------------|
| `BenchmarkNormalize_Relative` | Relative path resolution (`../products/123`) |
| `BenchmarkNormalize_Absolute` | Absolute URL normalization |
| `BenchmarkNormalize_Fragment` | Fragment stripping (`/page#section`) |
| `BenchmarkNormalize_QueryParams` | Query parameter handling |
| `BenchmarkIsSameHost_Match` | Host comparison (same domain) |
| `BenchmarkIsSameHost_NoMatch` | Host comparison (different domain) |

### Parser (`benchmark/parser_bench_test.go`)

| Benchmark | What it measures |
|-----------|------------------|
| `BenchmarkExtractLinks_Small` | 8-link HTML page (~15 lines) |
| `BenchmarkExtractLinks_Large100Links` | 100-link HTML page (~50KB, generated) |

### Frontier (`benchmark/frontier_bench_test.go`)

| Benchmark | What it measures |
|-----------|------------------|
| `BenchmarkFrontierEnqueue` | Enqueue throughput including `sync.Map` dedup + channel send |
| `BenchmarkFrontierDequeue` | Dequeue + Done throughput from a pre-filled queue |

### Stats (`benchmark/stats_bench_test.go`)

| Benchmark | What it measures |
|-----------|------------------|
| `BenchmarkComputeStats_1k` | Stats aggregation over 1,000 PageResults |
| `BenchmarkComputeStats_10k` | Stats aggregation over 10,000 PageResults |

## pprof

pprof is enabled via the `ANANSI_DEBUG` environment variable. When set, a debug HTTP server starts on `localhost:6060` with all standard pprof endpoints.

```bash
# Start a crawl with pprof enabled
ANANSI_DEBUG=1 bin/anansi -workers 20 -rate 5000 -max-depth 0 https://crawlme.monzo.com/

# In another terminal:
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30   # CPU
go tool pprof http://localhost:6060/debug/pprof/heap                 # Heap
curl http://localhost:6060/debug/pprof/goroutine?debug=2             # Goroutines
```

### What to profile and why

| Profile | Target | What it reveals |
|---------|--------|-----------------|
| CPU | `normalizer.Normalize`, `parser.ExtractLinks` | Hot path per page. Reveals whether time is in our code or the standard library. |
| Heap | `frontier.InMemory`, `weaver.pages` | `sync.Map` and `[]PageResult` grow monotonically. Quantifies per-page memory cost and whether `FoundLinks []string` dominates. |
| Goroutine | Worker pool lifecycle | Expect N+2 goroutines (N crawlers + monitor + main). Reveals stalls: rate limiter, frontier channel, or HTTP I/O. |
| Mutex | `weaver.mu`, `weaver.outputMu` | Separate mutexes for pages vs stdout. Reveals whether contention justifies the split at high worker counts. |
| Block | `frontier.Dequeue`, `rate.Limiter.Wait` | Expected: blocking in Dequeue and limiter. Unexpected blocking in recordPage/printPage signals contention. |

## Future benchmarks

With more time, the following would be added:

- **`BenchmarkWeave` integration benchmark** using `httptest.NewServer` with a canned 100-1000 page site to measure end-to-end crawler overhead without network variability
- **`BenchmarkRecordPage` contention** measuring mutex overhead with N goroutines appending PageResults
- **`BenchmarkParseXRobotsTag`** verifying sub-microsecond directive parsing

## Memory considerations

The current design stores all `PageResult` data in memory until the crawl completes. At 42k pages with `FoundLinks` populated, this is the dominant memory consumer.

- **Per-page cost:** `PageResult` struct + URL string + `FoundLinks` slice + content-type string
- **Total working set:** Tens of MB for a 42k page crawl
- **Growth:** Linear with page count. A streaming output approach (writing results as they arrive) would cap memory at O(workers) instead of O(pages), at the cost of more complex I/O coordination.
