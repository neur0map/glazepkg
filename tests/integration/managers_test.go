package integration

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
)

// TestManagerScan runs a real scan for each available manager.
// Skips managers that aren't installed on the current system.
func TestManagerScan(t *testing.T) {
	for _, mgr := range manager.All() {
		mgr := mgr
		t.Run(string(mgr.Name()), func(t *testing.T) {
			if !mgr.Available() {
				t.Skipf("%s not available on this system", mgr.Name())
			}
			pkgs, err := mgr.Scan()
			if err != nil {
				t.Fatalf("Scan() error: %v", err)
			}
			t.Logf("%s: scanned %d packages", mgr.Name(), len(pkgs))

			for _, p := range pkgs {
				if p.Name == "" {
					t.Error("package with empty name")
				}
			}
		})
	}
}

// TestManagerCheckUpdates runs update checks for available managers.
func TestManagerCheckUpdates(t *testing.T) {
	for _, mgr := range manager.All() {
		checker, ok := mgr.(interface {
			CheckUpdates([]interface{}) map[string]string
		})
		_ = checker
		if !ok {
			continue
		}
		mgr := mgr
		t.Run(string(mgr.Name()), func(t *testing.T) {
			if !mgr.Available() {
				t.Skipf("%s not available on this system", mgr.Name())
			}
			t.Logf("%s: update checker available", mgr.Name())
		})
	}
}
