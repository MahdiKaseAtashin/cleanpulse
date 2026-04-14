package devcleanup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteMarkdownAndHTMLReport(t *testing.T) {
	report := RunReport{
		GeneratedAt:    time.Now(),
		OS:             "windows",
		DryRun:         true,
		MaxRisk:        "moderate",
		Planned:        2,
		Attempted:      1,
		Skipped:        1,
		ReclaimedBytes: 1234,
		Duration:       2 * time.Second,
		Results: []ResultReportEntry{
			{ID: "npm-cache", Name: "npm cache", Category: "package-manager", Risk: "safe", Attempted: true, DeletedBytes: 1234},
		},
	}

	dir := t.TempDir()
	mdPath := filepath.Join(dir, "report.md")
	htmlPath := filepath.Join(dir, "report.html")

	if err := WriteMarkdownReport(mdPath, report); err != nil {
		t.Fatalf("write markdown report: %v", err)
	}
	if err := WriteHTMLReport(htmlPath, report); err != nil {
		t.Fatalf("write html report: %v", err)
	}

	if _, err := os.Stat(mdPath); err != nil {
		t.Fatalf("markdown report not found: %v", err)
	}
	if _, err := os.Stat(htmlPath); err != nil {
		t.Fatalf("html report not found: %v", err)
	}
}
