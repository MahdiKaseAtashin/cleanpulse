package ui

import (
	"fmt"
	"strings"
	"time"

	"duplica-scan/src/internal/duplicates"
	"duplica-scan/src/internal/scanner"
)

type Console struct {
	lastScanUpdate time.Time
	lastHashUpdate time.Time
}

func NewConsole() *Console {
	return &Console{}
}

func (c *Console) OnScanProgress(p scanner.Progress) {
	if time.Since(c.lastScanUpdate) < 150*time.Millisecond {
		return
	}
	c.lastScanUpdate = time.Now()
	fmt.Printf("\r[scan] files: %-8d size: %-10s current: %s", p.FilesSeen, formatBytes(p.BytesSeen), trimPath(p.Current, 60))
}

func (c *Console) OnHashProgress(p duplicates.Progress) {
	if time.Since(c.lastHashUpdate) < 100*time.Millisecond {
		return
	}
	c.lastHashUpdate = time.Now()

	percent := float64(0)
	if p.TotalToHash > 0 {
		percent = (float64(p.HashedFiles) / float64(p.TotalToHash)) * 100
	}

	fmt.Printf("\r[hash] %6.2f%% (%d/%d) current: %s", percent, p.HashedFiles, p.TotalToHash, trimPath(p.CurrentPath, 60))
}

func (c *Console) PrintDuplicateGroups(groups []duplicates.Group) {
	fmt.Println()
	fmt.Println()
	fmt.Println("Duplicate groups:")
	if len(groups) == 0 {
		fmt.Println("- No duplicates found.")
		return
	}

	for i, group := range groups {
		fmt.Printf("\nGroup %d | size: %s | hash: %s\n", i+1, formatBytes(group.Size), group.Hash)
		for _, file := range group.Files {
			fmt.Printf("  - name: %s\n", file.Name)
			fmt.Printf("    path: %s\n", file.Path)
			fmt.Printf("    size: %s\n", formatBytes(file.Size))
		}
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func trimPath(path string, max int) string {
	if len(path) <= max {
		return path
	}
	if max < 8 {
		return path[:max]
	}
	return "..." + strings.TrimPrefix(path[len(path)-max+3:], `\`)
}
