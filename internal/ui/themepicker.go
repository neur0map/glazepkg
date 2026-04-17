package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// themeBody returns the theme list with active marker + cursor highlight.
// Pure content — no title, no frame, no overlay.
//
// The `●` marker tracks the PERSISTED theme (m.prevThemeID), not the
// cursor/preview position. During live navigation the cursor highlight moves
// independently of the marker; the marker only shifts once Enter commits the
// selection.
func themeBody(m *Model) string {
	var b strings.Builder
	separatorDrawn := false
	for i, t := range m.themeList {
		if !separatorDrawn && i > 0 && !t.Builtin && t.ID != "system" {
			prev := m.themeList[i-1]
			if prev.Builtin || prev.ID == "system" {
				b.WriteString(StyleDim.Render(strings.Repeat("─", 38)))
				b.WriteString("\n")
				separatorDrawn = true
			}
		}

		marker := "  "
		if t.ID == m.prevThemeID {
			marker = "● "
		}

		style := lipgloss.NewStyle().Foreground(ColorText)
		if i == m.themeCursor {
			style = lipgloss.NewStyle().Reverse(true).Bold(true)
		}

		b.WriteString(StyleDim.Render(marker))
		b.WriteString(style.Render(t.Name))
		if i < len(m.themeList)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
