package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// seedScanCache writes a cache file directly so tests control the Timestamp.
func seedScanCache(t *testing.T, c scanCache) {
	t.Helper()
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := scanCachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// readScanCache reads the raw cache file back, bypassing the TTL check.
func readScanCache(t *testing.T) scanCache {
	t.Helper()
	data, err := os.ReadFile(scanCachePath())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var c scanCache
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return c
}

func TestUpdateScanCacheUpsertNew(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ts := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	seedScanCache(t, scanCache{
		Timestamp: ts,
		Packages:  []model.Package{{Name: "bash", Source: model.SourcePacman, Version: "5.2"}},
	})

	err := UpdateScanCache([]model.Package{{Name: "ripgrep", Source: model.SourcePacman}}, nil)
	if err != nil {
		t.Fatalf("UpdateScanCache: %v", err)
	}

	c := readScanCache(t)
	if len(c.Packages) != 2 {
		t.Fatalf("got %d packages, want 2", len(c.Packages))
	}
	var rg *model.Package
	for i := range c.Packages {
		if c.Packages[i].Name == "ripgrep" {
			rg = &c.Packages[i]
		}
	}
	if rg == nil {
		t.Fatal("ripgrep not upserted")
	}
	if rg.InstalledAt.IsZero() {
		t.Error("new entry should get InstalledAt stamped")
	}
	if !c.Timestamp.Equal(ts) {
		t.Errorf("Timestamp = %v, want original %v", c.Timestamp, ts)
	}
}

func TestUpdateScanCacheUpsertExistingPreservesInstalledAt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ts := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	installedAt := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	seedScanCache(t, scanCache{
		Timestamp: ts,
		Packages: []model.Package{
			{Name: "bash", Source: model.SourcePacman, Version: "5.2", InstalledAt: installedAt},
		},
	})

	// Skeletal upsert with a new version and a zero InstalledAt: the
	// version updates, the recorded install time survives.
	err := UpdateScanCache([]model.Package{{Name: "bash", Source: model.SourcePacman, Version: "5.3"}}, nil)
	if err != nil {
		t.Fatalf("UpdateScanCache: %v", err)
	}

	c := readScanCache(t)
	if len(c.Packages) != 1 {
		t.Fatalf("got %d packages, want 1", len(c.Packages))
	}
	got := c.Packages[0]
	if got.Version != "5.3" {
		t.Errorf("Version = %q, want 5.3", got.Version)
	}
	if !got.InstalledAt.Equal(installedAt) {
		t.Errorf("InstalledAt = %v, want preserved %v", got.InstalledAt, installedAt)
	}
}

func TestUpdateScanCacheUpsertExistingStampsZeroInstalledAt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	seedScanCache(t, scanCache{
		Timestamp: time.Now(),
		Packages:  []model.Package{{Name: "bash", Source: model.SourcePacman, Version: "5.2"}},
	})

	err := UpdateScanCache([]model.Package{{Name: "bash", Source: model.SourcePacman}}, nil)
	if err != nil {
		t.Fatalf("UpdateScanCache: %v", err)
	}

	c := readScanCache(t)
	if len(c.Packages) != 1 {
		t.Fatalf("got %d packages, want 1", len(c.Packages))
	}
	if c.Packages[0].InstalledAt.IsZero() {
		t.Error("zero InstalledAt should get stamped on upsert")
	}
}

func TestUpdateScanCacheRemove(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ts := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	seedScanCache(t, scanCache{
		Timestamp: ts,
		Packages: []model.Package{
			{Name: "bash", Source: model.SourcePacman, Version: "5.2"},
			{Name: "ripgrep", Source: model.SourcePacman, Version: "14.1"},
		},
	})

	err := UpdateScanCache(nil, []string{"pacman:ripgrep"})
	if err != nil {
		t.Fatalf("UpdateScanCache: %v", err)
	}

	c := readScanCache(t)
	if len(c.Packages) != 1 {
		t.Fatalf("got %d packages, want 1", len(c.Packages))
	}
	if c.Packages[0].Name != "bash" {
		t.Errorf("kept %q, want bash", c.Packages[0].Name)
	}
	if !c.Timestamp.Equal(ts) {
		t.Errorf("Timestamp = %v, want original %v", c.Timestamp, ts)
	}
}

func TestUpdateScanCacheMissingFileIsNoop(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	err := UpdateScanCache([]model.Package{{Name: "ripgrep", Source: model.SourcePacman}}, nil)
	if err != nil {
		t.Fatalf("UpdateScanCache on missing file: %v", err)
	}
	if _, err := os.Stat(scanCachePath()); !os.IsNotExist(err) {
		t.Error("missing cache file should not be created by an update")
	}
}
