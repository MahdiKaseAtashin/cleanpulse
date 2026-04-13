//go:build gui && cgo
// +build gui,cgo

package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"duplica-scan/src/internal/buildinfo"
	"duplica-scan/src/internal/duplicates"
	"duplica-scan/src/internal/hash"
	"duplica-scan/src/internal/report"
	"duplica-scan/src/internal/scanner"
	"duplica-scan/src/internal/selection"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

//go:embed logo.png
var appLogoPNG []byte

func main() {
	a := app.New()
	logoResource := fyne.NewStaticResource("duplica-scan-logo.png", appLogoPNG)
	a.SetIcon(logoResource)
	w := a.NewWindow(fmt.Sprintf("Duplica Scan %s", buildinfo.Version))
	w.SetIcon(logoResource)
	w.Resize(fyne.NewSize(980, 760))

	var scanView fyne.CanvasObject

	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("Select directory or drive root")

	hashWorkersEntry := widget.NewEntry()
	hashWorkersEntry.SetText(strconv.Itoa(runtime.NumCPU()))

	excludeExtsEntry := widget.NewEntry()
	excludeExtsEntry.SetPlaceHolder(".log,.tmp")

	excludeDirsEntry := widget.NewEntry()
	excludeDirsEntry.SetPlaceHolder("node_modules,.git")

	minSizeEntry := widget.NewEntry()
	minSizeEntry.SetText("0")
	maxSizeEntry := widget.NewEntry()
	maxSizeEntry.SetText("0")

	dryRunCheck := widget.NewCheck("Dry run (no deletion)", nil)
	dryRunCheck.SetChecked(true)

	autoSelectSelect := widget.NewSelect([]string{"none", "newest", "oldest"}, nil)
	autoSelectSelect.SetSelected("none")

	exportFormatSelect := widget.NewSelect([]string{"none", "csv", "json"}, nil)
	exportFormatSelect.SetSelected("none")

	exportPathEntry := widget.NewEntry()
	exportPathEntry.SetPlaceHolder("./reports/duplicate-report-*.json")

	statusLabel := widget.NewLabel("Ready")
	stepLabel := widget.NewLabel("")
	stepLabel.Hide()
	scanProgress := widget.NewProgressBarInfinite()
	scanProgress.Hide()
	hashProgress := widget.NewProgressBar()
	hashProgress.Hide()
	detailLabel := widget.NewLabel("")
	detailLabel.Hide()

	output := widget.NewMultiLineEntry()
	output.Wrapping = fyne.TextWrapWord
	output.Disable()

	appendOutput := func(text string) {
		fyne.Do(func() {
			output.SetText(output.Text + text + "\n")
		})
	}

	browseBtn := widget.NewButton("Browse...", func() {
		dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				return
			}
			pathEntry.SetText(uri.Path())
		}, w).Show()
	})

	var runBtn *widget.Button
	runBtn = widget.NewButton("Run Scan", func() {
		rootPath := strings.TrimSpace(pathEntry.Text)
		if rootPath == "" {
			dialog.ShowInformation("Missing path", "Please choose a directory or drive root to scan.", w)
			return
		}

		hashWorkers, err := parseInt(hashWorkersEntry.Text, runtime.NumCPU())
		if err != nil || hashWorkers < 1 {
			dialog.ShowInformation("Invalid hash workers", "Hash workers must be a positive integer.", w)
			return
		}
		minSize, err := parseInt64(minSizeEntry.Text, 0)
		if err != nil || minSize < 0 {
			dialog.ShowInformation("Invalid min size", "Min size must be a non-negative integer.", w)
			return
		}
		maxSize, err := parseInt64(maxSizeEntry.Text, 0)
		if err != nil || maxSize < 0 {
			dialog.ShowInformation("Invalid max size", "Max size must be a non-negative integer.", w)
			return
		}
		if maxSize > 0 && minSize > maxSize {
			dialog.ShowInformation("Invalid size range", "Min size cannot be greater than max size.", w)
			return
		}

		autoSelectRaw := strings.TrimSpace(autoSelectSelect.Selected)
		if autoSelectRaw == "none" {
			autoSelectRaw = ""
		}
		strategy, err := selection.NormalizeStrategy(autoSelectRaw)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		exportFormat := strings.TrimSpace(exportFormatSelect.Selected)
		if exportFormat == "none" {
			exportFormat = ""
		}
		exportPath := strings.TrimSpace(exportPathEntry.Text)
		if exportFormat != "" && exportPath == "" {
			exportPath = defaultExportPath(exportFormat)
		}

		output.SetText("")
		fyne.Do(func() {
			statusLabel.SetText("Running scan…")
			stepLabel.SetText("Step 1 of 2 · Scanning filesystem")
			stepLabel.Show()
			detailLabel.SetText("Files found: 0")
			detailLabel.Show()
			scanProgress.Show()
			hashProgress.Hide()
			hashProgress.SetValue(0)
		})
		runBtn.Disable()

		go func() {
			start := time.Now()
			filterOptions := scanner.ScanOptions{
				ExcludeExtensions: parseExtensions(excludeExtsEntry.Text),
				ExcludeDirs:       parseNames(excludeDirsEntry.Text),
				MinSizeBytes:      minSize,
				MaxSizeBytes:      maxSize,
			}

			var lastScanUpdate time.Time
			scanSummary, scanErr := scanner.ScanWithOptions(rootPath, func(p scanner.Progress) {
				now := time.Now()
				if now.Sub(lastScanUpdate) < 40*time.Millisecond && p.FilesSeen%50 != 0 {
					return
				}
				lastScanUpdate = now
				cur := p.Current
				if len(cur) > 72 {
					cur = "…" + cur[len(cur)-69:]
				}
				fyne.Do(func() {
					detailLabel.SetText(fmt.Sprintf("Files found: %d · %s", p.FilesSeen, cur))
				})
			}, filterOptions)
			if scanErr != nil {
				fyne.Do(func() {
					scanProgress.Hide()
					hashProgress.Hide()
					stepLabel.Hide()
					detailLabel.Hide()
					runBtn.Enable()
					statusLabel.SetText("Scan failed")
					dialog.ShowError(scanErr, w)
				})
				return
			}

			appendOutput(fmt.Sprintf("Scanned files: %d", len(scanSummary.Files)))

			fyne.Do(func() {
				stepLabel.SetText("Step 2 of 2 · Hashing candidate files")
				scanProgress.Hide()
				hashProgress.Show()
				hashProgress.SetValue(0)
				detailLabel.SetText("Preparing hash…")
			})

			groups, hashErrors := duplicates.DetectWithOptions(
				scanSummary.Files,
				hash.SHA256File,
				func(p duplicates.Progress) {
					fyne.Do(func() {
						if p.TotalToHash > 0 {
							hashProgress.SetValue(float64(p.HashedFiles) / float64(p.TotalToHash))
						}
						cur := p.CurrentPath
						if len(cur) > 64 {
							cur = "…" + cur[len(cur)-61:]
						}
						detailLabel.SetText(fmt.Sprintf("Hashed %d / %d · %s", p.HashedFiles, p.TotalToHash, cur))
					})
				},
				duplicates.DetectOptions{HashWorkers: hashWorkers},
			)
			fyne.Do(func() {
				hashProgress.SetValue(1)
			})

			appendOutput(fmt.Sprintf("Duplicate groups found: %d", len(groups)))
			appendOutput(fmt.Sprintf("Scanner non-fatal errors: %d", len(scanSummary.Errors)))
			appendOutput(fmt.Sprintf("Hash non-fatal errors: %d", len(hashErrors)))
			appendOutput("")
			appendOutput(renderGroups(groups))

			initialSelection := make(map[string]struct{})
			if strategy != selection.StrategyManual {
				for _, path := range selection.AutoSelect(groups, strategy) {
					initialSelection[path] = struct{}{}
				}
				appendOutput(fmt.Sprintf("Auto-select (%s) picked %d file(s).", strategy, len(initialSelection)))
			}

			sorted := append([]duplicates.Group(nil), groups...)
			sort.Slice(sorted, func(i, j int) bool {
				if sorted[i].Size == sorted[j].Size {
					return sorted[i].Hash < sorted[j].Hash
				}
				return sorted[i].Size > sorted[j].Size
			})

			onBack := func() {
				fyne.Do(func() {
					w.SetContent(scanView)
					statusLabel.SetText("Ready")
				})
			}
			resultsView := buildResultsView(w, onBack, groups, sorted, dryRunCheck.Checked, initialSelection, appendOutput)

			if exportFormat != "" {
				if err := report.Export(groups, exportFormat, exportPath); err != nil {
					appendOutput(fmt.Sprintf("Export failed: %v", err))
				} else {
					appendOutput(fmt.Sprintf("Report exported: %s", exportPath))
				}
			}

			fyne.Do(func() {
				scanProgress.Hide()
				hashProgress.Hide()
				stepLabel.Hide()
				detailLabel.Hide()
				runBtn.Enable()
				statusLabel.SetText(fmt.Sprintf("Done in %s", time.Since(start).Round(time.Millisecond)))
				w.SetContent(resultsView)
			})
		}()
	})

	form := widget.NewForm(
		widget.NewFormItem("Path", container.NewBorder(nil, nil, nil, browseBtn, pathEntry)),
		widget.NewFormItem("Hash workers", hashWorkersEntry),
		widget.NewFormItem("Exclude extensions", excludeExtsEntry),
		widget.NewFormItem("Exclude directories", excludeDirsEntry),
		widget.NewFormItem("Min size bytes", minSizeEntry),
		widget.NewFormItem("Max size bytes", maxSizeEntry),
		widget.NewFormItem("Auto-select", autoSelectSelect),
		widget.NewFormItem("Export format", exportFormatSelect),
		widget.NewFormItem("Export path", exportPathEntry),
	)

	scanView = container.NewBorder(
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Duplica Scan GUI %s", buildinfo.Version)),
			dryRunCheck,
			form,
			runBtn,
			statusLabel,
			stepLabel,
			scanProgress,
			hashProgress,
			detailLabel,
		),
		nil,
		nil,
		nil,
		container.NewVScroll(output),
	)
	w.SetContent(scanView)
	w.ShowAndRun()
}

func parseInt(raw string, fallback int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

func parseInt64(raw string, fallback int64) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

func defaultExportPath(format string) string {
	base := fmt.Sprintf("duplicate-report-%s", time.Now().Format("20060102-150405"))
	switch strings.ToLower(strings.TrimSpace(format)) {
	case report.FormatCSV:
		return filepath.Join(".", "reports", base+".csv")
	case report.FormatJSON:
		return filepath.Join(".", "reports", base+".json")
	default:
		return filepath.Join(".", "reports", base+".txt")
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

func parseExtensions(raw string) map[string]struct{} {
	return parseCSVSet(raw, func(v string) string {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			return ""
		}
		if !strings.HasPrefix(v, ".") {
			return "." + v
		}
		return v
	})
}

func parseNames(raw string) map[string]struct{} {
	return parseCSVSet(raw, func(v string) string {
		return strings.ToLower(strings.TrimSpace(v))
	})
}

func parseCSVSet(raw string, normalizer func(string) string) map[string]struct{} {
	result := make(map[string]struct{})
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return result
	}
	for _, part := range strings.Split(raw, ",") {
		value := normalizer(part)
		if value == "" {
			continue
		}
		result[value] = struct{}{}
	}
	return result
}

func renderGroups(groups []duplicates.Group) string {
	if len(groups) == 0 {
		return "No duplicates found."
	}
	var b strings.Builder
	for i, group := range groups {
		b.WriteString(fmt.Sprintf("Group %d | size: %d | hash: %s\n", i+1, group.Size, group.Hash))
		for _, file := range group.Files {
			b.WriteString(fmt.Sprintf("  - %s | %s | %d\n", file.Name, file.Path, file.Size))
		}
		b.WriteString("\n")
	}
	return b.String()
}

const (
	resultsGroupsPerPage    = 12
	resultsMaxFilesPerGroup = 80
)

// resultsTableRow is one logical row in the duplicate-files table.
type resultsTableRow struct {
	path         string
	name         string
	size         int64
	groupNum     int
	fileNum      int
	overflowNote string // non-empty => informational row (no checkbox)
}

func buildResultsView(
	parent fyne.Window,
	onBack func(),
	originalGroups []duplicates.Group,
	sortedGroups []duplicates.Group,
	dryRun bool,
	initialSelection map[string]struct{},
	appendOutput func(string),
) fyne.CanvasObject {
	totalGroupCount := len(sortedGroups)

	selected := make(map[string]struct{}, 512)
	for p := range initialSelection {
		selected[p] = struct{}{}
	}
	var pageRows []resultsTableRow

	totalFiles := 0
	totalReclaimable := int64(0)
	for _, g := range originalGroups {
		totalFiles += len(g.Files)
		if len(g.Files) > 1 {
			totalReclaimable += int64(len(g.Files)-1) * g.Size
		}
	}

	countLabel := widget.NewLabel("")
	summaryLabel := widget.NewLabel(
		fmt.Sprintf(
			"Groups: %d | Candidate files: %d | Estimated reclaimable: %s | Mode: %s",
			len(originalGroups),
			totalFiles,
			formatBytes(totalReclaimable),
			map[bool]string{true: "Dry run (scan only)", false: "Delete"}[dryRun],
		),
	)

	updateCount := func() {
		countLabel.SetText(fmt.Sprintf("Selected files: %d", len(selected)))
	}

	allPathsFromSorted := func() []string {
		paths := make([]string, 0, totalFiles)
		for _, g := range sortedGroups {
			for _, f := range g.Files {
				paths = append(paths, f.Path)
			}
		}
		return paths
	}

	currentPage := 0
	totalPages := (len(sortedGroups) + resultsGroupsPerPage - 1) / resultsGroupsPerPage
	if totalPages < 1 {
		totalPages = 1
	}

	pageLabel := widget.NewLabel("")
	firstBtn := widget.NewButton("First", func() {})
	prevBtn := widget.NewButton("Previous", func() {})
	nextBtn := widget.NewButton("Next", func() {})
	lastBtn := widget.NewButton("Last", func() {})

	var resultsTable *widget.Table

	rebuildPage := func() {
		start := currentPage * resultsGroupsPerPage
		end := start + resultsGroupsPerPage
		if end > len(sortedGroups) {
			end = len(sortedGroups)
		}

		pageLabel.SetText(fmt.Sprintf("Page %d / %d", currentPage+1, totalPages))
		firstBtn.Disable()
		prevBtn.Disable()
		nextBtn.Disable()
		lastBtn.Disable()
		if currentPage > 0 {
			firstBtn.Enable()
			prevBtn.Enable()
		}
		if currentPage < totalPages-1 {
			nextBtn.Enable()
			lastBtn.Enable()
		}

		pageRows = nil
		if len(sortedGroups) == 0 {
			if resultsTable != nil {
				resultsTable.Refresh()
			}
			return
		}

		globalIdx := start
		for _, group := range sortedGroups[start:end] {
			gnum := globalIdx + 1
			globalIdx++

			files := group.Files
			totalFilesInGroup := len(files)
			overflow := 0
			if len(files) > resultsMaxFilesPerGroup {
				overflow = len(files) - resultsMaxFilesPerGroup
				files = files[:resultsMaxFilesPerGroup]
			}

			for fi, file := range files {
				pageRows = append(pageRows, resultsTableRow{
					path:     file.Path,
					name:     file.Name,
					size:     file.Size,
					groupNum: gnum,
					fileNum:  fi + 1,
				})
			}
			if overflow > 0 {
				pageRows = append(pageRows, resultsTableRow{
					overflowNote: fmt.Sprintf(
						"… files %d–%d not shown (%d more; %d file(s) in this group — use Export for the full list).",
						resultsMaxFilesPerGroup+1, totalFilesInGroup, overflow, totalFilesInGroup,
					),
				})
			}
		}

		if resultsTable != nil {
			resultsTable.ScrollToTop()
			resultsTable.Refresh()
		}
	}

	createTableCell := func() fyne.CanvasObject {
		chk := widget.NewCheck("", nil)
		lab := widget.NewLabel("")
		lab.Wrapping = fyne.TextWrapOff
		lab.Truncation = fyne.TextTruncateEllipsis
		return container.NewStack(lab, chk)
	}

	updateTableCell := func(id widget.TableCellID, obj fyne.CanvasObject) {
		if id.Row < 0 || id.Col < 0 {
			return
		}
		if id.Row >= len(pageRows) {
			return
		}
		row := pageRows[id.Row]
		st := obj.(*fyne.Container)
		lab := st.Objects[0].(*widget.Label)
		chk := st.Objects[1].(*widget.Check)

		if row.overflowNote != "" {
			chk.Hide()
			lab.Show()
			switch id.Col {
			case 0, 1, 2, 3, 5:
				lab.SetText("")
			case 4:
				lab.Wrapping = fyne.TextWrapOff
				lab.Truncation = fyne.TextTruncateEllipsis
				lab.SetText(row.overflowNote)
			}
			return
		}

		path := row.path
		switch id.Col {
		case 0:
			lab.Hide()
			chk.Show()
			_, on := selected[path]
			chk.SetChecked(on)
			chk.OnChanged = func(on bool) {
				if on {
					selected[path] = struct{}{}
				} else {
					delete(selected, path)
				}
				updateCount()
			}
		case 1:
			chk.Hide()
			lab.Show()
			lab.Wrapping = fyne.TextWrapOff
			lab.Truncation = fyne.TextTruncateEllipsis
			lab.SetText(strconv.Itoa(row.fileNum))
		case 2:
			chk.Hide()
			lab.Show()
			lab.Wrapping = fyne.TextWrapOff
			lab.Truncation = fyne.TextTruncateEllipsis
			lab.SetText(fmt.Sprintf("%d / %d", row.groupNum, totalGroupCount))
		case 3:
			chk.Hide()
			lab.Show()
			lab.Wrapping = fyne.TextWrapOff
			lab.Truncation = fyne.TextTruncateEllipsis
			lab.SetText(row.name)
		case 4:
			chk.Hide()
			lab.Show()
			lab.Wrapping = fyne.TextWrapOff
			lab.Truncation = fyne.TextTruncateEllipsis
			lab.SetText(row.path)
		case 5:
			chk.Hide()
			lab.Show()
			lab.Wrapping = fyne.TextWrapOff
			lab.Truncation = fyne.TextTruncateEllipsis
			lab.SetText(formatBytes(row.size))
		}
	}

	resultsTable = widget.NewTable(
		func() (int, int) { return len(pageRows), 6 },
		createTableCell,
		updateTableCell,
	)
	resultsTable.ShowHeaderRow = true
	resultsTable.ShowHeaderColumn = false
	resultsTable.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		l := o.(*widget.Label)
		l.TextStyle = fyne.TextStyle{Bold: true}
		if id.Row != -1 || id.Col < 0 {
			return
		}
		headers := []string{"Select", "#", "Group", "Name", "Path", "Size"}
		if id.Col < len(headers) {
			l.SetText(headers[id.Col])
		}
	}
	resultsTable.SetColumnWidth(0, 72)
	resultsTable.SetColumnWidth(1, 40)
	resultsTable.SetColumnWidth(2, 88)
	resultsTable.SetColumnWidth(3, 160)
	resultsTable.SetColumnWidth(4, 320)
	resultsTable.SetColumnWidth(5, 112)

	syncVisibleChecks := func() {
		if resultsTable != nil {
			resultsTable.Refresh()
		}
		updateCount()
	}

	setSelection := func(paths []string) {
		selected = make(map[string]struct{}, len(paths))
		for _, p := range paths {
			selected[p] = struct{}{}
		}
		syncVisibleChecks()
	}

	selectAllBtn := widget.NewButton("Select All", func() {
		setSelection(allPathsFromSorted())
	})

	clearBtn := widget.NewButton("Clear", func() {
		setSelection(nil)
	})

	keepNewestBtn := widget.NewButton("Keep Newest", func() {
		setSelection(selection.AutoSelect(originalGroups, selection.StrategyNewest))
	})

	keepOldestBtn := widget.NewButton("Keep Oldest", func() {
		setSelection(selection.AutoSelect(originalGroups, selection.StrategyOldest))
	})

	deleteLabel := "Delete"

	confirmAndDelete := func() {
		if len(selected) == 0 {
			dialog.ShowInformation("No selection", "Select at least one file.", parent)
			return
		}
		paths := make([]string, 0, len(selected))
		for p := range selected {
			paths = append(paths, p)
		}
		sort.Strings(paths)

		title := "Are you sure?"
		message := fmt.Sprintf(
			"This will permanently delete %d selected file(s) from disk. This cannot be undone.",
			len(paths),
		)
		dialog.NewConfirm(
			title,
			message,
			func(ok bool) {
				if !ok {
					return
				}
				n := len(paths)
				prog := widget.NewProgressBar()
				prog.Max = 1
				status := widget.NewLabel("")
				body := container.NewVBox(
					status,
					prog,
				)
				dlg := dialog.NewCustomWithoutButtons("Deleting files", body, parent)
				dlg.Show()

				go func() {
					failures := 0
					for i, path := range paths {
						idx := i + 1
						fyne.Do(func() {
							if n > 0 {
								prog.SetValue(float64(idx) / float64(n))
								status.SetText(fmt.Sprintf("Deleting %d of %d (%.0f%%)…", idx, n, 100*float64(idx)/float64(n)))
							}
						})
						err := os.Remove(path)
						if err != nil {
							failures++
							appendOutput(fmt.Sprintf("Failed: %s (%v)", path, err))
						}
					}
					fyne.Do(func() {
						dlg.Hide()
						appendOutput(fmt.Sprintf("Result action completed. Success: %d, Failed: %d", n-failures, failures))
						onBack()
					})
				}()
			},
			parent,
		).Show()
	}

	bottomDeleteBtn := widget.NewButton(deleteLabel, confirmAndDelete)
	bottomDeleteBtn.Importance = widget.DangerImportance

	exportCSVBtn := widget.NewButton("Export CSV", func() {
		path := defaultExportPath(report.FormatCSV)
		if err := report.Export(originalGroups, report.FormatCSV, path); err != nil {
			dialog.ShowError(err, parent)
			return
		}
		appendOutput("CSV exported: " + path)
		dialog.ShowInformation("Export complete", "CSV exported to:\n"+path, parent)
	})

	exportJSONBtn := widget.NewButton("Export JSON", func() {
		path := defaultExportPath(report.FormatJSON)
		if err := report.Export(originalGroups, report.FormatJSON, path); err != nil {
			dialog.ShowError(err, parent)
			return
		}
		appendOutput("JSON exported: " + path)
		dialog.ShowInformation("Export complete", "JSON exported to:\n"+path, parent)
	})

	firstBtn.OnTapped = func() {
		if currentPage > 0 {
			currentPage = 0
			rebuildPage()
		}
	}
	prevBtn.OnTapped = func() {
		if currentPage > 0 {
			currentPage--
			rebuildPage()
		}
	}
	nextBtn.OnTapped = func() {
		if currentPage < totalPages-1 {
			currentPage++
			rebuildPage()
		}
	}
	lastBtn.OnTapped = func() {
		if currentPage < totalPages-1 {
			currentPage = totalPages - 1
			rebuildPage()
		}
	}

	updateCount()

	toolbar := container.NewHBox(
		selectAllBtn, clearBtn, keepNewestBtn, keepOldestBtn,
		exportCSVBtn, exportJSONBtn,
	)
	toolbarScroll := container.NewHScroll(toolbar)

	paginationBar := container.NewHBox(
		layout.NewSpacer(),
		firstBtn,
		prevBtn,
		pageLabel,
		nextBtn,
		lastBtn,
		layout.NewSpacer(),
	)

	cancelBtn := widget.NewButton("Back to scan", func() {
		onBack()
	})
	actionBar := container.NewHBox(layout.NewSpacer(), cancelBtn, bottomDeleteBtn)

	bottomStack := container.NewVBox(
		paginationBar,
		actionBar,
	)

	top := container.NewVBox(
		widget.NewLabel("Review duplicates and choose actions"),
		summaryLabel,
		countLabel,
		toolbarScroll,
	)

	out := container.NewBorder(
		container.NewPadded(top),
		container.NewPadded(bottomStack),
		nil,
		nil,
		resultsTable,
	)
	rebuildPage()
	return out
}
