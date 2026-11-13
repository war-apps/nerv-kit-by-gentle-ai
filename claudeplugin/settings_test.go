package claudeplugin_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/war-apps/nerv-kit-by-gentle-ai/claudeplugin"
)

func TestLoadSettings_MissingFileReportsNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	_, err := claudeplugin.LoadSettings(path)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("LoadSettings() error = %v, want fs.ErrNotExist", err)
	}
}

func TestLoadSettings_MalformedJSONErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := claudeplugin.LoadSettings(path)
	if err == nil {
		t.Fatal("LoadSettings() error = nil, want an error for malformed JSON")
	}
}

func TestSettings_RegisterMarketplace_ChangedOnFirstCallOnly(t *testing.T) {
	s := claudeplugin.NewSettings()

	if changed := s.RegisterMarketplace("kit", `C:\home\.kit\marketplace`); !changed {
		t.Error("first RegisterMarketplace() changed = false, want true")
	}
	if changed := s.RegisterMarketplace("kit", `C:\home\.kit\marketplace`); changed {
		t.Error("second RegisterMarketplace() with same path changed = true, want false")
	}
	if changed := s.RegisterMarketplace("kit", `C:\home\.kit\marketplace2`); !changed {
		t.Error("RegisterMarketplace() with a different path changed = false, want true")
	}
}

func TestSettings_RegisterMarketplace_WritesExpectedShape(t *testing.T) {
	s := claudeplugin.NewSettings()
	s.RegisterMarketplace("kit", `C:\home\.kit\marketplace`)

	encoded, err := s.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v", err)
	}

	var decoded struct {
		ExtraKnownMarketplaces struct {
			Kit struct {
				Source struct {
					Source string `json:"source"`
					Path   string `json:"path"`
				} `json:"source"`
			} `json:"kit"`
		} `json:"extraKnownMarketplaces"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Bytes() did not produce valid JSON: %v", err)
	}
	if decoded.ExtraKnownMarketplaces.Kit.Source.Source != "directory" {
		t.Errorf("source.source = %q, want %q", decoded.ExtraKnownMarketplaces.Kit.Source.Source, "directory")
	}
	if decoded.ExtraKnownMarketplaces.Kit.Source.Path != `C:\home\.kit\marketplace` {
		t.Errorf("source.path = %q, want %q", decoded.ExtraKnownMarketplaces.Kit.Source.Path, `C:\home\.kit\marketplace`)
	}
}

func TestSettings_EnablePlugin_ChangedOnFirstCallOnly(t *testing.T) {
	s := claudeplugin.NewSettings()

	if changed := s.EnablePlugin("kit@kit"); !changed {
		t.Error("first EnablePlugin() changed = false, want true")
	}
	if changed := s.EnablePlugin("kit@kit"); changed {
		t.Error("second EnablePlugin() changed = true, want false")
	}

	encoded, err := s.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v", err)
	}
	var decoded struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Bytes() did not produce valid JSON: %v", err)
	}
	if !decoded.EnabledPlugins["kit@kit"] {
		t.Error(`enabledPlugins["kit@kit"] not true`)
	}
}

func TestSettings_Unregister_RemovesBothKeysAndReportsChanged(t *testing.T) {
	s := claudeplugin.NewSettings()
	s.RegisterMarketplace("kit", `C:\home\.kit\marketplace`)
	s.EnablePlugin("kit@kit")

	if changed := s.Unregister("kit", "kit@kit"); !changed {
		t.Error("Unregister() changed = false, want true")
	}
	if changed := s.Unregister("kit", "kit@kit"); changed {
		t.Error("second Unregister() changed = true, want false (already absent)")
	}

	encoded, err := s.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v", err)
	}
	if strings.Contains(string(encoded), "kit@kit") || strings.Contains(string(encoded), "marketplace") {
		t.Errorf("Bytes() still mentions removed entries: %s", encoded)
	}
}

func TestSettings_PreservesUnrelatedKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	original := `{
  "dimTools": { "enabled": true },
  "theme": "dark"
}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := claudeplugin.LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}
	s.RegisterMarketplace("kit", `C:\home\.kit\marketplace`)
	s.EnablePlugin("kit@kit")

	encoded, err := s.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Bytes() did not produce valid JSON: %v", err)
	}
	if _, ok := decoded["dimTools"]; !ok {
		t.Error("dimTools key was dropped")
	}
	if decoded["theme"] != "dark" {
		t.Errorf("theme = %v, want dark", decoded["theme"])
	}
}

func TestSaveSettings_NoopWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	s := claudeplugin.NewSettings()
	previous, err := s.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, previous, 0o644); err != nil {
		t.Fatal(err)
	}

	written, backup, err := claudeplugin.SaveSettings(path, s, previous, time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC), "bak-kit-")
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	if written {
		t.Error("written = true, want false (unchanged)")
	}
	if backup != "" {
		t.Errorf("backup = %q, want empty", backup)
	}
}

func TestSaveSettings_BacksUpAndWritesAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	original := []byte(`{"theme":"dark"}`)
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := claudeplugin.LoadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	s.EnablePlugin("kit@kit")

	now := time.Date(2026, 9, 29, 15, 4, 5, 0, time.UTC)
	written, backup, err := claudeplugin.SaveSettings(path, s, original, now, "bak-kit-")
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	if !written {
		t.Fatal("written = false, want true")
	}
	wantBackup := path + ".bak-kit-20260929-150405"
	if backup != wantBackup {
		t.Errorf("backup = %q, want %q", backup, wantBackup)
	}
	backupBytes, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(backupBytes) != string(original) {
		t.Errorf("backup content = %s, want %s", backupBytes, original)
	}

	finalBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading final settings: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(finalBytes, &decoded); err != nil {
		t.Fatalf("final settings.json did not parse: %v", err)
	}

	// No leftover temp file.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestSaveSettings_NoBackupWhenFileDidNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	s := claudeplugin.NewSettings()
	s.EnablePlugin("kit@kit")

	written, backup, err := claudeplugin.SaveSettings(path, s, nil, time.Now(), "bak-kit-")
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	if !written {
		t.Fatal("written = false, want true")
	}
	if backup != "" {
		t.Errorf("backup = %q, want empty (file did not exist)", backup)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("settings.json not created: %v", err)
	}
}
