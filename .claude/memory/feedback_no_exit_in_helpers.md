---
name: No os.Exit or log.Fatal in helpers
description: Helper/factory functions must return errors, never call os.Exit or log.Fatal — only main() decides exit behavior
type: feedback
originSessionId: 3f4c6514-8089-464c-8811-b878edd55863
---
Helper methods and factory functions must never call `os.Exit` or `log.Fatal`. Always return an `error` and let the caller decide what to do.

**Why:** User wants callers to retain control over error handling and exit behavior. Helpers that exit break composability and testability.

**How to apply:** Any function other than `main()` should return errors up the stack. Only `main()` calls `os.Exit`. This applies to `parseFlags`, config loaders, setup functions, etc.
