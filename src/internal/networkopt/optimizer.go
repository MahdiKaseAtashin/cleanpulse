package networkopt

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type RiskLevel string

const (
	RiskSafe       RiskLevel = "safe"
	RiskModerate   RiskLevel = "moderate"
	RiskAggressive RiskLevel = "aggressive"
)

type Action struct {
	ID          string
	Name        string
	Description string
	Risk        RiskLevel
	Commands    [][]string
}

type Result struct {
	ActionID string
	Name     string
	Risk     RiskLevel
	Executed bool
	Success  bool
	Output   string
	Error    string
	Duration time.Duration
}

type Report struct {
	StartedAt  time.Time
	FinishedAt time.Time
	Results    []Result
}

func (r Report) SuccessCount() int {
	count := 0
	for _, item := range r.Results {
		if item.Executed && item.Success {
			count++
		}
	}
	return count
}

func (r Report) FailedCount() int {
	count := 0
	for _, item := range r.Results {
		if item.Executed && !item.Success {
			count++
		}
	}
	return count
}

func (r Report) SkippedCount() int {
	count := 0
	for _, item := range r.Results {
		if !item.Executed {
			count++
		}
	}
	return count
}

func AvailableActions() []Action {
	switch runtime.GOOS {
	case "windows":
		return []Action{
			{
				ID:          "flush-dns",
				Name:        "Flush DNS cache",
				Description: "Clears stale DNS entries to fix name resolution issues.",
				Risk:        RiskSafe,
				Commands:    [][]string{{"ipconfig", "/flushdns"}},
			},
			{
				ID:          "reset-winsock",
				Name:        "Reset Winsock catalog",
				Description: "Repairs socket stack issues. Restart may be required.",
				Risk:        RiskModerate,
				Commands:    [][]string{{"netsh", "winsock", "reset"}},
			},
			{
				ID:          "renew-ip",
				Name:        "Renew IP lease",
				Description: "Releases and renews network lease. Temporary disconnect expected.",
				Risk:        RiskAggressive,
				Commands:    [][]string{{"ipconfig", "/release"}, {"ipconfig", "/renew"}},
			},
		}
	case "darwin":
		return []Action{
			{
				ID:          "flush-dns",
				Name:        "Flush DNS cache",
				Description: "Clears stale DNS entries to fix name resolution issues.",
				Risk:        RiskSafe,
				Commands: [][]string{
					{"dscacheutil", "-flushcache"},
					{"killall", "-HUP", "mDNSResponder"},
				},
			},
		}
	default:
		return []Action{
			{
				ID:          "flush-dns",
				Name:        "Flush DNS cache",
				Description: "Clears stale DNS entries to fix name resolution issues.",
				Risk:        RiskSafe,
				Commands:    [][]string{{"resolvectl", "flush-caches"}},
			},
		}
	}
}

func Run(ctx context.Context, selectedIDs []string, dryRun bool) Report {
	choices := make(map[string]struct{}, len(selectedIDs))
	for _, id := range selectedIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			choices[id] = struct{}{}
		}
	}
	report := Report{StartedAt: time.Now()}
	actions := AvailableActions()
	for _, action := range actions {
		if len(choices) > 0 {
			if _, ok := choices[action.ID]; !ok {
				continue
			}
		}
		result := Result{
			ActionID: action.ID,
			Name:     action.Name,
			Risk:     action.Risk,
		}
		started := time.Now()
		if dryRun {
			result.Executed = false
			result.Success = true
			result.Output = "Dry run: no command executed."
			result.Duration = time.Since(started)
			report.Results = append(report.Results, result)
			continue
		}
		result.Executed = true
		if err := executeAction(ctx, action, &result); err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
		}
		result.Duration = time.Since(started)
		report.Results = append(report.Results, result)
	}
	report.FinishedAt = time.Now()
	return report
}

func executeAction(ctx context.Context, action Action, result *Result) error {
	if len(action.Commands) == 0 {
		return fmt.Errorf("no commands configured for action")
	}
	var output strings.Builder
	for _, commandParts := range action.Commands {
		if len(commandParts) == 0 {
			continue
		}
		cmdName := strings.TrimSpace(commandParts[0])
		args := commandParts[1:]
		if _, err := exec.LookPath(cmdName); err != nil {
			return fmt.Errorf("%s is not available on this machine", cmdName)
		}
		cmd := exec.CommandContext(ctx, cmdName, args...)
		raw, err := cmd.CombinedOutput()
		if output.Len() > 0 {
			output.WriteString("\n")
		}
		output.WriteString(strings.TrimSpace(string(raw)))
		if err != nil {
			result.Output = strings.TrimSpace(output.String())
			return fmt.Errorf("%s failed: %w", cmdName, err)
		}
	}
	result.Output = strings.TrimSpace(output.String())
	return nil
}
