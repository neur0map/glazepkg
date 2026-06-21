package snapshot

import (
	"os"
	"path/filepath"
)

const dirName = "glazepkg"

// Dir returns the snapshot storage directory (~/.local/share/glazepkg/snapshots/).
func Dir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, dirName, "snapshots")
}

// EnsureDir creates the snapshot directory if it doesn't exist.
func EnsureDir() error {
	return os.MkdirAll(Dir(), 0o755)
}

// writeFileAtomic writes data to path via a temp file in the same directory
// then renames it, so an interrupted write never leaves a truncated file.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}
