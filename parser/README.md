# parser

HTML link extraction for the Anansi web crawler. Uses `golang.org/x/net/html` tokenizer.

## Functions

| Function | Purpose |
|----------|---------|
| `ExtractLinks(ctx, r)` | Scans HTML for `<a href>` tags, returns raw href strings |

## Design Notes

- **Tokenizer is a state machine** — sequential iteration, no parallelism within a single page parse. Parallelism belongs in the crawler (multiple workers parsing different pages).
- **Context checked each iteration** for cancellation of large or buffered documents.
- **Returns raw hrefs unfiltered** — normalization, scheme filtering, and domain scoping are the caller's responsibility.
- **Handles `SelfClosingTagToken`** for `<a href="/page"/>` edge cases.
- **`bytes.Equal`** comparison for href attribute key — avoids string allocation on every attribute check.
- **Only extracts `<a>` tags** — `<link>`, `<script>`, `<img>`, and `<area>` are ignored.
