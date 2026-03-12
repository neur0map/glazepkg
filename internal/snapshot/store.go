package snapshot

import (
	"os"
	"path/filepath"
)

const dirName = "glazepkg"

// Dir returns the snapshot storage directory (~/.local/share/pkgtrack/snapshots/).
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
