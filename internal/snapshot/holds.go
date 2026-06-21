package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/neur0map/glazepkg/internal/model"
)

// Hold pins a package so gpk leaves it alone during upgrades.
type Hold struct {
	Source model.Source `json:"source"`
	Name   string       `json:"name"`
}

func holdsFile() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, dirName, "holds.json")
}

// LoadHolds returns the held packages, or an empty slice.
func LoadHolds() []Hold {
	data, err := os.ReadFile(holdsFile())
	if err != nil {
		return nil
	}
	var holds []Hold
	if err := json.Unmarshal(data, &holds); err != nil {
		return nil
	}
	return holds
}

// SaveHolds writes the held packages to disk.
func SaveHolds(holds []Hold) error {
	path := holdsFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(holds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// IsHeld reports whether a package is held. An empty source matches any source
// with the same name, so a held name blocks it everywhere.
func IsHeld(holds []Hold, source model.Source, name string) bool {
	for _, h := range holds {
		if h.Name == name && (h.Source == source || h.Source == "") {
			return true
		}
	}
	return false
}

// HeldNames returns the names held for a given source.
func HeldNames(holds []Hold, source model.Source) []string {
	var names []string
	for _, h := range holds {
		if h.Source == source || h.Source == "" {
			names = append(names, h.Name)
		}
	}
	return names
}
