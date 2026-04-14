package devcleanup

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"strings"
	"time"
)

type ConsolePrompt struct {
	in  io.Reader
	out io.Writer
}

func NewConsolePrompt(in io.Reader, out io.Writer) *ConsolePrompt {
	return &ConsolePrompt{in: in, out: out}
}

func (p *ConsolePrompt) Confirm(message string) bool {
	if p == nil {
		return false
	}
	fmt.Fprintf(p.out, "%s [y/N]: ", message)
	reader := bufio.NewReader(p.in)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func WriteJSONReport(path string, report RunReport) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func PrintRunSummary(out io.Writer, report RunReport) {
	fmt.Fprintln(out, "=== Dev Cleanup Summary ===")
	fmt.Fprintf(out, "OS: %s\n", report.OS)
	fmt.Fprintf(out, "Dry run: %t\n", report.DryRun)
	fmt.Fprintf(out, "Max risk: %s\n", report.MaxRisk)
	fmt.Fprintf(out, "Planned tasks: %d | Attempted: %d | Skipped: %d\n", report.Planned, report.Attempted, report.Skipped)
	fmt.Fprintf(out, "Reclaimed bytes: %d\n", report.ReclaimedBytes)
	fmt.Fprintf(out, "Duration: %s\n", report.Duration)
}

func WriteMarkdownReport(path string, report RunReport) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	builder := strings.Builder{}
	builder.WriteString("# Dev Cleanup Report\n\n")
	builder.WriteString(fmt.Sprintf("- Generated: %s\n", report.GeneratedAt.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- OS: %s\n", report.OS))
	builder.WriteString(fmt.Sprintf("- Dry run: %t\n", report.DryRun))
	builder.WriteString(fmt.Sprintf("- Max risk: %s\n", report.MaxRisk))
	builder.WriteString(fmt.Sprintf("- Planned: %d | Attempted: %d | Skipped: %d\n", report.Planned, report.Attempted, report.Skipped))
	builder.WriteString(fmt.Sprintf("- Reclaimed bytes: %d\n", report.ReclaimedBytes))
	builder.WriteString(fmt.Sprintf("- Duration: %s\n\n", report.Duration))
	builder.WriteString("| ID | Name | Category | Risk | Attempted | Skipped | Deleted Items | Deleted Bytes | Error |\n")
	builder.WriteString("|---|---|---|---|---|---|---:|---:|---|\n")
	for _, result := range report.Results {
		builder.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %t | %t | %d | %d | %s |\n",
			escapeMarkdown(result.ID),
			escapeMarkdown(result.Name),
			escapeMarkdown(result.Category),
			escapeMarkdown(result.Risk),
			result.Attempted,
			result.Skipped,
			result.DeletedItems,
			result.DeletedBytes,
			escapeMarkdown(result.Error),
		))
	}
	_, err = file.WriteString(builder.String())
	return err
}

func WriteHTMLReport(path string, report RunReport) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	builder := strings.Builder{}
	builder.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>Dev Cleanup Report</title>")
	builder.WriteString("<style>body{font-family:Segoe UI,Arial,sans-serif;padding:16px;}table{border-collapse:collapse;width:100%;}th,td{border:1px solid #ddd;padding:8px;font-size:13px;}th{background:#f2f2f2;text-align:left;}tr:nth-child(even){background:#fafafa;}</style>")
	builder.WriteString("</head><body>")
	builder.WriteString("<h1>Dev Cleanup Report</h1>")
	builder.WriteString("<ul>")
	builder.WriteString(fmt.Sprintf("<li>Generated: %s</li>", html.EscapeString(report.GeneratedAt.Format(time.RFC3339))))
	builder.WriteString(fmt.Sprintf("<li>OS: %s</li>", html.EscapeString(report.OS)))
	builder.WriteString(fmt.Sprintf("<li>Dry run: %t</li>", report.DryRun))
	builder.WriteString(fmt.Sprintf("<li>Max risk: %s</li>", html.EscapeString(report.MaxRisk)))
	builder.WriteString(fmt.Sprintf("<li>Planned: %d | Attempted: %d | Skipped: %d</li>", report.Planned, report.Attempted, report.Skipped))
	builder.WriteString(fmt.Sprintf("<li>Reclaimed bytes: %d</li>", report.ReclaimedBytes))
	builder.WriteString(fmt.Sprintf("<li>Duration: %s</li>", html.EscapeString(report.Duration.String())))
	builder.WriteString("</ul>")
	builder.WriteString("<table><thead><tr><th>ID</th><th>Name</th><th>Category</th><th>Risk</th><th>Attempted</th><th>Skipped</th><th>Deleted Items</th><th>Deleted Bytes</th><th>Error</th></tr></thead><tbody>")
	for _, result := range report.Results {
		builder.WriteString("<tr>")
		builder.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(result.ID)))
		builder.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(result.Name)))
		builder.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(result.Category)))
		builder.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(result.Risk)))
		builder.WriteString(fmt.Sprintf("<td>%t</td>", result.Attempted))
		builder.WriteString(fmt.Sprintf("<td>%t</td>", result.Skipped))
		builder.WriteString(fmt.Sprintf("<td>%d</td>", result.DeletedItems))
		builder.WriteString(fmt.Sprintf("<td>%d</td>", result.DeletedBytes))
		builder.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(result.Error)))
		builder.WriteString("</tr>")
	}
	builder.WriteString("</tbody></table></body></html>")
	_, err = file.WriteString(builder.String())
	return err
}

func escapeMarkdown(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
