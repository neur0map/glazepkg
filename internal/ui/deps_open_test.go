package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestSelectedDepName(t *testing.T) {
	m := &Model{detailPkg: model.Package{
		DependsOn:  []string{"libc", "zlib"},
		RequiredBy: []string{"app1"},
	}}

	m.depsCursor = 0
	if got := m.selectedDepName(); got != "libc" {
		t.Errorf("cursor 0 = %q, want libc", got)
	}
	m.depsCursor = 2 // first RequiredBy
	if got := m.selectedDepName(); got != "app1" {
		t.Errorf("cursor 2 = %q, want app1", got)
	}
	m.depsCursor = 9 // out of range
	if got := m.selectedDepName(); got != "" {
		t.Errorf("out-of-range = %q, want empty", got)
	}
}

func TestFindPackageByName(t *testing.T) {
	m := &Model{allPkgs: []model.Package{{Name: "zlib", Source: model.SourcePacman}}}
	if p, ok := m.findPackageByName("zlib"); !ok || p.Source != model.SourcePacman {
		t.Errorf("findPackageByName(zlib) = %+v, %v", p, ok)
	}
	if _, ok := m.findPackageByName("nope"); ok {
		t.Error("expected miss for unknown package")
	}
}

func TestDepsModalEnterOpensPackage(t *testing.T) {
	m := &Model{
		modal:     ModalDeps,
		allPkgs:   []model.Package{{Name: "zlib", Version: "1.3", Source: model.SourcePacman}},
		detailPkg: model.Package{Name: "curl", DependsOn: []string{"zlib"}},
	}
	m.depsCursor = 0
	_, _ = handleDepsModalKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.detailPkg.Name != "zlib" {
		t.Errorf("detailPkg switched to %q, want zlib", m.detailPkg.Name)
	}
}

func TestDepsModalEnterMissingPackageNoop(t *testing.T) {
	m := &Model{
		modal:     ModalDeps,
		detailPkg: model.Package{Name: "curl", DependsOn: []string{"ghost"}},
	}
	m.depsCursor = 0
	_, _ = handleDepsModalKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.detailPkg.Name != "curl" {
		t.Errorf("detailPkg changed to %q, want unchanged curl", m.detailPkg.Name)
	}
}
