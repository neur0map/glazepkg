package ui

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestBuildTabsKeepsBrewCaskOutOfAll(t *testing.T) {
	tabs := buildTabs([]model.Package{
		{Name: "curl", Source: model.SourceBrew},
		{Name: "firefox", Source: model.SourceBrewCask},
	})

	if len(tabs) == 0 || tabs[0].Label != "ALL" {
		t.Fatalf("first tab = %#v, want ALL", tabs)
	}
	if tabs[0].Count != 1 {
		t.Fatalf("ALL count = %d, want 1", tabs[0].Count)
	}
}

func TestBuildTabsPlacesCaskAfterBrew(t *testing.T) {
	tabs := buildTabs([]model.Package{
		{Name: "curl", Source: model.SourceBrew},
		{Name: "firefox", Source: model.SourceBrewCask},
		{Name: "git", Source: model.SourceApt},
	})

	want := []string{"ALL", "brew", "cask", "apt"}
	if len(tabs) < len(want) {
		t.Fatalf("got %d tabs, want at least %d: %#v", len(tabs), len(want), tabs)
	}
	for i, label := range want {
		if tabs[i].Label != label {
			t.Fatalf("tab %d = %q, want %q; tabs=%#v", i, tabs[i].Label, label, tabs)
		}
	}
}

func TestBuildTabsOmitsCaskWhenNoCasksExist(t *testing.T) {
	tabs := buildTabs([]model.Package{{Name: "curl", Source: model.SourceBrew}})
	for _, tab := range tabs {
		if tab.Label == "cask" || tab.Source == string(model.SourceBrewCask) {
			t.Fatalf("unexpected cask tab with no cask packages: %#v", tabs)
		}
	}
}

func TestApplyFilterKeepsBrewCaskOutOfAllButShowsCaskTab(t *testing.T) {
	pkgs := []model.Package{
		{Name: "curl", Source: model.SourceBrew},
		{Name: "firefox", Source: model.SourceBrewCask},
	}
	m := Model{
		allPkgs: pkgs,
		tabs:    buildTabs(pkgs),
	}

	m.applyFilter()
	if got := packageNames(m.filteredPkgs); len(got) != 1 || got[0] != "curl" {
		t.Fatalf("ALL filtered packages = %#v, want [curl]", got)
	}

	for i, tab := range m.tabs {
		if tab.Source == string(model.SourceBrewCask) {
			m.activeTab = i
			break
		}
	}
	m.applyFilter()
	if got := packageNames(m.filteredPkgs); len(got) != 1 || got[0] != "firefox" {
		t.Fatalf("cask filtered packages = %#v, want [firefox]", got)
	}
}

func packageNames(pkgs []model.Package) []string {
	names := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		names[i] = pkg.Name
	}
	return names
}
