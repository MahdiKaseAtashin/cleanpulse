package devcleanup

import (
	"path/filepath"
)

type staticProvider struct {
	id    string
	tasks []CleanupTask
}

func (p staticProvider) ID() string { return p.id }

func (p staticProvider) Tasks(_ Environment) []CleanupTask {
	out := make([]CleanupTask, len(p.tasks))
	copy(out, p.tasks)
	return out
}

func BuiltinProviders(env Environment) []Provider {
	home := env.HomeDir
	temp := env.TempDir

	tasks := []CleanupTask{
		pathTask("nuget-packages", "NuGet package cache", "package-manager", RiskSafe, filepath.Join(home, ".nuget", "packages")),
		pathTask("dotnet-http-cache", ".NET HTTP cache", "package-manager", RiskSafe, filepath.Join(home, ".local", "share", "NuGet", "v3-cache")),
		pathTask("npm-cache", "npm cache", "package-manager", RiskSafe, filepath.Join(home, ".npm")),
		pathTask("yarn-cache", "Yarn cache", "package-manager", RiskSafe, filepath.Join(home, ".cache", "yarn")),
		pathTask("pnpm-store", "pnpm store", "package-manager", RiskSafe, filepath.Join(home, ".pnpm-store")),
		pathTask("pip-cache", "pip cache", "package-manager", RiskSafe, filepath.Join(home, ".cache", "pip")),
		pathTask("cargo-registry", "Cargo registry cache", "package-manager", RiskSafe, filepath.Join(home, ".cargo", "registry")),
		pathTask("cargo-git", "Cargo git cache", "package-manager", RiskSafe, filepath.Join(home, ".cargo", "git")),
		pathTask("gradle-cache", "Gradle cache", "package-manager", RiskSafe, filepath.Join(home, ".gradle", "caches")),
		pathTask("maven-cache", "Maven local repo", "package-manager", RiskSafe, filepath.Join(home, ".m2", "repository")),
		pathTask("dev-temp", "User temporary files", "os-temp", RiskSafe, temp),
		pathTask("vscode-cache", "VS Code cache", "ide", RiskModerate, filepath.Join(home, ".config", "Code", "Cache")),
		pathTask("vscode-workspace-storage", "VS Code workspace storage", "ide", RiskModerate, filepath.Join(home, ".config", "Code", "User", "workspaceStorage")),
		pathTask("jetbrains-cache", "JetBrains caches", "ide", RiskModerate, filepath.Join(home, ".cache", "JetBrains")),
		pathTask("docker-desktop-cache", "Docker desktop cache", "container", RiskModerate, filepath.Join(home, ".docker", "buildx")),
		pathTask("browser-cache-chrome", "Chrome cache", "browser", RiskModerate, filepath.Join(home, ".cache", "google-chrome")),
		pathTask("crash-dumps", "Crash dumps", "logs", RiskModerate, filepath.Join(home, ".local", "share", "CrashDumps")),
		commandTask("dotnet-locals", ".NET CLI cache cleanup", "package-manager", RiskSafe, "dotnet", "nuget", "locals", "all", "--clear"),
		commandTask("npm-clean-force", "npm force clean", "package-manager", RiskModerate, "npm", "cache", "clean", "--force"),
		commandTask("docker-prune", "Docker prune (images/volumes)", "container", RiskAggressive, "docker", "system", "prune", "-a", "--volumes", "-f"),
	}

	if env.OS == "windows" {
		tasks = append(tasks,
			pathTask("vscode-cache-win", "VS Code cache (Windows)", "ide", RiskModerate, filepath.Join(home, "AppData", "Roaming", "Code", "Cache")),
			pathTask("vscode-workspace-win", "VS Code workspace storage (Windows)", "ide", RiskModerate, filepath.Join(home, "AppData", "Roaming", "Code", "User", "workspaceStorage")),
			pathTask("visual-studio-cache-win", "Visual Studio component cache", "ide", RiskModerate, filepath.Join(home, "AppData", "Local", "Microsoft", "VisualStudio")),
			pathTask("windows-temp", "Windows temp", "os-temp", RiskSafe, filepath.Join(home, "AppData", "Local", "Temp")),
			pathTask("browser-cache-edge-win", "Edge cache", "browser", RiskModerate, filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data", "Default", "Cache")),
		)
	}

	if env.OS == "darwin" {
		tasks = append(tasks,
			pathTask("xcode-derived-data", "Xcode derived data", "build-artifact", RiskModerate, filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")),
			pathTask("vscode-cache-macos", "VS Code cache (macOS)", "ide", RiskModerate, filepath.Join(home, "Library", "Application Support", "Code", "Cache")),
			pathTask("jetbrains-cache-macos", "JetBrains caches (macOS)", "ide", RiskModerate, filepath.Join(home, "Library", "Caches", "JetBrains")),
		)
	}

	if env.OS == "linux" {
		tasks = append(tasks,
			pathTask("thumbnails-linux", "Desktop thumbnails", "os-temp", RiskSafe, filepath.Join(home, ".cache", "thumbnails")),
		)
	}

	return []Provider{staticProvider{id: "builtin", tasks: tasks}}
}

func pathTask(id, name, category string, risk RiskLevel, path string) CleanupTask {
	return CleanupTask{
		ID:          id,
		Kind:        TaskKindPath,
		Name:        name,
		Category:    category,
		Description: "cleanup path contents",
		Risk:        risk,
		PathTask: &PathTask{
			Path:            path,
			RemoveDirectory: false,
		},
	}
}

func commandTask(id, name, category string, risk RiskLevel, executable string, args ...string) CleanupTask {
	return CleanupTask{
		ID:          id,
		Kind:        TaskKindCommand,
		Name:        name,
		Category:    category,
		Description: "execute cleanup command",
		Risk:        risk,
		CommandTask: &CommandTask{
			Executable: executable,
			Args:       args,
		},
	}
}
