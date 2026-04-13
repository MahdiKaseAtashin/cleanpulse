# Duplica Scan

Windows-friendly duplicate file scanner written in Go.

## Features

- Scan a drive or directory recursively.
- Detect duplicates by content hash (SHA-256).
- Group duplicates and display name, full path, and size.
- Stream file hashing in chunks to keep memory usage low.
- Use size pre-filtering to avoid unnecessary hashing.
- Show real-time scan and hash progress.
- Review duplicate groups and select files to delete.
- Require explicit `YES` confirmation before deletion.
- Support dry run mode.

## Project Structure

```text
duplica-scan/
  src/
    cmd/duplica-scan/        # CLI entrypoint
    internal/model/          # shared data models
    internal/scanner/        # recursive file discovery
    internal/hash/           # chunk-based hashing
    internal/duplicates/     # duplicate detection engine
    internal/ui/             # console interaction and progress
    internal/cleanup/        # deletion workflow
  tests/                     # reserved for integration/e2e tests
  docs/                      # architecture notes and docs
```

## Run on Windows

### Prerequisites

- Go 1.22+ installed and available in `PATH`

### Build

```powershell
go build -o .\bin\duplica-scan.exe .\src\cmd\duplica-scan
```

### Scan (Dry Run)

```powershell
.\bin\duplica-scan.exe -path "D:\Data" -dry-run=true
```

### Scan and Delete (Interactive)

```powershell
.\bin\duplica-scan.exe -path "D:\Data" -dry-run=false
```

When prompted:
1. Enter comma-separated index numbers of files to delete in each group.
2. Type `YES` to confirm deletion.

## Development

```powershell
go test ./...
```
