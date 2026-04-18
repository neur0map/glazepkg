package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

// renderDetail builds the package detail view as a bordered-panel layout.
// The whole string returned is the full body of the detail view — the
// top title bar is rendered by View(); renderDetail owns everything below
// that, including the bottom keybind bar.
func renderDetail(m *Model) string {
	pkg := m.detailPkg
	w, h := m.width, m.height

	borderColor := ColorSubtext
	accentColor := ColorBlue

	// Inner panel style (Info / Description / Depends / Required-by).
	innerPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	// Figure out how wide the outer panel should be and how much of that
	// the inner panels can consume. The outer panel has border (2) + padding
	// (4) = 6 cols of chrome. Inner panels, when side-by-side, share a 2-col
	// gutter between them. Each inner panel has its own border (2) + padding
	// (2) = 4 cols of chrome.
	outerMaxW := w - 6 // leave 3 cols of horizontal margin on each side
	if outerMaxW < 40 {
		outerMaxW = 40
	}
	if outerMaxW > 92 {
		outerMaxW = 92
	}
	outerInnerW := outerMaxW - 6 // content area inside outer panel

	// Panel widths for side-by-side layout.
	sideBySide := w >= 80
	var panelW int
	if sideBySide {
		panelW = (outerInnerW - 2) / 2 // 2-col gutter
		if panelW < 24 {
			panelW = 24
		}
	} else {
		panelW = outerInnerW
		if panelW < 24 {
			panelW = 24
		}
	}
	// Inner content width inside each panel = panelW - border(2) - padding(2).
	bodyW := panelW - 4
	if bodyW < 16 {
		bodyW = 16
	}

	header := headerLine(pkg, accentColor, outerInnerW)

	infoPanel := innerPanel.Width(panelW).Render(infoBody(pkg, accentColor, bodyW))
	descPanel := innerPanel.Width(panelW).Render(descBody(m, accentColor, bodyW))
	depsPanel := innerPanel.Width(panelW).Render(depsInlineBody(pkg.DependsOn, "Depends on", accentColor, bodyW))
	reqByPanel := innerPanel.Width(panelW).Render(depsInlineBody(pkg.RequiredBy, "Required by", accentColor, bodyW))

	var row1, row2 string
	if sideBySide {
		row1 = lipgloss.JoinHorizontal(lipgloss.Top, infoPanel, "  ", descPanel)
		row2 = lipgloss.JoinHorizontal(lipgloss.Top, depsPanel, "  ", reqByPanel)
	} else {
		row1 = lipgloss.JoinVertical(lipgloss.Left, infoPanel, "", descPanel)
		row2 = lipgloss.JoinVertical(lipgloss.Left, depsPanel, "", reqByPanel)
	}

	// Optional update banner, sits between header and row1.
	blocks := []string{header, ""}
	if pkg.LatestVersion != "" && pkg.LatestVersion != pkg.Version {
		updateLine := fmt.Sprintf("↑ Update available: %s → %s", pkg.Version, pkg.LatestVersion)
		blocks = append(blocks, StyleUpdateBanner.Render(updateLine), "")
	}
	blocks = append(blocks, row1, "", row2)
	innerContent := lipgloss.JoinVertical(lipgloss.Left, blocks...)

	outerPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Render(innerContent)

	// Center the outer panel horizontally.
	outerWidth := lipgloss.Width(outerPanel)
	centerPad := (w - outerWidth) / 2
	if centerPad < 0 {
		centerPad = 0
	}
	pad := strings.Repeat(" ", centerPad)
	var centered strings.Builder
	lines := strings.Split(outerPanel, "\n")
	for i, line := range lines {
		centered.WriteString(pad)
		centered.WriteString(line)
		if i < len(lines)-1 {
			centered.WriteString("\n")
		}
	}
	content := centered.String()

	// App title + keybinds, both centered to the terminal width so they group
	// visually with the centered panel.
	title := StyleTitle.Render("GlazePKG")
	if m.updateBanner != "" {
		title += "  " + StyleUpdateBanner.Render(m.updateBanner)
	}
	title = lipgloss.PlaceHorizontal(w, lipgloss.Center, title)
	keybinds := lipgloss.PlaceHorizontal(w, lipgloss.Center, detailKeybinds(m))

	// Stack title + panel + keybinds into one block with a 1-row gap around
	// the panel, then vertically center the block inside the terminal.
	block := lipgloss.JoinVertical(lipgloss.Left, title, "", content, "", keybinds)
	blockHeight := lipgloss.Height(block)
	topFill := (h - blockHeight) / 2
	if topFill < 0 {
		topFill = 0
	}

	var out strings.Builder
	if topFill > 0 {
		out.WriteString(strings.Repeat("\n", topFill))
	}
	out.WriteString(block)

	return out.String()
}

// headerLine renders "← <name>" left-aligned with the source badge right-aligned.
func headerLine(pkg model.Package, accent lipgloss.TerminalColor, width int) string {
	nameStyle := lipgloss.NewStyle().Foreground(accent).Bold(true)
	name := nameStyle.Render("← " + pkg.Name)
	badge := RenderBadge(pkg.Source)

	gap := width - lipgloss.Width(name) - lipgloss.Width(badge)
	if gap < 2 {
		gap = 2
	}
	return name + strings.Repeat(" ", gap) + badge
}

// infoBody renders the metadata field list inside the Info panel.
func infoBody(pkg model.Package, accent lipgloss.TerminalColor, bodyW int) string {
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("Info")

	keyStyle := lipgloss.NewStyle().Foreground(ColorSubtext).Width(11)
	valStyle := lipgloss.NewStyle().Foreground(ColorText)

	valW := bodyW - 11
	if valW < 8 {
		valW = 8
	}

	fields := []struct {
		key string
		val string
	}{
		{"Version", pkg.Version},
		{"Source", formatSource(pkg)},
		{"Installed", formatInstalled(pkg)},
		{"Location", pkg.Location},
		{"Size", pkg.Size},
	}

	var rows []string
	for _, f := range fields {
		if f.val == "" {
			continue
		}
		val := truncateToWidth(f.val, valW)
		rows = append(rows, keyStyle.Render(f.key)+valStyle.Render(val))
	}
	if len(rows) == 0 {
		rows = append(rows, StyleDim.Render("(no metadata)"))
	}

	return title + "\n" + strings.Join(rows, "\n")
}

// descBody renders the description (or the edit input when editing).
func descBody(m *Model, accent lipgloss.TerminalColor, bodyW int) string {
	pkg := m.detailPkg
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("Description")
	if m.editingDesc {
		return title + "\n" + m.descInput.View()
	}
	body := pkg.Description
	if body == "" {
		body = StyleDim.Render("(none) — press e to add")
	} else {
		body = lipgloss.NewStyle().Foreground(ColorText).Width(bodyW).Render(body)
	}
	return title + "\n" + body
}

// depsInlineBody renders a "Depends on (N)" or "Required by (N)" section
// with the first handful of items listed inline. Shows "—" for empty lists.
func depsInlineBody(items []string, label string, accent lipgloss.TerminalColor, bodyW int) string {
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render(fmt.Sprintf("%s (%d)", label, len(items)))
	if len(items) == 0 {
		return title + "\n" + StyleDim.Render("—")
	}

	const maxVisible = 5
	visible := items
	if len(visible) > maxVisible {
		visible = visible[:maxVisible]
	}
	valStyle := lipgloss.NewStyle().Foreground(ColorText)

	var lines []string
	for _, item := range visible {
		lines = append(lines, valStyle.Render(truncateToWidth(item, bodyW)))
	}
	body := strings.Join(lines, "\n")
	if len(items) > maxVisible {
		body += "\n" + StyleDim.Render(fmt.Sprintf("…and %d more", len(items)-maxVisible))
	}
	return title + "\n" + body
}

// detailKeybinds returns the bottom keybind hint bar used in the detail view.
// Adapts to editingDesc / modal state so it replaces the status-bar hints
// entirely for viewDetail.
func detailKeybinds(m *Model) string {
	keyStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(ColorText)

	var pairs []struct{ key, desc string }
	switch {
	case m.editingDesc:
		pairs = []struct{ key, desc string }{
			{"enter", "save"}, {"esc", "cancel"},
		}
	case m.modal == ModalPkgHelp:
		pairs = []struct{ key, desc string }{
			{"j/k", "scroll"}, {"pgdn/pgup", "page"}, {"esc", "close"},
		}
	case m.modal == ModalDeps:
		pairs = []struct{ key, desc string }{
			{"j/k", "navigate"}, {"esc", "close"},
		}
	default:
		// Mirror the capability checks the key handler uses, so we don't
		// advertise keys that the current package can't actually act on:
		// hide u/x when the manager lacks the interface OR is currently
		// unavailable (the handler would otherwise bail out with a
		// "not available" status message), and hide d when there is no
		// dependency data to show.
		if mgr := manager.BySource(m.detailPkg.Source); mgr != nil && mgr.Available() {
			if _, ok := mgr.(manager.Upgrader); ok {
				pairs = append(pairs, struct{ key, desc string }{"u", "upgrade"})
			}
			if _, ok := mgr.(manager.Remover); ok {
				pairs = append(pairs, struct{ key, desc string }{"x", "remove"})
			}
		}
		pairs = append(pairs, struct{ key, desc string }{"e", "edit"})
		if len(m.detailPkg.DependsOn) > 0 || len(m.detailPkg.RequiredBy) > 0 {
			pairs = append(pairs, struct{ key, desc string }{"d", "deps"})
		}
		pairs = append(pairs,
			struct{ key, desc string }{"h", "help"},
			struct{ key, desc string }{"esc", "back"},
			struct{ key, desc string }{"q", "quit"},
		)
	}

	var parts []string
	for _, p := range pairs {
		parts = append(parts, keyStyle.Render(p.key)+descStyle.Render(" "+p.desc))
	}
	return strings.Join(parts, "   ")
}

// truncateToWidth clips a plain string (no ANSI) to n columns, appending an
// ellipsis if truncated. Safe for field values that don't contain styling.
func truncateToWidth(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	// Byte-based truncation is fine for the values we pass (paths, versions,
	// simple identifiers). Multibyte characters are rare here.
	runes := []rune(s)
	if len(runes) > n-1 {
		runes = runes[:n-1]
	}
	return string(runes) + "…"
}

func formatSource(pkg model.Package) string {
	if pkg.Repository != "" {
		return fmt.Sprintf("%s (%s)", pkg.Source, pkg.Repository)
	}
	return string(pkg.Source)
}

func formatInstalled(pkg model.Package) string {
	if pkg.InstalledAt.IsZero() {
		return ""
	}
	return pkg.InstalledAt.Format("2006-01-02")
}

func formatList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, ", ")
}

func formatListShort(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) <= 3 {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s  +%d more", strings.Join(items[:3], ", "), len(items)-3)
}

// depsBody returns the scrollable body content for the dependencies modal.
// Pure content — the modal frame owns the title and outer border.
func depsBody(m *Model) string {
	pkg := m.detailPkg
	cursor := m.depsCursor
	height := m.height

	var b strings.Builder

	hasDeps := len(pkg.DependsOn) > 0
	hasReqBy := len(pkg.RequiredBy) > 0
	total := len(pkg.DependsOn) + len(pkg.RequiredBy)

	if total == 0 {
		b.WriteString(StyleDim.Render("  No dependencies"))
		return b.String()
	}

	maxVisible := height - 12
	if maxVisible < 5 {
		maxVisible = 5
	}
	if maxVisible > total {
		maxVisible = total
	}

	// Adjust scroll window
	start := 0
	if cursor >= maxVisible {
		start = cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > total {
		end = total
		start = max(0, end-maxVisible)
	}

	// Render "Depends on" section
	if hasDeps {
		b.WriteString(StyleDetailKey.Render(fmt.Sprintf("  Depends on (%d)", len(pkg.DependsOn))))
		b.WriteString("\n")
	}

	for i := start; i < end && i < len(pkg.DependsOn); i++ {
		name := pkg.DependsOn[i]
		if i == cursor {
			b.WriteString(StyleSelected.Render(fmt.Sprintf("  ▸ %-44s", name)))
		} else {
			b.WriteString("  ")
			b.WriteString(StyleDetailVal.Render(fmt.Sprintf("  %-44s", name)))
		}
		b.WriteString("\n")
	}

	// Render "Required by" section
	if hasReqBy {
		// Show header if any required-by items are visible
		reqStart := max(0, start-len(pkg.DependsOn))
		reqEnd := end - len(pkg.DependsOn)
		if reqEnd > 0 {
			b.WriteString("\n")
			b.WriteString(StyleDetailKey.Render(fmt.Sprintf("  Required by (%d)", len(pkg.RequiredBy))))
			b.WriteString("\n")

			if reqStart < 0 {
				reqStart = 0
			}
			for i := reqStart; i < reqEnd && i < len(pkg.RequiredBy); i++ {
				globalIdx := len(pkg.DependsOn) + i
				name := pkg.RequiredBy[i]
				if globalIdx == cursor {
					b.WriteString(StyleSelected.Render(fmt.Sprintf("  ▸ %-44s", name)))
				} else {
					b.WriteString("  ")
					b.WriteString(StyleDetailVal.Render(fmt.Sprintf("  %-44s", name)))
				}
				b.WriteString("\n")
			}
		}
	}

	// Scroll indicator
	b.WriteString("\n")
	indicator := fmt.Sprintf("  %d/%d", cursor+1, total)
	b.WriteString(StyleDim.Render(indicator))

	return b.String()
}

// pkgHelpBody returns the visible slice of m.pkgHelpLines starting at
// m.pkgHelpScroll, styled with heading/flag/normal colors, truncated to a
// reasonable width. Pure content — no framing, no overlay.
func pkgHelpBody(m *Model) string {
	lines := m.pkgHelpLines
	scroll := m.pkgHelpScroll
	if len(lines) == 0 {
		return StyleDim.Render("No help available")
	}

	// Match old behavior: cap visible lines to terminal height minus modal chrome.
	visibleLines := m.height - 10
	if visibleLines < 5 {
		visibleLines = 5
	}

	// Content width: cap at 100 cols to match old "min(120, width-10)" intent
	// while leaving room for frame border + padding.
	contentWidth := m.width - 10
	if contentWidth > 100 {
		contentWidth = 100
	}
	if contentWidth < 30 {
		contentWidth = 30
	}

	end := scroll + visibleLines
	if end > len(lines) {
		end = len(lines)
	}

	headingStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	flagStyle := lipgloss.NewStyle().Foreground(ColorGreen)
	normalStyle := lipgloss.NewStyle().Foreground(ColorText)

	var b strings.Builder
	for i := scroll; i < end; i++ {
		line := lines[i]
		if len(line) > contentWidth {
			line = line[:contentWidth-1] + "…"
		}
		trimmed := strings.TrimSpace(line)

		// Style classification: cyan bold for headings, green for flags, normal for body.
		var styled string
		switch {
		case trimmed == "":
			styled = line
		case isHelpHeading(trimmed):
			styled = headingStyle.Render(line)
		case strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "--"):
			styled = flagStyle.Render(line)
		default:
			styled = normalStyle.Render(line)
		}
		b.WriteString(styled)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Scroll indicator at the bottom if more content exists.
	if end < len(lines) {
		b.WriteString("\n")
		b.WriteString(StyleDim.Render(fmt.Sprintf("── %d more lines (j/k to scroll) ──", len(lines)-end)))
	}

	return b.String()
}

// isHelpHeading detects section headings in help output.
func isHelpHeading(line string) bool {
	if len(line) == 0 {
		return false
	}
	// "USAGE:", "OPTIONS:", "COMMANDS:", etc.
	if strings.ToUpper(line) == line && strings.HasSuffix(line, ":") {
		return true
	}
	// "Usage:", "Options:", "Commands:", etc.
	if line[0] >= 'A' && line[0] <= 'Z' && strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
		return true
	}
	// Common patterns
	upper := strings.ToUpper(line)
	for _, heading := range []string{"USAGE", "OPTIONS", "COMMANDS", "FLAGS", "ARGUMENTS", "EXAMPLES", "DESCRIPTION", "SYNOPSIS", "SUBCOMMANDS", "GLOBAL OPTIONS", "AVAILABLE COMMANDS"} {
		if strings.HasPrefix(upper, heading) {
			return true
		}
	}
	return false
}
