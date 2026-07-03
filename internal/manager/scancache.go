package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

const scanCacheTTL = 10 * 24 * time.Hour // 10 days

type scanCache struct {
	Timestamp time.Time       `json:"timestamp"`
	Packages  []model.Package `json:"packages"`
}

func scanCachePath() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "glazepkg", "cache", "scan.json")
}

// LoadScanCache returns cached packages if the cache exists and is fresh.
// Returns nil if stale or missing.
func LoadScanCache() []model.Package {
	data, err := os.ReadFile(scanCachePath())
	if err != nil {
		return nil
	}
	var c scanCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	if time.Since(c.Timestamp) > scanCacheTTL {
		return nil
	}
	return c.Packages
}

// ScanCacheAge returns how old the cache is, or -1 if no cache.
func ScanCacheAge() time.Duration {
	data, err := os.ReadFile(scanCachePath())
	if err != nil {
		return -1
	}
	var c scanCache
	if err := json.Unmarshal(data, &c); err != nil {
		return -1
	}
	return time.Since(c.Timestamp)
}

// UpdateScanCache surgically applies installs and removals to the cache file
// so write operations don't force a full rescan on the next gpk list. Entries
// in add are upserted (keyed by source:name, InstalledAt stamped with now if
// zero); keys in removeKeys are dropped. The original Timestamp is preserved
// so an updated cache never masquerades as a fresh scan. A missing cache file
// is a no-op: search treats an absent cache as unknown, matching the old
// delete-the-file behavior.
func UpdateScanCache(add []model.Package, removeKeys []string) error {
	path := scanCachePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var c scanCache
	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}

	index := make(map[string]int, len(c.Packages))
	for i, p := range c.Packages {
		index[string(p.Source)+":"+p.Name] = i
	}
	for _, p := range add {
		key := string(p.Source) + ":" + p.Name
		if i, ok := index[key]; ok {
			// Merge onto the existing entry: callers often pass just
			// {Name, Source}, and blanking the scanned fields would
			// degrade gpk list output.
			merged := c.Packages[i]
			if p.Version != "" {
				merged.Version = p.Version
			}
			if !p.InstalledAt.IsZero() {
				merged.InstalledAt = p.InstalledAt
			}
			if merged.InstalledAt.IsZero() {
				merged.InstalledAt = time.Now()
			}
			c.Packages[i] = merged
		} else {
			if p.InstalledAt.IsZero() {
				p.InstalledAt = time.Now()
			}
			c.Packages = append(c.Packages, p)
			index[key] = len(c.Packages) - 1
		}
	}
	if len(removeKeys) > 0 {
		drop := make(map[string]bool, len(removeKeys))
		for _, k := range removeKeys {
			drop[k] = true
		}
		kept := c.Packages[:0]
		for _, p := range c.Packages {
			if !drop[string(p.Source)+":"+p.Name] {
				kept = append(kept, p)
			}
		}
		c.Packages = kept
	}

	out, err := json.Marshal(c)
	if err != nil {
		return err
	}
	// Temp file + rename so a concurrent reader never sees a partial write.
	tmp, err := os.CreateTemp(filepath.Dir(path), "scan-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	// CreateTemp uses 0600; match SaveScanCache's 0644.
	if err := os.Chmod(tmpName, 0o644); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// DeleteScanCache removes the cache file entirely so the next gpk list does
// a fresh scan. Used when a write's effect on installed packages is
// ambiguous (autoremove, history undo).
func DeleteScanCache() {
	_ = os.Remove(scanCachePath())
}

// SaveScanCache writes the package list to the cache file.
func SaveScanCache(pkgs []model.Package) {
	c := scanCache{
		Timestamp: time.Now(),
		Packages:  pkgs,
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	path := scanCachePath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, data, 0o644)
}
