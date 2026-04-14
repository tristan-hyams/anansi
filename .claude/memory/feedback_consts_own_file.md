---
name: Constants in consts.go
description: Constants go in their own consts.go file, not inline in the file that uses them
type: feedback
originSessionId: 3f4c6514-8089-464c-8811-b878edd55863
---
Constants belong in a dedicated `consts.go` file within the package, not declared inline in the consuming file.

**Why:** Keeps files focused on a single responsibility - config logic shouldn't also define default values.

**How to apply:** When adding constants to a package, put them in `consts.go`. If the file already exists, add to it.
