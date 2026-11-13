package aiworkflow_test

import (
	"path/filepath"
	"testing"

	"github.com/war-apps/nerv-kit-by-gentle-ai/aiworkflow"
)

func TestUserConfigDir_Windows(t *testing.T) {
	t.Setenv("AppData", "")
	home := `C:\Users\walter`
	got := aiworkflow.UserConfigDir(home, "windows")
	want := filepath.Join(home, "AppData", "Roaming")
	if got != want {
		t.Errorf("UserConfigDir(windows) = %q, want %q", got, want)
	}
}

func TestUserConfigDir_WindowsHonorsAppDataEnv(t *testing.T) {
	t.Setenv("AppData", `D:\custom\AppData\Roaming`)
	got := aiworkflow.UserConfigDir(`C:\Users\walter`, "windows")
	if got != `D:\custom\AppData\Roaming` {
		t.Errorf("UserConfigDir(windows) = %q, want the AppData env override", got)
	}
}

func TestUserConfigDir_Darwin(t *testing.T) {
	home := "/Users/walter"
	got := aiworkflow.UserConfigDir(home, "darwin")
	want := filepath.Join(home, "Library", "Application Support")
	if got != want {
		t.Errorf("UserConfigDir(darwin) = %q, want %q", got, want)
	}
}

func TestUserConfigDir_LinuxDefaultsToDotConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home := "/home/walter"
	got := aiworkflow.UserConfigDir(home, "linux")
	want := filepath.Join(home, ".config")
	if got != want {
		t.Errorf("UserConfigDir(linux) = %q, want %q", got, want)
	}
}

func TestUserConfigDir_LinuxHonorsXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
	got := aiworkflow.UserConfigDir("/home/walter", "linux")
	if got != "/custom/xdg" {
		t.Errorf("UserConfigDir(linux) = %q, want the XDG_CONFIG_HOME override", got)
	}
}

func TestUserConfigPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home := "/home/walter"
	got := aiworkflow.UserConfigPath(home, "linux", "nerv")
	want := filepath.Join(home, ".config", "ai-workflow", "nerv.yaml")
	if got != want {
		t.Errorf("UserConfigPath() = %q, want %q", got, want)
	}
}

func TestProjectDir(t *testing.T) {
	got := aiworkflow.ProjectDir(`C:\repo`)
	want := filepath.Join(`C:\repo`, ".ai-workflow")
	if got != want {
		t.Errorf("ProjectDir() = %q, want %q", got, want)
	}
}

func TestProjectConfigPath(t *testing.T) {
	got := aiworkflow.ProjectConfigPath(`C:\repo`, "nerv-task")
	want := filepath.Join(`C:\repo`, ".ai-workflow", "nerv-task.yaml")
	if got != want {
		t.Errorf("ProjectConfigPath() = %q, want %q", got, want)
	}
}
