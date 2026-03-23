package ui

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestIsPrivilegedSource(t *testing.T) {
	privileged := []model.Source{
		model.SourceApt, model.SourceDnf, model.SourcePacman,
		model.SourceSnap, model.SourceApk, model.SourceXbps,
		model.SourceChocolatey,
	}
	for _, src := range privileged {
		if !isPrivilegedSource(src) {
			t.Errorf("%s should require confirmation", src)
		}
	}

	unprivileged := []model.Source{
		model.SourceBrew, model.SourcePip, model.SourceCargo,
		model.SourceNpm, model.SourceFlatpak, model.SourceGem,
	}
	for _, src := range unprivileged {
		if isPrivilegedSource(src) {
			t.Errorf("%s should not require confirmation", src)
		}
	}
}

func TestSelectedPackageNegativeCursor(t *testing.T) {
	m := Model{
		cursor:       -1,
		filteredPkgs: []model.Package{{Name: "curl"}},
	}
	_, ok := m.selectedPackage()
	if ok {
		t.Error("negative cursor should return false")
	}
}

func TestSelectedPackageEmptyList(t *testing.T) {
	m := Model{
		cursor:       0,
		filteredPkgs: nil,
	}
	_, ok := m.selectedPackage()
	if ok {
		t.Error("empty list should return false")
	}
}

func TestSelectedPackageBeyondBounds(t *testing.T) {
	m := Model{
		cursor:       5,
		filteredPkgs: []model.Package{{Name: "curl"}},
	}
	_, ok := m.selectedPackage()
	if ok {
		t.Error("cursor beyond bounds should return false")
	}
}

func TestSelectedPackageValid(t *testing.T) {
	pkgs := []model.Package{
		{Name: "curl", Source: model.SourceBrew},
		{Name: "wget", Source: model.SourceBrew},
	}
	m := Model{
		cursor:       1,
		filteredPkgs: pkgs,
	}
	pkg, ok := m.selectedPackage()
	if !ok {
		t.Fatal("expected valid selection")
	}
	if pkg.Name != "wget" {
		t.Errorf("expected wget, got %s", pkg.Name)
	}
}
