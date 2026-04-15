# robots

Robot compliance for the Anansi web crawler. Two mechanisms:

## robots.txt (once at crawl start)

| Function | Purpose |
|----------|---------|
| `Fetch(ctx, client, baseURL, logger)` | Fetches and parses robots.txt |
| `Rules.IsAllowed(path)` | Checks path against `User-agent: *` Disallow rules |
| `Rules.CrawlDelay()` | Returns `Crawl-delay` directive as `time.Duration` |

### Status Code Handling

- **404:** No file. Allow all.
- **403:** Treated as not found. CDN/static hosts (S3, CloudFront, Azure Blob, GCS) return 403 for missing files instead of 404 because the bucket doesn't allow public listing.
- **5xx / other 4xx:** Returned as error to the caller.
- **Network error:** Allow all with warning log.

## X-Robots-Tag (per page response)

| Function | Purpose |
|----------|---------|
| `ParseXRobotsTag(header)` | Parses `X-Robots-Tag` HTTP header into directives |
| `Directives.ShouldFollow()` | Returns false if `nofollow` or `none` is set |

Checked on every page response. No additional HTTP request needed — the header is on the GET response the crawler already makes.

### Supported Directives

| Directive | Effect |
|-----------|--------|
| `noindex` | Not relevant for crawling (search engine concern) |
| `nofollow` | Skip link extraction for this page |
| `none` | Equivalent to `noindex, nofollow` |
| `follow` | Proceed normally (default) |
