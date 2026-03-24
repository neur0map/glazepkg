package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/config"
)

func renderThemeOverlay(themes []config.Theme, cursor int, activeID string, width, height int) string {
	var b strings.Builder
	b.WriteString(StyleOverlayTitle.Render("  Theme"))
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", 36)))
	b.WriteString("\n\n")

	separatorDrawn := false
	for i, t := range themes {
		if !separatorDrawn && i > 0 && !t.Builtin && t.ID != "system" {
			prev := themes[i-1]
			if prev.Builtin || prev.ID == "system" {
				b.WriteString(StyleDim.Render("  " + strings.Repeat("─", 36)))
				b.WriteString("\n")
				separatorDrawn = true
			}
		}

		marker := "  "
		if t.ID == activeID {
			marker = "● "
		}

		style := lipgloss.NewStyle().Foreground(ColorText)
		if i == cursor {
			style = lipgloss.NewStyle().
				Reverse(true).
				Bold(true)
		}

		b.WriteString("  ")
		b.WriteString(StyleDim.Render(marker))
		b.WriteString(style.Render(t.Name))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  j/k navigate  enter apply  esc cancel"))

	overlayWidth := 44
	overlay := StyleOverlay.
		Width(overlayWidth).
		Render(b.String())

	// Horizontally center, no vertical padding
	overlayW := lipgloss.Width(overlay)
	padLeft := (width - overlayW) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	var out strings.Builder
	for _, line := range strings.Split(overlay, "\n") {
		out.WriteString(strings.Repeat(" ", padLeft))
		out.WriteString(line)
		out.WriteString("\n")
	}
	return out.String()
}
