package claudeplugin

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// PluginVersion reads the "version" field from .claude-plugin/plugin.json
// in the given filesystem.
func PluginVersion(fsys fs.FS) (string, error) {
	data, err := fs.ReadFile(fsys, ".claude-plugin/plugin.json")
	if err != nil {
		return "", fmt.Errorf("reading .claude-plugin/plugin.json: %w", err)
	}

	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("parsing .claude-plugin/plugin.json: %w", err)
	}
	if manifest.Version == "" {
		return "", fmt.Errorf(".claude-plugin/plugin.json: version is empty")
	}
	return manifest.Version, nil
}
