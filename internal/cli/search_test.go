package cli

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestRankSearchRows(t *testing.T) {
	rows := []searchRow{
		{pkg: model.Package{Name: "ripgrep-all"}},
		{pkg: model.Package{Name: "repgrep"}},
		{pkg: model.Package{Name: "ripgrep"}},
	}
	rankSearchRows(rows, "ripgrep")
	if rows[0].pkg.Name != "ripgrep" {
		t.Errorf("exact match should rank first, got %q", rows[0].pkg.Name)
	}
	if rows[1].pkg.Name != "ripgrep-all" {
		t.Errorf("prefix match should rank second, got %q", rows[1].pkg.Name)
	}
}

func TestSearchManagersDedupesToCanonical(t *testing.T) {
	pacman := &fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{
				{Name: "x", Source: model.SourcePacman},
				{Name: "yay", Source: model.SourceAUR},
			}, nil
		},
	}
	aur := &fakeManager{
		name: model.SourceAUR, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "yay", Source: model.SourceAUR}}, nil
		},
	}
	rows := searchManagers([]manager.Manager{pacman, aur}, "x")
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	for _, r := range rows {
		if r.pkg.Name == "yay" && r.mgr.Name() != model.SourceAUR {
			t.Errorf("yay should map to the aur manager, got %s", r.mgr.Name())
		}
	}
}

func TestMarkInstalledRows(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	manager.SaveScanCache([]model.Package{{Name: "x", Source: model.SourcePacman}})
	rows := []searchRow{
		{pkg: model.Package{Name: "x", Source: model.SourcePacman}},
		{pkg: model.Package{Name: "y", Source: model.SourcePacman}},
	}
	markInstalledRows(rows)
	if !rows[0].installed {
		t.Error("x should be marked installed")
	}
	if rows[1].installed {
		t.Error("y should not be marked installed")
	}
}
