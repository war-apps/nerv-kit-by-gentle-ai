package claudeplugin_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/war-apps/nerv-kit-by-gentle-ai/claudeplugin"
)

func TestInstalledPlugin_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "installed_plugins.json")

	info, found, err := claudeplugin.InstalledPlugin(path, "kit@kit")
	if err != nil {
		t.Fatalf("InstalledPlugin() error = %v", err)
	}
	if found {
		t.Errorf("found = true, want false; info = %+v", info)
	}
}

func TestInstalledPlugin_MalformedJSONErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "installed_plugins.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := claudeplugin.InstalledPlugin(path, "kit@kit")
	if err == nil {
		t.Fatal("InstalledPlugin() error = nil, want an error")
	}
}

func TestInstalledPlugin_ParsesTheRequestedEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "installed_plugins.json")
	content := `{
  "plugins": {
    "kit@kit": [
      { "installPath": "/home/user/.claude/plugins/cache/kit/kit/0.4.0", "version": "0.4.0" }
    ]
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	info, found, err := claudeplugin.InstalledPlugin(path, "kit@kit")
	if err != nil {
		t.Fatalf("InstalledPlugin() error = %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if info.Version != "0.4.0" {
		t.Errorf("Version = %q, want %q", info.Version, "0.4.0")
	}
	if info.InstallPath != "/home/user/.claude/plugins/cache/kit/kit/0.4.0" {
		t.Errorf("InstallPath = %q, want the cache path", info.InstallPath)
	}
}

func TestInstalledPlugin_MissingEntryNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "installed_plugins.json")
	if err := os.WriteFile(path, []byte(`{"plugins":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, found, err := claudeplugin.InstalledPlugin(path, "kit@kit")
	if err != nil {
		t.Fatalf("InstalledPlugin() error = %v", err)
	}
	if found {
		t.Error("found = true, want false")
	}
}

func TestCacheAgentsDir(t *testing.T) {
	got := claudeplugin.CacheAgentsDir(`C:\Users\walter`, "kit", "kit", "0.4.0")
	want := filepath.Join(`C:\Users\walter`, ".claude", "plugins", "cache", "kit", "kit", "0.4.0", "agents")
	if got != want {
		t.Errorf("CacheAgentsDir() = %q, want %q", got, want)
	}
}
