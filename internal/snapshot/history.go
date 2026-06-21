package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// historyLimit caps how many actions are kept on disk.
const historyLimit = 200

// HistoryOp identifies what gpk did to a package.
type HistoryOp string

const (
	OpInstall   HistoryOp = "install"
	OpRemove    HistoryOp = "remove"
	OpUpgrade   HistoryOp = "upgrade"
	OpDowngrade HistoryOp = "downgrade"
)

// HistoryItem is one package change gpk performed. Items sharing a Group were
// part of the same command, which is the unit `gpk undo` reverses.
type HistoryItem struct {
	Group       int64        `json:"group"`
	Time        time.Time    `json:"time"`
	Op          HistoryOp    `json:"op"`
	Source      model.Source `json:"source"`
	Name        string       `json:"name"`
	Version     string       `json:"version,omitempty"`
	PrevVersion string       `json:"prev_version,omitempty"`
}

func historyFile() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, dirName, "history.json")
}

// LoadHistory returns recorded actions, oldest first. Missing or unreadable
// history is treated as empty.
func LoadHistory() []HistoryItem {
	data, err := os.ReadFile(historyFile())
	if err != nil {
		return nil
	}
	var items []HistoryItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil
	}
	return items
}

// SaveHistory writes the action log, trimming to the most recent historyLimit.
func SaveHistory(items []HistoryItem) error {
	if len(items) > historyLimit {
		items = items[len(items)-historyLimit:]
	}
	path := historyFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// AppendHistory adds items to the log.
func AppendHistory(items ...HistoryItem) error {
	if len(items) == 0 {
		return nil
	}
	return SaveHistory(append(LoadHistory(), items...))
}
