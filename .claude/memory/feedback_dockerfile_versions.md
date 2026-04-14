---
name: Dockerfile version policy
description: User manually updates Dockerfile base images to latest versions — don't revert or downgrade
type: feedback
originSessionId: 3f4c6514-8089-464c-8811-b878edd55863
---
User keeps Dockerfile base images at latest stable versions (e.g. golang:1.26-alpine, alpine:3.23). They update these themselves.

**Why:** User prefers staying on latest and manages version bumps manually.

**How to apply:** When creating or modifying Dockerfiles, use the latest versions visible in the current Dockerfile. Don't pin to older versions from docs/plan. Don't revert user's version bumps.
