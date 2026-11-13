package aiworkflow_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/war-apps/nerv-kit-by-gentle-ai/aiworkflow"
)

type testConfig struct {
	Model string `yaml:"model"`
	Count int    `yaml:"count"`
}

func TestLoad_MissingFileReportsNotFoundWithoutError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	var cfg testConfig

	found, err := aiworkflow.Load(path, &cfg)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if found {
		t.Error("found = true, want false for a missing file")
	}
}

func TestLoad_ParsesYAMLIntoV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("model: sonnet\ncount: 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var cfg testConfig

	found, err := aiworkflow.Load(path, &cfg)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if cfg.Model != "sonnet" || cfg.Count != 3 {
		t.Errorf("cfg = %+v, want {sonnet 3}", cfg)
	}
}

func TestLoad_MalformedYAMLErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("not: [valid: yaml"), 0o600); err != nil {
		t.Fatal(err)
	}
	var cfg testConfig

	if _, err := aiworkflow.Load(path, &cfg); err == nil {
		t.Fatal("Load() error = nil, want an error for malformed YAML")
	}
}

func TestSave_WritesYAMLAndDefaultsToMode0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := testConfig{Model: "sonnet", Count: 2}

	written, backup, err := aiworkflow.Save(path, cfg, aiworkflow.SaveOptions{Now: time.Now()})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if !written {
		t.Fatal("written = false, want true")
	}
	if backup != "" {
		t.Errorf("backup = %q, want empty (no prior file)", backup)
	}

	var got testConfig
	found, err := aiworkflow.Load(path, &got)
	if err != nil || !found {
		t.Fatalf("Load() = %v, %v, want it readable back", found, err)
	}
	if got != cfg {
		t.Errorf("round-tripped = %+v, want %+v", got, cfg)
	}

	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestSave_NoopWhenContentUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := testConfig{Model: "sonnet"}

	if _, _, err := aiworkflow.Save(path, cfg, aiworkflow.SaveOptions{Now: time.Now()}); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	written, backup, err := aiworkflow.Save(path, cfg, aiworkflow.SaveOptions{Now: time.Now(), BackupSuffix: "bak-test-"})
	if err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	if written {
		t.Error("written = true, want false (content unchanged)")
	}
	if backup != "" {
		t.Errorf("backup = %q, want empty", backup)
	}
}

func TestSave_BacksUpExistingFileOnChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	now := time.Date(2026, 9, 29, 12, 34, 56, 0, time.UTC)

	if _, _, err := aiworkflow.Save(path, testConfig{Model: "sonnet"}, aiworkflow.SaveOptions{Now: now, BackupSuffix: "bak-test-"}); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	written, backup, err := aiworkflow.Save(path, testConfig{Model: "opus"}, aiworkflow.SaveOptions{Now: now, BackupSuffix: "bak-test-"})
	if err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	if !written {
		t.Fatal("written = false, want true (content changed)")
	}
	wantBackup := path + ".bak-test-20260929-123456"
	if backup != wantBackup {
		t.Errorf("backup = %q, want %q", backup, wantBackup)
	}
	backupContent, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(backupContent), "sonnet") {
		t.Errorf("backup content = %q, want the previous (sonnet) content", backupContent)
	}
}

func TestSave_CustomModeOverridesTheDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningfully enforced by the Windows filesystem")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if _, _, err := aiworkflow.Save(path, testConfig{Model: "sonnet"}, aiworkflow.SaveOptions{Now: time.Now(), Mode: 0o644}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v, want the custom 0644", info.Mode().Perm())
	}
}
