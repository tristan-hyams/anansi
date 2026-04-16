package fileutil

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tristan-hyams/anansi/weaver"
)

// chdirTemp changes to a temp directory and restores on cleanup.
func chdirTemp(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
}

func TestWriteOutputFiles_CreatesFiles(t *testing.T) {
	chdirTemp(t)

	web := &weaver.Web{
		Visited:   1,
		OriginURL: "https://example.com/",
		Duration:  time.Second,
		Pages: []weaver.PageResult{
			{
				URL: "https://example.com/", Links: 2, Status: 200,
				ContentType: "text/html", Duration: 50 * time.Millisecond,
				Timestamp: time.Now(),
			},
		},
	}

	var stderr bytes.Buffer
	if err := WriteOutputFiles(web, &stderr); err != nil {
		t.Fatalf("WriteOutputFiles failed: %v", err)
	}

	assertFileNotEmpty(t, outputResultsFile)
	assertFileNotEmpty(t, outputJSONFile)

	// No errors, so error file should not exist.
	if _, err := os.Stat(outputErrorsFile); err == nil {
		t.Fatal("error file should not exist when there are no errors")
	}

	out := stderr.String()
	assertContains(t, out, "results written to")
	assertContains(t, out, "json written to")
}

func TestWriteOutputFiles_CreatesErrorFile(t *testing.T) {
	chdirTemp(t)

	web := &weaver.Web{
		Skipped:   1,
		OriginURL: "https://example.com/",
		Duration:  time.Second,
		Pages: []weaver.PageResult{
			{
				URL: "https://example.com/fail", Error: os.ErrNotExist,
				Depth: 1, Timestamp: time.Now(),
			},
		},
	}

	var stderr bytes.Buffer
	if err := WriteOutputFiles(web, &stderr); err != nil {
		t.Fatalf("WriteOutputFiles failed: %v", err)
	}

	assertFileNotEmpty(t, outputErrorsFile)
}

func assertFileNotEmpty(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if len(data) == 0 {
		t.Fatalf("%s is empty", path)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("expected output to contain %q", substr)
	}
}
