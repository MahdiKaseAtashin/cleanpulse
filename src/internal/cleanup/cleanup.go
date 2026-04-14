package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Result struct {
	Path       string
	Deleted    bool
	BackupPath string
	Err        error
}

type DeletionMode string

const (
	DeletionModeDelete     DeletionMode = "delete"
	DeletionModeQuarantine DeletionMode = "quarantine"
)

type DeleteOptions struct {
	DryRun        bool
	Mode          DeletionMode
	QuarantineDir string
}

// DeleteFiles removes selected files unless dryRun is enabled.
func DeleteFiles(paths []string, dryRun bool) []Result {
	return DeleteFilesWithOptions(paths, DeleteOptions{
		DryRun: dryRun,
		Mode:   DeletionModeDelete,
	})
}

func DeleteFilesWithOptions(paths []string, options DeleteOptions) []Result {
	if options.Mode == "" {
		options.Mode = DeletionModeDelete
	}

	results := make([]Result, 0, len(paths))
	for _, path := range paths {
		if options.DryRun {
			results = append(results, Result{
				Path:    path,
				Deleted: false,
				Err:     nil,
			})
			continue
		}

		switch options.Mode {
		case DeletionModeQuarantine:
			backupPath, err := moveToQuarantine(path, options.QuarantineDir)
			results = append(results, Result{
				Path:       path,
				Deleted:    err == nil,
				BackupPath: backupPath,
				Err:        err,
			})
		default:
			err := os.Remove(path)
			results = append(results, Result{
				Path:    path,
				Deleted: err == nil,
				Err:     err,
			})
		}
	}
	return results
}

func moveToQuarantine(path string, quarantineDir string) (string, error) {
	baseDir := stringsTrimSpace(quarantineDir)
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(home, ".duplica-scan", "quarantine")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", err
	}
	name := filepath.Base(path)
	target := filepath.Join(baseDir, fmt.Sprintf("%d-%s-%s", time.Now().UnixNano(), strconv.Itoa(os.Getpid()), name))
	if err := os.Rename(path, target); err != nil {
		return "", err
	}
	return target, nil
}

func stringsTrimSpace(v string) string {
	// local helper to avoid importing strings for one call.
	start := 0
	end := len(v)
	for start < end && (v[start] == ' ' || v[start] == '\t' || v[start] == '\n' || v[start] == '\r') {
		start++
	}
	for end > start && (v[end-1] == ' ' || v[end-1] == '\t' || v[end-1] == '\n' || v[end-1] == '\r') {
		end--
	}
	if start == 0 && end == len(v) {
		return v
	}
	return v[start:end]
}
