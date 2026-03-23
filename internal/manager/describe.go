package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// Describer is implemented by managers that can fetch package descriptions.
type Describer interface {
	Describe(pkgs []model.Package) map[string]string
}

// DescriptionCache persists descriptions to disk with a 24h TTL.
type DescriptionCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	path    string
}

type cacheEntry struct {
	Desc    string    `json:"desc"`
	Fetched time.Time `json:"fetched"`
}

const cacheTTL = 24 * time.Hour

func NewDescriptionCache() *DescriptionCache {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	path := filepath.Join(base, "glazepkg", "cache", "descriptions.json")

	dc := &DescriptionCache{
		entries: make(map[string]cacheEntry),
		path:    path,
	}
	dc.load()
	return dc
}

func (dc *DescriptionCache) load() {
	data, err := os.ReadFile(dc.path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &dc.entries)
}

func (dc *DescriptionCache) save() {
	dir := filepath.Dir(dc.path)
	_ = os.MkdirAll(dir, 0o755)
	data, err := json.Marshal(dc.entries)
	if err != nil {
		return
	}
	_ = os.WriteFile(dc.path, data, 0o644)
}

// Get returns a cached description if it exists and hasn't expired.
func (dc *DescriptionCache) Get(key string) (string, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	e, ok := dc.entries[key]
	if !ok || time.Since(e.Fetched) > cacheTTL {
		return "", false
	}
	return e.Desc, true
}

// Set stores a description in the cache.
func (dc *DescriptionCache) Set(key, desc string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.entries[key] = cacheEntry{Desc: desc, Fetched: time.Now()}
}

// Flush writes the cache to disk.
func (dc *DescriptionCache) Flush() {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	dc.save()
}

// SanitizeDesc strips HTML tags and collapses whitespace from descriptions.
func SanitizeDesc(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// FetchDescriptions fetches descriptions for all packages, using the cache
// where possible and falling back to the manager's Describer interface.
func FetchDescriptions(mgrs []Manager, pkgs []model.Package, cache *DescriptionCache) map[string]string {
	// Group packages by source
	bySource := make(map[model.Source][]model.Package)
	for _, p := range pkgs {
		bySource[p.Source] = append(bySource[p.Source], p)
	}

	var mu sync.Mutex
	result := make(map[string]string)
	var wg sync.WaitGroup

	for _, mgr := range mgrs {
		desc, ok := mgr.(Describer)
		if !ok {
			continue
		}
		srcPkgs := bySource[mgr.Name()]
		if len(srcPkgs) == 0 {
			continue
		}

		// Check cache first — only fetch uncached ones
		var uncached []model.Package
		for _, p := range srcPkgs {
			if d, ok := cache.Get(p.Key()); ok {
				if d != "" {
					mu.Lock()
					result[p.Key()] = SanitizeDesc(d)
					mu.Unlock()
				}
			} else if p.Description != "" {
				cache.Set(p.Key(), p.Description)
				mu.Lock()
				result[p.Key()] = SanitizeDesc(p.Description)
				mu.Unlock()
			} else {
				uncached = append(uncached, p)
			}
		}

		if len(uncached) == 0 {
			continue
		}

		wg.Add(1)
		go func(d Describer, pkgs []model.Package) {
			defer wg.Done()
			fetched := d.Describe(pkgs)
			mu.Lock()
			for _, p := range pkgs {
				if desc, ok := fetched[p.Name]; ok && desc != "" {
					desc = SanitizeDesc(desc)
					result[p.Key()] = desc
					cache.Set(p.Key(), desc)
				} else {
					cache.Set(p.Key(), "")
				}
			}
			mu.Unlock()
		}(desc, uncached)
	}

	wg.Wait()
	cache.Flush()
	return result
}
