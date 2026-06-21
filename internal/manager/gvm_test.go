package manager

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestParseGvmVersions(t *testing.T) {
	names := []string{"go1.21.0", "system", "go1.20.5", "go1.4", ".keep"}
	pkgs := parseGvmVersions("/r", names)

	// Sorted by Name; "system" and dotfiles excluded.
	want := []model.Package{
		{Name: "go1.20.5", Version: "1.20.5", Source: model.SourceGvm, Location: "/r/gos/go1.20.5"},
		{Name: "go1.21.0", Version: "1.21.0", Source: model.SourceGvm, Location: "/r/gos/go1.21.0"},
		{Name: "go1.4", Version: "1.4", Source: model.SourceGvm, Location: "/r/gos/go1.4"},
	}
	if len(pkgs) != len(want) {
		t.Fatalf("got %d packages, want %d", len(pkgs), len(want))
	}
	for i, w := range want {
		got := pkgs[i]
		if got.Name != w.Name {
			t.Errorf("pkgs[%d].Name = %q, want %q", i, got.Name, w.Name)
		}
		if got.Version != w.Version {
			t.Errorf("pkgs[%d].Version = %q, want %q", i, got.Version, w.Version)
		}
		if got.Source != w.Source {
			t.Errorf("pkgs[%d].Source = %q, want %q", i, got.Source, w.Source)
		}
		if got.Location != w.Location {
			t.Errorf("pkgs[%d].Location = %q, want %q", i, got.Location, w.Location)
		}
	}
}

func TestParseGvmVersionsEmpty(t *testing.T) {
	if pkgs := parseGvmVersions("/r", []string{}); len(pkgs) != 0 {
		t.Fatalf("got %d packages, want 0", len(pkgs))
	}
	if pkgs := parseGvmVersions("/r", nil); len(pkgs) != 0 {
		t.Fatalf("got %d packages, want 0", len(pkgs))
	}
}
