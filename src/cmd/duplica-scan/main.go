package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"duplica-scan/src/internal/duplicates"
	"duplica-scan/src/internal/hash"
	"duplica-scan/src/internal/scanner"
	"duplica-scan/src/internal/ui"
)

func main() {
	rootPath := flag.String("path", "", "Drive or directory to scan (required)")
	dryRun := flag.Bool("dry-run", true, "Dry run mode: report duplicates without deletion")
	flag.Parse()

	if *rootPath == "" {
		fmt.Println("Usage: duplica-scan -path <directory_or_drive_root> [-dry-run=true]")
		os.Exit(1)
	}

	console := ui.NewConsole()
	start := time.Now()

	fmt.Printf("Scanning: %s\n", *rootPath)
	scanSummary, err := scanner.Scan(*rootPath, console.OnScanProgress)
	if err != nil {
		log.Fatalf("scan failed: %v", err)
	}
	fmt.Println()

	fmt.Printf("Collected %d files. Hashing candidate files...\n", len(scanSummary.Files))
	groups, hashErrors := duplicates.Detect(scanSummary.Files, hash.SHA256File, console.OnHashProgress)
	fmt.Println()

	console.PrintDuplicateGroups(groups)

	fmt.Println()
	fmt.Printf("Dry run mode: %t\n", *dryRun)
	fmt.Printf("Duplicate groups found: %d\n", len(groups))
	fmt.Printf("Scanner non-fatal errors: %d | Hash non-fatal errors: %d\n", len(scanSummary.Errors), len(hashErrors))
	fmt.Printf("Completed in %s\n", time.Since(start).Round(time.Millisecond))
}
