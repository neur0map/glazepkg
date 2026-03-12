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
