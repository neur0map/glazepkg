package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// UpdateChecker is implemented by managers that can check for available updates.
type UpdateChecker interface {
	// CheckUpdates returns a map of package name → latest available version.
	CheckUpdates(pkgs []model.Package) map[string]string
}

// UpdateCache persists update info to disk with a 7-day TTL.
type UpdateCache struct {
	mu      sync.RWMutex
	entries map[string]updateEntry
	path    string
}

type updateEntry struct {
	Latest  string    `json:"latest"`
	Fetched time.Time `json:"fetched"`
}

const updateCacheTTL = 7 * 24 * time.Hour

func NewUpdateCache() *UpdateCache {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	path := filepath.Join(base, "glazepkg", "cache", "updates.json")

	uc := &UpdateCache{
		entries: make(map[string]updateEntry),
		path:    path,
	}
	uc.load()
	return uc
}

func (uc *UpdateCache) load() {
	data, err := os.ReadFile(uc.path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &uc.entries)
}

func (uc *UpdateCache) save() {
	dir := filepath.Dir(uc.path)
	_ = os.MkdirAll(dir, 0o755)
	data, err := json.Marshal(uc.entries)
	if err != nil {
		return
	}
	_ = os.WriteFile(uc.path, data, 0o644)
}

func (uc *UpdateCache) Get(key string) (string, bool) {
	uc.mu.RLock()
	defer uc.mu.RUnlock() //nolint:errcheck
	e, ok := uc.entries[key]
	if !ok || time.Since(e.Fetched) > updateCacheTTL {
		return "", false
	}
	return e.Latest, true
}

func (uc *UpdateCache) Set(key, latest string) {
	uc.mu.Lock()
	defer uc.mu.Unlock() //nolint:errcheck
	uc.entries[key] = updateEntry{Latest: latest, Fetched: time.Now()}
}

func (uc *UpdateCache) Flush() {
	uc.mu.RLock()
	defer uc.mu.RUnlock() //nolint:errcheck
	uc.save()
}

func (uc *UpdateCache) Invalidate(keys []string) {
	if len(keys) == 0 {
		return
	}
	uc.mu.Lock()
	defer uc.mu.Unlock() //nolint:errcheck
	for _, key := range keys {
		delete(uc.entries, key)
	}
	uc.save()
}

func (uc *UpdateCache) Invalidate(keys []string) {
	if len(keys) == 0 {
		return
	}
	uc.mu.Lock()
	defer uc.mu.Unlock()
	for _, key := range keys {
		delete(uc.entries, key)
	}
	uc.save()
}

// FetchUpdates checks for available updates across all managers, using the cache.
// Returns a map of package key → latest version.
func FetchUpdates(mgrs []Manager, pkgs []model.Package, cache *UpdateCache) map[string]string {
	// Group packages by source
	bySource := make(map[model.Source][]model.Package)
	for _, p := range pkgs {
		bySource[p.Source] = append(bySource[p.Source], p)
	}

	var mu sync.Mutex
	result := make(map[string]string)
	var wg sync.WaitGroup

	for _, mgr := range mgrs {
		checker, ok := mgr.(UpdateChecker)
		if !ok {
			continue
		}
		srcPkgs := bySource[mgr.Name()]
		if len(srcPkgs) == 0 {
			continue
		}

		// Check cache first
		var uncached []model.Package
		for _, p := range srcPkgs {
			if latest, ok := cache.Get(p.Key()); ok {
				if latest != p.Version {
					mu.Lock()
					result[p.Key()] = latest
					mu.Unlock()
				}
			} else {
				uncached = append(uncached, p)
			}
		}

		if len(uncached) == 0 {
			continue
		}

		wg.Add(1)
		go func(c UpdateChecker, pkgs []model.Package) {
			defer wg.Done()
			fetched := c.CheckUpdates(pkgs)
			mu.Lock()
			for _, p := range pkgs {
				if latest, ok := fetched[p.Name]; ok && latest != "" {
					cache.Set(p.Key(), latest)
					if latest != p.Version {
						result[p.Key()] = latest
					}
				} else {
					cache.Set(p.Key(), p.Version)
				}
			}
			mu.Unlock()
		}(checker, uncached)
	}

	wg.Wait()
	cache.Flush()
	return result
}
