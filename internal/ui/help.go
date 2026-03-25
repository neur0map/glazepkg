package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderHelpOverlay(width, height int) string {
	keybinds := []struct {
		key  string
		desc string
	}{
		{"j/k, ↑/↓", "Navigate up/down"},
		{"g/G", "Jump to top/bottom"},
		{"Ctrl+d/u", "Half page down/up"},
		{"PgDn/PgUp", "Page down/up"},
		{"Tab/Shift+Tab", "Cycle manager tabs"},
		{"/", "Fuzzy search"},
		{"Esc", "Clear search / close overlay"},
		{"Enter", "Package details"},
		{"u (detail)", "Upgrade package"},
		{"x (detail)", "Remove package"},
		{"e (detail)", "Edit description"},
		{"d (detail)", "View dependencies"},
		{"h (detail)", "Package help/usage"},
		{"f", "Cycle size filter"},
		{"r", "Rescan all managers"},
		{"s", "Save snapshot"},
		{"i", "Search + install packages"},
		{"d", "Diff against last snapshot"},
		{"e", "Export packages"},
		{"t", "Switch theme"},
		{"?", "Toggle this help"},
		{"q", "Quit"},
	}

	var b strings.Builder
	b.WriteString(StyleOverlayTitle.Render("  Keybinds"))
	b.WriteString(StyleOverlayBase.Render("\n"))
	b.WriteString(StyleOverlayBase.Copy().Foreground(ColorSubtext).Render("  " + strings.Repeat("─", 36)))
	b.WriteString(StyleOverlayBase.Render("\n\n"))

	for _, kb := range keybinds {
		keyStyle := lipgloss.NewStyle().
			Foreground(ColorCyan).
			Background(ColorOverlay).
			Width(18)
		descStyle := lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorOverlay)

		b.WriteString(StyleOverlayBase.Render("  "))
		b.WriteString(keyStyle.Render(kb.key))
		b.WriteString(descStyle.Render(kb.desc))
		b.WriteString(StyleOverlayBase.Render("\n"))
	}

	b.WriteString(StyleOverlayBase.Render("\n"))
	b.WriteString(StyleOverlayBase.Copy().Foreground(ColorSubtext).Render("  Press any key to dismiss"))

	content := b.String()

	overlayWidth := 44
	overlayHeight := len(keybinds) + 7

	overlay := StyleOverlay.
		Width(overlayWidth).
		Height(overlayHeight).
		Render(content)

	return placeOverlay(width, height, overlay)
}

func placeOverlay(width, height int, overlay string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay, lipgloss.WithWhitespaceBackground(ColorBase))
}
