# testutil

Shared test helpers for the Anansi web crawler.

## Functions

| Function | Purpose |
|----------|---------|
| `SkipIfNoIntegration(t)` | Loads `.env.test` from project root, skips test if `GO_RUN_INTEGRATIONS != "true"` |

## Usage

```go
func TestSomething_LiveSite(t *testing.T) {
    testutil.SkipIfNoIntegration(t)
    t.Parallel()
    // ... live network test
}
```

## Design Notes

- **Project root found via `.git`** — walks up from `os.Getwd()`. Uses `.git` instead of `go.mod` so mono-repos with multiple modules find the correct root.
- **`.env.test` required** — `require.NoError` if the file is missing. Fails loudly, not silently.
- **Env vars override** — CI can set `GO_RUN_INTEGRATIONS=true` directly without the file.
