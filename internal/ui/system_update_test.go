package ui

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestActiveTabSource(t *testing.T) {
	m := &Model{tabs: []tabItem{{Source: ""}, {Source: "pacman"}}, activeTab: 1}
	if got := m.activeTabSource(); got != model.SourcePacman {
		t.Errorf("activeTabSource = %q, want pacman", got)
	}
	m.activeTab = 0
	if got := m.activeTabSource(); got != "" {
		t.Errorf("ALL tab source = %q, want empty", got)
	}
}

func TestSystemUpdateOnAllTab(t *testing.T) {
	m := &Model{tabs: []tabItem{{Label: "ALL", Source: ""}}, activeTab: 0}
	if cmd := m.systemUpdate(); cmd != nil {
		t.Error("expected no command on the ALL tab")
	}
	if m.statusMsg == "" {
		t.Error("expected a status hint guiding the user to a manager tab")
	}
}
