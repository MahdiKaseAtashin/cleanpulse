package cleanup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteFilesDryRunDoesNotDelete(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "sample.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	results := DeleteFiles([]string{filePath}, true)
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if results[0].Deleted {
		t.Fatalf("expected no deletion in dry run")
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("file should still exist: %v", err)
	}
}

func TestDeleteFilesDeletesWhenNotDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "delete-me.txt")
	if err := os.WriteFile(filePath, []byte("bye"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	results := DeleteFiles([]string{filePath}, false)
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if !results[0].Deleted {
		t.Fatalf("expected deletion success, err: %v", results[0].Err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, got err: %v", err)
	}
}
