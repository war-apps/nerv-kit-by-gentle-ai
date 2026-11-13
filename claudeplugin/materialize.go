// Package claudeplugin materializes a plugin tree onto disk, registers it
// in Claude Code as a directory marketplace, drives the "claude plugin"
// CLI through a runner.Runner, and edits Claude Code's own settings.json
// and installed_plugins.json.
package claudeplugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	filePerm = 0o644
	dirPerm  = 0o755

	pluginManifestPath     = ".claude-plugin/plugin.json"
	marketplaceManifestRel = ".claude-plugin/marketplace.json"
)

// Marketplace identifies the directory marketplace Materialize registers
// the plugin tree under: its name (also used as the core plugin's own
// name) and its owner's display name.
type Marketplace struct {
	Name      string
	OwnerName string
}

// Result reports what Materialize did, as paths relative to dest.
type Result struct {
	// Written lists files that were created or whose content changed.
	Written []string
	// Unchanged lists files whose on-disk bytes already matched the source.
	Unchanged []string
	// Removed lists stale files under dest/plugin or dest/plugins that are
	// not present in the sources anymore.
	Removed []string
}

type marketplaceManifest struct {
	Name    string              `json:"name"`
	Owner   marketplaceOwner    `json:"owner"`
	Plugins []marketplacePlugin `json:"plugins"`
}

type marketplaceOwner struct {
	Name string `json:"name"`
}

type marketplacePlugin struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

// Materialize writes the whole tree from src under dest/plugin, byte-exact,
// and writes dest/.claude-plugin/marketplace.json registering it as a
// directory marketplace named m.Name. It is MaterializeBundle with no
// extensions.
func Materialize(src fs.FS, dest string, m Marketplace) (Result, error) {
	return MaterializeBundle(src, nil, dest, m)
}

// MaterializeBundle writes the core plugin tree from core under
// dest/plugin and every top-level plugin folder of extensions (one folder
// per plugin, each holding its own .claude-plugin/plugin.json whose name
// matches the folder) under dest/plugins/<name>, byte-exact, then writes
// dest/.claude-plugin/marketplace.json listing the core plugin first and
// the extensions in name order. It is idempotent: a file whose bytes
// already match the source is left untouched, and files under
// dest/plugin or dest/plugins that are no longer present in a source are
// removed. It never writes or removes anything outside dest/plugin,
// dest/plugins and dest/.claude-plugin. A nil extensions writes none.
func MaterializeBundle(core, extensions fs.FS, dest string, m Marketplace) (Result, error) {
	var result Result

	if err := ensureDestIsUsable(dest); err != nil {
		return result, err
	}

	extNames, err := extensionNames(extensions)
	if err != nil {
		return result, err
	}

	knownFiles := make(map[string]struct{})
	destPluginRoot := filepath.Join(dest, "plugin")
	if err := writeTree(core, destPluginRoot, "plugin/", knownFiles, &result); err != nil {
		return result, err
	}
	destExtRoot := filepath.Join(dest, "plugins")
	for _, name := range extNames {
		sub, err := fs.Sub(extensions, name)
		if err != nil {
			return result, err
		}
		root := filepath.Join(destExtRoot, name)
		if err := writeTree(sub, root, "plugins/"+name+"/", knownFiles, &result); err != nil {
			return result, err
		}
	}

	for _, root := range []string{destPluginRoot, destExtRoot} {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		removed, err := removeStaleFiles(root, knownFiles)
		if err != nil {
			return result, err
		}
		result.Removed = append(result.Removed, removed...)
	}

	description, err := readPluginDescription(core)
	if err != nil {
		return result, err
	}
	plugins := []marketplacePlugin{{Name: m.Name, Source: "./plugin", Description: description}}
	for _, name := range extNames {
		sub, err := fs.Sub(extensions, name)
		if err != nil {
			return result, err
		}
		desc, err := readPluginDescription(sub)
		if err != nil {
			return result, err
		}
		plugins = append(plugins, marketplacePlugin{Name: name, Source: "./plugins/" + name, Description: desc})
	}

	if err := writeMarketplaceManifest(dest, m, plugins, &result); err != nil {
		return result, err
	}

	sort.Strings(result.Written)
	sort.Strings(result.Unchanged)
	sort.Strings(result.Removed)

	return result, nil
}

// extensionNames lists the top-level folders of extensions that hold a
// .claude-plugin/plugin.json, in name order, refusing one whose manifest
// name does not match its folder. A nil extensions reports none.
func extensionNames(extensions fs.FS) ([]string, error) {
	if extensions == nil {
		return nil, nil
	}
	entries, err := fs.ReadDir(extensions, ".")
	if err != nil {
		return nil, fmt.Errorf("listing extensions: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := e.Name() + "/" + pluginManifestPath
		data, err := fs.ReadFile(extensions, manifestPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", manifestPath, err)
		}
		var manifest struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", manifestPath, err)
		}
		if manifest.Name != e.Name() {
			return nil, fmt.Errorf("%s: name %q must match its folder %q", manifestPath, manifest.Name, e.Name())
		}
		names = append(names, manifest.Name)
	}
	sort.Strings(names)
	return names, nil
}

// writeTree copies every file of src under destRoot, recording each one in
// knownFiles and result as relPrefix + its path.
func writeTree(src fs.FS, destRoot, relPrefix string, knownFiles map[string]struct{}, result *Result) error {
	if err := os.MkdirAll(destRoot, dirPerm); err != nil {
		return fmt.Errorf("creating %s: %w", destRoot, err)
	}

	return fs.WalkDir(src, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walking source at %q: %w", path, walkErr)
		}
		if path == "." {
			return nil
		}
		if err := checkSafeRelPath(path); err != nil {
			return err
		}

		destPath := filepath.Join(destRoot, filepath.FromSlash(path))
		if err := checkWithinRoot(destRoot, destPath); err != nil {
			return err
		}

		if d.IsDir() {
			if err := os.MkdirAll(destPath, dirPerm); err != nil {
				return fmt.Errorf("creating directory %s: %w", destPath, err)
			}
			return nil
		}

		data, err := fs.ReadFile(src, path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		relDest := relPrefix + path
		knownFiles[relDest] = struct{}{}

		changed, err := writeIfChanged(destPath, data)
		if err != nil {
			return err
		}
		if changed {
			result.Written = append(result.Written, relDest)
		} else {
			result.Unchanged = append(result.Unchanged, relDest)
		}
		return nil
	})
}

func ensureDestIsUsable(dest string) error {
	info, err := os.Stat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking dest %s: %w", dest, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("dest %s exists and is not a directory", dest)
	}
	return nil
}

// checkSafeRelPath refuses any fs entry whose path escapes the source root,
// either through a ".." segment or an absolute path.
func checkSafeRelPath(path string) error {
	if filepath.IsAbs(path) || strings.HasPrefix(path, "/") {
		return fmt.Errorf("refusing absolute path from source filesystem: %q", path)
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == ".." {
			return fmt.Errorf("refusing path traversal from source filesystem: %q", path)
		}
	}
	return nil
}

// checkWithinRoot is a defense-in-depth guard ensuring the resolved
// destination path still lives under root.
func checkWithinRoot(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return fmt.Errorf("resolving %s relative to %s: %w", target, root, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to write outside %s: %s", root, target)
	}
	return nil
}

// writeIfChanged writes data to destPath only when the existing content
// differs, creating parent directories as needed. It reports whether it
// wrote a new file or new content.
func writeIfChanged(destPath string, data []byte) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(destPath), dirPerm); err != nil {
		return false, fmt.Errorf("creating directory for %s: %w", destPath, err)
	}

	existing, err := os.ReadFile(destPath)
	if err == nil && bytes.Equal(existing, data) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("reading %s: %w", destPath, err)
	}

	if err := os.WriteFile(destPath, data, filePerm); err != nil {
		return false, fmt.Errorf("writing %s: %w", destPath, err)
	}
	return true, nil
}

// removeStaleFiles deletes files under root that are not present in
// knownFiles (keyed by their path relative to root's parent, such as
// "plugin/<relative path>"), then removes any directories left empty by
// that cleanup.
func removeStaleFiles(root string, knownFiles map[string]struct{}) ([]string, error) {
	var removed []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root || d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(filepath.Dir(root), path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)

		if _, ok := knownFiles[relSlash]; ok {
			return nil
		}

		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing stale file %s: %w", path, err)
		}
		removed = append(removed, relSlash)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := removeEmptyDirs(root); err != nil {
		return nil, err
	}

	return removed, nil
}

// removeEmptyDirs prunes directories under root left empty after stale-file
// removal, without ever removing root itself.
func removeEmptyDirs(root string) error {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != root {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Remove deepest directories first so parents become empty in turn.
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("reading directory %s: %w", dir, err)
		}
		if len(entries) == 0 {
			if err := os.Remove(dir); err != nil {
				return fmt.Errorf("removing empty directory %s: %w", dir, err)
			}
		}
	}
	return nil
}

func readPluginDescription(src fs.FS) (string, error) {
	data, err := fs.ReadFile(src, pluginManifestPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", pluginManifestPath, err)
	}
	var manifest struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("parsing %s: %w", pluginManifestPath, err)
	}
	return manifest.Description, nil
}

func writeMarketplaceManifest(dest string, m Marketplace, plugins []marketplacePlugin, result *Result) error {
	manifest := marketplaceManifest{
		Name:    m.Name,
		Owner:   marketplaceOwner{Name: m.OwnerName},
		Plugins: plugins,
	}

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding marketplace.json: %w", err)
	}
	encoded = append(encoded, '\n')

	claudePluginDir := filepath.Join(dest, ".claude-plugin")
	if err := os.MkdirAll(claudePluginDir, dirPerm); err != nil {
		return fmt.Errorf("creating %s: %w", claudePluginDir, err)
	}

	destPath := filepath.Join(claudePluginDir, "marketplace.json")
	changed, err := writeIfChanged(destPath, encoded)
	if err != nil {
		return err
	}
	if changed {
		result.Written = append(result.Written, marketplaceManifestRel)
	} else {
		result.Unchanged = append(result.Unchanged, marketplaceManifestRel)
	}
	return nil
}
