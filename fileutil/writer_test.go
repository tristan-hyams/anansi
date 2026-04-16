package fileutil

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/tristan-hyams/anansi/weaver"
)

func TestWriteOutputFiles_CreatesFiles(t *testing.T) {
	// Run in a temp directory to avoid polluting the project.
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

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

	// Check markdown file exists and has content.
	md, err := os.ReadFile(outputResultsFile)
	if err != nil {
		t.Fatalf("reading %s: %v", outputResultsFile, err)
	}
	if len(md) == 0 {
		t.Fatal("markdown file is empty")
	}

	// Check JSON file exists and is valid JSON.
	js, err := os.ReadFile(outputJSONFile)
	if err != nil {
		t.Fatalf("reading %s: %v", outputJSONFile, err)
	}
	if len(js) == 0 {
		t.Fatal("JSON file is empty")
	}

	// No errors, so error file should not exist.
	if _, err := os.Stat(outputErrorsFile); err == nil {
		t.Fatal("error file should not exist when there are no errors")
	}

	// Status messages written to stderr.
	out := stderr.String()
	if !bytes.Contains([]byte(out), []byte("results written to")) {
		t.Fatal("expected status message for results file")
	}
	if !bytes.Contains([]byte(out), []byte("json written to")) {
		t.Fatal("expected status message for JSON file")
	}
}

func TestWriteOutputFiles_CreatesErrorFile(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

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

	errData, err := os.ReadFile(outputErrorsFile)
	if err != nil {
		t.Fatalf("reading %s: %v", outputErrorsFile, err)
	}
	if len(errData) == 0 {
		t.Fatal("error file is empty")
	}
}
