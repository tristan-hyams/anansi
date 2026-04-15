# normalizer

URL canonicalization for the Anansi web crawler. Pure functions, zero side effects.

## Functions

| Function | Purpose |
|----------|---------|
| `Normalize(base, raw)` | Resolves relative URLs, strips fragments, lowercases scheme/host, strips default ports |
| `IsSameHost(origin, candidate)` | Strict hostname match — `crawlme.monzo.com` does not match `monzo.com` |
| `IsFollowableScheme(u)` | Returns true for `http` and `https` only |

## Design Notes

- **Trailing slashes not normalized.** Per RFC 3986 Section 6, `/path` and `/path/` are distinct URIs. The origin server decides whether they return the same content.
- **Default port stripping** uses `net.SplitHostPort` for safe parsing — no string suffix ambiguity with ports like `:8080` vs `:80`.
- **Host comparison** is case-insensitive and default-port-aware.
