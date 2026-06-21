package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// notesFile returns the path to the user notes file.
func notesFile() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, dirName, "notes.json")
}

// LoadNotes reads user-defined descriptions from disk.
// Returns an empty map if the file doesn't exist.
func LoadNotes() map[string]string {
	data, err := os.ReadFile(notesFile())
	if err != nil {
		return make(map[string]string)
	}
	var notes map[string]string
	if err := json.Unmarshal(data, &notes); err != nil {
		return make(map[string]string)
	}
	return notes
}

// SaveNotes writes user-defined descriptions to disk.
func SaveNotes(notes map[string]string) error {
	path := notesFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0o644)
}
