package claudeplugin_test

import (
	"testing"
	"testing/fstest"

	"github.com/war-apps/nerv-kit-by-gentle-ai/claudeplugin"
)

func TestPluginVersion_ReadsVersionFromManifest(t *testing.T) {
	mapFS := fstest.MapFS{
		".claude-plugin/plugin.json": &fstest.MapFile{
			Data: []byte(`{"name":"kit","version":"1.2.3"}`),
		},
	}

	got, err := claudeplugin.PluginVersion(mapFS)
	if err != nil {
		t.Fatalf("PluginVersion returned error: %v", err)
	}
	if got != "1.2.3" {
		t.Errorf("PluginVersion() = %q, want %q", got, "1.2.3")
	}
}

func TestPluginVersion_MissingManifestReturnsError(t *testing.T) {
	mapFS := fstest.MapFS{}

	if _, err := claudeplugin.PluginVersion(mapFS); err == nil {
		t.Fatal("expected an error for a missing plugin.json, got nil")
	}
}

func TestPluginVersion_EmptyVersionReturnsError(t *testing.T) {
	mapFS := fstest.MapFS{
		".claude-plugin/plugin.json": &fstest.MapFile{
			Data: []byte(`{"name":"kit","version":""}`),
		},
	}

	if _, err := claudeplugin.PluginVersion(mapFS); err == nil {
		t.Fatal("expected an error for an empty version, got nil")
	}
}
