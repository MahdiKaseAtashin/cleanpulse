package devcleanup

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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
