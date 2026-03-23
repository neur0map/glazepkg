package manager

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestCache(t *testing.T) *UpdateCache {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "updates.json")
	return &UpdateCache{
		entries: make(map[string]updateEntry),
		path:    path,
	}
}

func TestInvalidateRemovesKeys(t *testing.T) {
	uc := newTestCache(t)
	uc.Set("brew:curl", "8.0")
	uc.Set("brew:wget", "1.21")
	uc.Set("pip:requests", "2.31")

	uc.Invalidate([]string{"brew:curl", "brew:wget"})

	if _, ok := uc.Get("brew:curl"); ok {
		t.Error("brew:curl should have been invalidated")
	}
	if _, ok := uc.Get("brew:wget"); ok {
		t.Error("brew:wget should have been invalidated")
	}
	if _, ok := uc.Get("pip:requests"); !ok {
		t.Error("pip:requests should still be present")
	}
}

func TestInvalidateEmptyKeys(t *testing.T) {
	uc := newTestCache(t)
	uc.Set("brew:curl", "8.0")
	uc.Invalidate(nil)
	uc.Invalidate([]string{})
	if _, ok := uc.Get("brew:curl"); !ok {
		t.Error("empty invalidate should not remove entries")
	}
}

func TestInvalidatePersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "updates.json")

	uc := &UpdateCache{
		entries: make(map[string]updateEntry),
		path:    path,
	}
	uc.Set("brew:curl", "8.0")
	uc.Set("brew:wget", "1.21")
	uc.Flush()

	uc.Invalidate([]string{"brew:curl"})

	uc2 := &UpdateCache{
		entries: make(map[string]updateEntry),
		path:    path,
	}
	uc2.load()
	if _, ok := uc2.entries["brew:curl"]; ok {
		t.Error("invalidated key should not be present after reload")
	}
	if _, ok := uc2.entries["brew:wget"]; !ok {
		t.Error("non-invalidated key should persist after reload")
	}
}

func TestInvalidateMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "updates.json")
	uc := &UpdateCache{
		entries: make(map[string]updateEntry),
		path:    path,
	}
	uc.Set("brew:curl", "8.0")
	uc.Invalidate([]string{"brew:curl"})

	// Should have created the directory and written the file.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected cache file to be created: %v", err)
	}
}
