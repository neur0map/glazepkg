package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestNormalizeHotkeyMapsRussianLayout(t *testing.T) {
	tests := map[string]string{
		"\u0439":      "q",
		"\u043e":      "j",
		"\u043b":      "k",
		"\u043f":      "g",
		"\u041f":      "G",
		"\u0440":      "h",
		"ctrl+\u0441": "ctrl+c",
		".":           "/",
		",":           "?",
	}

	for input, want := range tests {
		if got := normalizeHotkey(input); got != want {
			t.Fatalf("normalizeHotkey(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestHandleKeyNavigatesWithRussianHotkeys(t *testing.T) {
	m := &Model{
		filteredPkgs: []model.Package{
			{Name: "curl"},
			{Name: "wget"},
		},
	}

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("\u043e")})

	if m.cursor != 1 {
		t.Fatalf("expected cursor to move down, got %d", m.cursor)
	}
}

func TestHandleKeyShowsHelpFromRussianLayout(t *testing.T) {
	m := &Model{}

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("\u0440")})

	if !m.showHelp {
		t.Fatal("expected help overlay to open from Russian-layout hotkey")
	}
}
