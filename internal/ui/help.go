package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// helpBody returns the keybinds reference text shown inside the help modal.
// Pure content — caller is responsible for framing.
func helpBody() string {
	keybinds := []struct {
		key  string
		desc string
	}{
		{"j/k, ↑/↓", "Navigate up/down"},
		{"g/G", "Jump to top/bottom"},
		{"Ctrl+d/u", "Half page down/up"},
		{"PgDn/PgUp", "Page down/up"},
		{"Tab/Shift+Tab", "Cycle manager tabs"},
		{"/ or Ctrl+f", "Fuzzy search"},
		{"Esc", "Clear search / close overlay"},
		{"Enter", "Package details"},
		{"u (detail)", "Upgrade package"},
		{"x (detail)", "Remove package"},
		{"e (detail)", "Edit description"},
		{"d (detail)", "View dependencies"},
		{"h (detail)", "Package help/usage"},
		{"f", "Cycle filter"},
		{"r", "Rescan all managers"},
		{"s", "Save snapshot"},
		{"i", "Search + install packages"},
		{"d", "Diff against last snapshot"},
		{"e", "Export packages"},
		{"t", "Switch theme"},
		{"?/h", "Toggle this help"},
		{"q", "Quit"},
	}

	var b strings.Builder
	for i, kb := range keybinds {
		keyStyle := lipgloss.NewStyle().Foreground(ColorCyan).Width(18)
		descStyle := lipgloss.NewStyle().Foreground(ColorText)
		b.WriteString(keyStyle.Render(kb.key))
		b.WriteString(descStyle.Render(kb.desc))
		if i < len(keybinds)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n\n")
	b.WriteString(StyleDim.Render("RU layout maps to same key positions"))
	return b.String()
}
