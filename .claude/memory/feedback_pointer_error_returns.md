---
name: Pointer returns on error paths
description: Functions returning structs on error should use pointers so they can return nil, err - never zero-value structs
type: feedback
originSessionId: 3f4c6514-8089-464c-8811-b878edd55863
---
Use pointer return types for functions that construct structs, so error paths return `nil, err` instead of an empty/zero-value struct.

**Why:** Returning a zero-value struct on error lets callers accidentally use an uninitialised value. A nil pointer forces callers to handle the error.

**How to apply:** Factory functions, parsers, loaders - anything returning a struct alongside an error should return `*T, error`, not `T, error`.
