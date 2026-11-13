package aiworkflow

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/war-apps/nerv-kit-by-gentle-ai/atomicfile"
)

// Load reads path as YAML into v. A missing file is not an error: found
// is false and v is left untouched.
func Load(path string, v any) (found bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		return false, fmt.Errorf("parsing %s: %w", path, err)
	}
	return true, nil
}

// SaveOptions configures one Save call.
type SaveOptions struct {
	// Now names the backup file; see atomicfile.Options.Now.
	Now time.Time
	// BackupSuffix distinguishes one caller's backups from another's.
	BackupSuffix string
	// Mode is the file's permission bits; the zero value defaults to
	// 0600, since these are user configuration files that can hold
	// secrets.
	Mode fs.FileMode
}

// Save marshals v as YAML and writes it to path atomically via
// atomicfile.Save, taking a timestamped backup of any existing file
// first. It is a no-op -- no write, no backup -- when the marshaled
// bytes already match the file's current content.
func Save(path string, v any, opts SaveOptions) (written bool, backup string, err error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return false, "", err
	}

	if existing, readErr := os.ReadFile(path); readErr == nil {
		if bytes.Equal(existing, data) {
			return false, "", nil
		}
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		return false, "", readErr
	}

	backup, err = atomicfile.Save(path, data, atomicfile.Options{
		Now:          opts.Now,
		BackupSuffix: opts.BackupSuffix,
	})
	if err != nil {
		return false, backup, err
	}

	mode := opts.Mode
	if mode == 0 {
		mode = 0o600
	}
	if err := os.Chmod(path, mode); err != nil {
		return true, backup, err
	}

	return true, backup, nil
}
