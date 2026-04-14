---
name: Use make targets for verification
description: Always use make build, make test, make lint — not raw go/revive commands — to match user's workflow
type: feedback
originSessionId: 3f4c6514-8089-464c-8811-b878edd55863
---
Always verify with `make build`, `make test`, `make lint` — never run `go build`, `go test`, or `revive` directly.

**Why:** The Makefile is the source of truth for how commands are run. Running raw commands can miss flags, env vars, or config paths that the Makefile sets. It also wastes time debugging discrepancies between what I run and what the user runs.

**How to apply:** Every time you need to check build/test/lint, use the make targets. Read the output carefully to catch issues early.
