package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// New creates a snapshot from a list of packages.
func New(pkgs []model.Package) *model.Snapshot {
	snap := &model.Snapshot{
		Timestamp: time.Now(),
		Packages:  make(map[string]model.Package, len(pkgs)),
	}
	for _, p := range pkgs {
		snap.Packages[p.Key()] = p
	}
	return snap
}

// Save writes a snapshot to disk.
func Save(snap *model.Snapshot) (string, error) {
	if err := EnsureDir(); err != nil {
		return "", err
	}

	filename := snap.Timestamp.Format("2006-01-02T15-04-05") + ".json"
	path := filepath.Join(Dir(), filename)

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", err
	}

	return path, os.WriteFile(path, data, 0o644)
}

// Load reads a snapshot from a file.
func Load(path string) (*model.Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap model.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// Latest returns the most recent snapshot, or nil if none exist.
func Latest() (*model.Snapshot, error) {
	entries, err := os.ReadDir(Dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var jsonFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			jsonFiles = append(jsonFiles, e.Name())
		}
	}
	if len(jsonFiles) == 0 {
		return nil, nil
	}

	sort.Strings(jsonFiles)
	latest := jsonFiles[len(jsonFiles)-1]
	return Load(filepath.Join(Dir(), latest))
}

// List returns all snapshot file paths, newest first.
func List() ([]string, error) {
	entries, err := os.ReadDir(Dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			paths = append(paths, filepath.Join(Dir(), e.Name()))
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	return paths, nil
}

// FormatDiff returns a human-readable diff summary.
func FormatDiff(d model.Diff) string {
	var b strings.Builder

	for _, p := range d.Added {
		fmt.Fprintf(&b, "  + %-30s %-15s %s\n", p.Name, p.Version, p.Source)
	}
	for _, e := range d.Upgraded {
		fmt.Fprintf(&b, "  ↑ %-30s %-7s → %-7s %s\n", e.New.Name, e.Old.Version, e.New.Version, e.New.Source)
	}
	for _, p := range d.Removed {
		fmt.Fprintf(&b, "  - %-30s %-15s %s\n", p.Name, p.Version, p.Source)
	}

	fmt.Fprintf(&b, "\n  +%d added    %d upgraded    %d removed\n",
		len(d.Added), len(d.Upgraded), len(d.Removed))

	return b.String()
}
