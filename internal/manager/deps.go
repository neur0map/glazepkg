package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// DependencyLister is implemented by managers that can fetch package dependencies.
type DependencyLister interface {
	ListDependencies(pkgs []model.Package) map[string][]string
}

// DepsCache persists dependency lists to disk with a 24h TTL.
type DepsCache struct {
	mu      sync.RWMutex
	entries map[string]depsCacheEntry
	path    string
}

type depsCacheEntry struct {
	Deps    []string  `json:"deps"`
	Fetched time.Time `json:"fetched"`
}

func NewDepsCache() *DepsCache {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	path := filepath.Join(base, "glazepkg", "cache", "dependencies.json")

	dc := &DepsCache{
		entries: make(map[string]depsCacheEntry),
		path:    path,
	}
	dc.load()
	return dc
}

func (dc *DepsCache) load() {
	data, err := os.ReadFile(dc.path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &dc.entries)
}

func (dc *DepsCache) save() {
	dir := filepath.Dir(dc.path)
	_ = os.MkdirAll(dir, 0o755)
	data, err := json.Marshal(dc.entries)
	if err != nil {
		return
	}
	_ = os.WriteFile(dc.path, data, 0o644)
}

// Get returns cached dependencies if they exist and haven't expired.
func (dc *DepsCache) Get(key string) ([]string, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	e, ok := dc.entries[key]
	if !ok || time.Since(e.Fetched) > cacheTTL {
		return nil, false
	}
	return e.Deps, true
}

// Set stores a dependency list in the cache.
func (dc *DepsCache) Set(key string, deps []string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.entries[key] = depsCacheEntry{Deps: deps, Fetched: time.Now()}
}

// Flush writes the cache to disk.
func (dc *DepsCache) Flush() {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	dc.save()
}

// FetchDependencies fetches dependency lists for all packages, using the cache
// where possible and falling back to the manager's DependencyLister interface.
func FetchDependencies(mgrs []Manager, pkgs []model.Package, cache *DepsCache) map[string][]string {
	result := make(map[string][]string)

	// Group packages by source
	bySource := make(map[model.Source][]model.Package)
	for _, p := range pkgs {
		bySource[p.Source] = append(bySource[p.Source], p)
	}

	for _, mgr := range mgrs {
		lister, ok := mgr.(DependencyLister)
		if !ok {
			continue
		}
		source := mgr.Name()
		srcPkgs := bySource[source]
		if len(srcPkgs) == 0 {
			continue
		}

		// Check cache first — only fetch uncached ones
		var uncached []model.Package
		for _, p := range srcPkgs {
			if deps, ok := cache.Get(p.Key()); ok {
				if len(deps) > 0 {
					result[p.Key()] = deps
				}
			} else {
				uncached = append(uncached, p)
			}
		}

		if len(uncached) == 0 {
			continue
		}

		fetched := lister.ListDependencies(uncached)
		for _, p := range uncached {
			if deps, ok := fetched[p.Name]; ok {
				if len(deps) > 0 {
					result[p.Key()] = deps
				}
				cache.Set(p.Key(), deps)
			}
		}
	}

	cache.Flush()
	return result
}
