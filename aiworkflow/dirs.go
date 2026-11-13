// Package aiworkflow resolves the user-scope and project-scope
// configuration locations the NERV ecosystem CLIs share (a tool's own
// "<dir>/ai-workflow/<tool>.yaml" and a repository's own
// "<root>/.ai-workflow/<tool>.yaml"), plus a generic YAML load/save of a
// Go struct over atomicfile.
package aiworkflow

import (
	"os"
	"path/filepath"
)

// UserConfigDir is the user-scope configuration directory for home on
// goos: $XDG_CONFIG_HOME (or "<home>/.config" when unset) on Linux and
// other POSIX systems, %AppData% (or "<home>/AppData/Roaming" when
// unset) on Windows, and "<home>/Library/Application Support" on macOS.
func UserConfigDir(home, goos string) string {
	switch goos {
	case "windows":
		if dir := os.Getenv("AppData"); dir != "" {
			return dir
		}
		return filepath.Join(home, "AppData", "Roaming")
	case "darwin", "ios":
		return filepath.Join(home, "Library", "Application Support")
	case "plan9":
		return filepath.Join(home, "lib")
	default:
		if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
			return dir
		}
		return filepath.Join(home, ".config")
	}
}

// UserConfigPath is tool's user-scope configuration file:
// "<UserConfigDir(home, goos)>/ai-workflow/<tool>.yaml".
func UserConfigPath(home, goos, tool string) string {
	return filepath.Join(UserConfigDir(home, goos), "ai-workflow", tool+".yaml")
}

// ProjectDir is the repository's committed ai-workflow folder:
// "<root>/.ai-workflow".
func ProjectDir(root string) string {
	return filepath.Join(root, ".ai-workflow")
}

// ProjectConfigPath is tool's project-scope configuration file:
// "<root>/.ai-workflow/<tool>.yaml".
func ProjectConfigPath(root, tool string) string {
	return filepath.Join(ProjectDir(root), tool+".yaml")
}
