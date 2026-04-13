package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanWithOptionsFiltersByExtDirAndSize(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "keep.txt"), 20)
	mustWriteFile(t, filepath.Join(root, "skip.log"), 30)

	skipDir := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(skipDir, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	mustWriteFile(t, filepath.Join(skipDir, "inside.txt"), 20)
	mustWriteFile(t, filepath.Join(root, "small.txt"), 3)
	mustWriteFile(t, filepath.Join(root, "large.txt"), 200)

	summary, err := ScanWithOptions(root, nil, ScanOptions{
		ExcludeExtensions: map[string]struct{}{".log": {}},
		ExcludeDirs:       map[string]struct{}{"node_modules": {}},
		MinSizeBytes:      10,
		MaxSizeBytes:      100,
	})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(summary.Files) != 1 {
		t.Fatalf("expected 1 file after filtering, got %d", len(summary.Files))
	}
	if summary.Files[0].Name != "keep.txt" {
		t.Fatalf("expected keep.txt, got %s", summary.Files[0].Name)
	}
}

func mustWriteFile(t *testing.T, path string, size int) {
	t.Helper()
	content := make([]byte, size)
	for i := range content {
		content[i] = 'a'
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
