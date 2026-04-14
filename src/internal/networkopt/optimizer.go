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

type Profile struct {
	ID          string
	Name        string
	Description string
	ActionIDs   []string
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
			{
				ID:          "disable-delivery-optimization-p2p",
				Name:        "Disable Delivery Optimization P2P",
				Description: "Stops peer-to-peer upload usage by Windows update delivery optimization.",
				Risk:        RiskModerate,
				Commands: [][]string{
					{"reg", "add", "HKLM\\SOFTWARE\\Policies\\Microsoft\\Windows\\DeliveryOptimization", "/v", "DODownloadMode", "/t", "REG_DWORD", "/d", "0", "/f"},
				},
			},
			{
				ID:          "disable-background-app-network",
				Name:        "Disable background app network access",
				Description: "Limits background network use by apps.",
				Risk:        RiskModerate,
				Commands: [][]string{
					{"reg", "add", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\BackgroundAccessApplications", "/v", "GlobalUserDisabled", "/t", "REG_DWORD", "/d", "1", "/f"},
				},
			},
			{
				ID:          "set-wu-notify",
				Name:        "Set Windows Update to notify",
				Description: "Avoids silent background downloads for Windows update.",
				Risk:        RiskModerate,
				Commands: [][]string{
					{"reg", "add", "HKLM\\SOFTWARE\\Policies\\Microsoft\\Windows\\WindowsUpdate\\AU", "/v", "AUOptions", "/t", "REG_DWORD", "/d", "2", "/f"},
				},
			},
			{
				ID:          "disable-diagtrack",
				Name:        "Disable telemetry service (DiagTrack)",
				Description: "Turns off telemetry background service traffic.",
				Risk:        RiskModerate,
				Commands: [][]string{
					{"sc", "stop", "DiagTrack"},
					{"sc", "config", "DiagTrack", "start=", "disabled"},
				},
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
			{
				ID:          "disable-macos-auto-downloads",
				Name:        "Disable macOS auto update downloads",
				Description: "Stops background software update downloads.",
				Risk:        RiskModerate,
				Commands: [][]string{
					{"defaults", "write", "/Library/Preferences/com.apple.SoftwareUpdate", "AutomaticDownload", "-bool", "false"},
					{"defaults", "write", "/Library/Preferences/com.apple.SoftwareUpdate", "AutomaticCheckEnabled", "-bool", "false"},
				},
			},
			{
				ID:          "disable-spotlight-online-suggestions",
				Name:        "Disable Spotlight online suggestions",
				Description: "Reduces online lookup traffic from Spotlight.",
				Risk:        RiskSafe,
				Commands: [][]string{
					{"defaults", "write", "com.apple.lookup.shared", "LookupSuggestionsDisabled", "-bool", "true"},
				},
			},
		}
	default:
		actions := []Action{
			{
				ID:          "flush-dns",
				Name:        "Flush DNS cache",
				Description: "Clears stale DNS entries to fix name resolution issues.",
				Risk:        RiskSafe,
				Commands:    [][]string{{"resolvectl", "flush-caches"}},
			},
		}
		distro := linuxDistroID()
		switch distro {
		case "ubuntu":
			actions = append(actions,
				Action{
					ID:          "disable-unattended-upgrades",
					Name:        "Disable unattended upgrades",
					Description: "Stops automatic background package downloads.",
					Risk:        RiskModerate,
					Commands:    [][]string{{"systemctl", "disable", "--now", "unattended-upgrades"}},
				},
				Action{
					ID:          "set-snap-offhours",
					Name:        "Restrict snap auto refresh to off-hours",
					Description: "Moves snap background refresh to midnight window.",
					Risk:        RiskModerate,
					Commands:    [][]string{{"snap", "set", "system", "refresh.schedule=00:00-05:00"}},
				},
				Action{
					ID:          "disable-apt-periodic",
					Name:        "Disable APT periodic checks",
					Description: "Turns off periodic APT metadata/update checks.",
					Risk:        RiskModerate,
					Commands: [][]string{
						{"sh", "-c", "printf 'APT::Periodic::Update-Package-Lists \"0\";\\nAPT::Periodic::Download-Upgradeable-Packages \"0\";\\nAPT::Periodic::AutocleanInterval \"0\";\\n' > /etc/apt/apt.conf.d/10periodic"},
					},
				},
			)
		case "fedora":
			actions = append(actions,
				Action{
					ID:          "disable-dnf-automatic",
					Name:        "Disable DNF automatic updates",
					Description: "Stops dnf-automatic timers that download updates in background.",
					Risk:        RiskModerate,
					Commands: [][]string{
						{"systemctl", "disable", "--now", "dnf-automatic.timer"},
						{"systemctl", "disable", "--now", "dnf-automatic-install.timer"},
						{"systemctl", "disable", "--now", "dnf-automatic-notifyonly.timer"},
					},
				},
				Action{
					ID:          "disable-dnf-makecache",
					Name:        "Disable DNF makecache timer",
					Description: "Stops periodic metadata cache refresh downloads.",
					Risk:        RiskModerate,
					Commands:    [][]string{{"systemctl", "disable", "--now", "dnf-makecache.timer"}},
				},
			)
		}
		return actions
	}
}

func AvailableProfiles() []Profile {
	switch runtime.GOOS {
	case "windows":
		return []Profile{
			{
				ID:          "windows-required",
				Name:        "Windows Required Profile",
				Description: "Matches required changes from windows_network_optimize.ps1.",
				ActionIDs: []string{
					"disable-delivery-optimization-p2p",
					"disable-background-app-network",
					"set-wu-notify",
					"disable-diagtrack",
					"flush-dns",
				},
			},
		}
	case "darwin":
		return []Profile{
			{
				ID:          "macos-required",
				Name:        "macOS Required Profile",
				Description: "Matches required changes from macos_network_optimize.sh.",
				ActionIDs: []string{
					"disable-macos-auto-downloads",
					"disable-spotlight-online-suggestions",
					"flush-dns",
				},
			},
		}
	default:
		if linuxDistroID() == "ubuntu" {
			return []Profile{
				{
					ID:          "ubuntu-required",
					Name:        "Ubuntu Required Profile",
					Description: "Matches required changes from ubuntu_network_optimize.sh.",
					ActionIDs: []string{
						"disable-unattended-upgrades",
						"set-snap-offhours",
						"disable-apt-periodic",
						"flush-dns",
					},
				},
			}
		}
		if linuxDistroID() == "fedora" {
			return []Profile{
				{
					ID:          "fedora-required",
					Name:        "Fedora Required Profile",
					Description: "Matches required changes from fedora_network_optimize.sh.",
					ActionIDs: []string{
						"disable-dnf-automatic",
						"disable-dnf-makecache",
						"flush-dns",
					},
				},
			}
		}
		return nil
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

func linuxDistroID() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	cmd := exec.Command("sh", "-c", "awk -F= '/^ID=/{gsub(/\"/,\"\",$2);print $2}' /etc/os-release 2>/dev/null")
	raw, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(string(raw)))
}
