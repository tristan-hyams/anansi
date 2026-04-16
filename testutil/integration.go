// Package testutil provides shared test helpers for Anansi.
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// SkipIfNoIntegration loads .env.test from the project root and skips
// the test if GO_RUN_INTEGRATIONS is not set to "true".
// Call this at the top of any integration test.
func SkipIfNoIntegration(t *testing.T) {
	t.Helper()

	root := findProjectRoot(t)
	envFile := filepath.Join(root, ".env.test")

	err := godotenv.Load(envFile)
	require.NoError(t, err, "failed to load .env.test from %s", root)

	if os.Getenv("GO_RUN_INTEGRATIONS") != "true" {
		t.Skip("integration tests disabled via GO_RUN_INTEGRATIONS")
	}
}

// findProjectRoot walks up from the current working directory looking
// for .git to identify the repository root. We could do `go.mod` instead
// but if we ended up with a mono-repo in the future, .git would be more reliable.
// since you will have multiple go.mod files per package.
func findProjectRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (no .git found)")
		}
		dir = parent
	}
}
