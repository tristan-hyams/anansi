package reporting

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tristan-hyams/anansi/weaver"
)

func TestWriteOutputFiles_CreatesFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
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
	if err := WriteOutputFiles(web, dir, &stderr); err != nil {
		t.Fatalf("WriteOutputFiles failed: %v", err)
	}

	assertFileNotEmpty(t, filepath.Join(dir, outputResultsFile))
	assertFileNotEmpty(t, filepath.Join(dir, outputJSONFile))

	// No errors, so error file should not exist.
	if _, err := os.Stat(filepath.Join(dir, outputErrorsFile)); err == nil {
		t.Fatal("error file should not exist when there are no errors")
	}

	out := stderr.String()
	assertContains(t, out, "results written to")
	assertContains(t, out, "json written to")
}

func TestWriteOutputFiles_CreatesErrorFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
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
	if err := WriteOutputFiles(web, dir, &stderr); err != nil {
		t.Fatalf("WriteOutputFiles failed: %v", err)
	}

	assertFileNotEmpty(t, filepath.Join(dir, outputErrorsFile))
}

func TestCreateOutputDir(t *testing.T) {
	t.Parallel()

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	dir, err := CreateOutputDir()
	if err != nil {
		t.Fatalf("CreateOutputDir failed: %v", err)
	}

	if !strings.HasPrefix(dir, "output") {
		t.Fatalf("expected dir under output/, got %s", dir)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("output dir does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("output path is not a directory")
	}
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
