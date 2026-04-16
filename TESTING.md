# Testing

## Running Tests

```bash
make test        # Unit tests with coverage
make lint        # Revive linter
```

## Integration Tests

Live network tests hit `crawlme.monzo.com` and are gated by `.env.test`. Disabled by default.

```bash
# Edit .env.test → set GO_RUN_INTEGRATIONS=true
make test
```

Integration tests exist in:
- `parser/parser_integration_test.go`
- `robots/robots_integration_test.go`
- `weaver/weaver_integration_test.go`

## Race Detector

`-race` requires CGO. Enabled in CI/Docker, not in local Windows builds.

```bash
# In Docker (CGO available)
CGO_ENABLED=1 go test -race ./...
```

## Test Strategy

| Approach | Packages |
|----------|----------|
| Table-driven unit tests | `normalizer`, `parser`, `robots`, `fileutil` |
| `httptest.NewServer` with canned HTML | `weaver`, `robots` |
| HTML fixtures in `testdata/` | `parser` |
| Singleton/transport assertions | `webutil` |
| Temp directory file I/O | `fileutil` (writer) |
| Live network integration | `parser`, `robots`, `weaver` |

## Test Categories (`parser/testdata/`)

| Directory | Scenario |
|-----------|----------|
| `simple/` | Happy path, 3 pages linking to each other |
| `cycle/` | A -> B -> A cycle detection |
| `external_links/` | Mix of internal and external links |
| `fragments/` | `#section` and `/page#section` handling |
| `non_html/` | Endpoints returning `application/json`, images |
| `malformed/` | Broken HTML, unclosed tags |
| `relative_urls/` | `../sibling`, `./child`, `//protocol-relative` |
| `schemes/` | `mailto:`, `javascript:void(0)`, `tel:` |
