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
		{"e (detail)", "Edit description"},
		{"d (detail)", "View dependencies"},
		{"h (detail)", "Package help/usage"},
		{"f", "Cycle filter"},
		{"r", "Rescan all managers"},
		{"s", "Save snapshot"},
		{"d", "Diff against last snapshot"},
		{"e", "Export packages"},
		{"?", "Toggle this help"},
		{"q", "Quit"},
	}

	var b strings.Builder
	b.WriteString(StyleOverlayTitle.Render("  Keybinds"))
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", 36)))
	b.WriteString("\n\n")

	for _, kb := range keybinds {
		keyStyle := lipgloss.NewStyle().
			Foreground(ColorCyan).
			Width(18)
		descStyle := lipgloss.NewStyle().
			Foreground(ColorText)

		b.WriteString("  ")
		b.WriteString(keyStyle.Render(kb.key))
		b.WriteString(descStyle.Render(kb.desc))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  Press any key to dismiss"))

	content := b.String()

	// Center the overlay
	overlayWidth := 44
	overlayHeight := len(keybinds) + 7

	overlay := StyleOverlay.
		Width(overlayWidth).
		Height(overlayHeight).
		Render(content)

	return placeOverlay(width, height, overlay)
}

func placeOverlay(width, height int, overlay string) string {
	overlayW := lipgloss.Width(overlay)
	overlayH := lipgloss.Height(overlay)

	padLeft := (width - overlayW) / 2
	padTop := (height - overlayH) / 2

	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	var b strings.Builder
	for range padTop {
		b.WriteString("\n")
	}

	lines := strings.Split(overlay, "\n")
	for _, line := range lines {
		b.WriteString(strings.Repeat(" ", padLeft))
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}
