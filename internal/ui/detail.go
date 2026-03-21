package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/model"
)

func renderDetail(pkg model.Package, editing bool, descInput string) string {
	var b strings.Builder

	// Header
	title := fmt.Sprintf("  ← %s", pkg.Name)
	badge := RenderBadge(pkg.Source)
	b.WriteString(StyleNormal.Bold(true).Render(title))
	b.WriteString(strings.Repeat(" ", max(2, 60-len(title)-8)))
	b.WriteString(badge)
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", 75)))
	b.WriteString("\n\n")

	hasUpdate := pkg.LatestVersion != "" && pkg.LatestVersion != pkg.Version

	// Fields
	fields := []struct {
		key string
		val string
	}{
		{"Version", pkg.Version},
		{"Source", formatSource(pkg)},
		{"Installed", formatInstalled(pkg)},
		{"Location", pkg.Location},
		{"Size", pkg.Size},
		{"Depends on", formatListShort(pkg.DependsOn)},
		{"Required by", formatListShort(pkg.RequiredBy)},
	}

	for _, f := range fields {
		if f.val == "" {
			continue
		}
		b.WriteString("  ")
		b.WriteString(StyleDetailKey.Render(f.key))
		b.WriteString(StyleDetailVal.Render(f.val))
		b.WriteString("\n")
	}

	// Update available banner
	if hasUpdate {
		b.WriteString("\n")
		updateLine := fmt.Sprintf("  ↑ Update available: %s → %s", pkg.Version, pkg.LatestVersion)
		b.WriteString(StyleUpdateBanner.Render(updateLine))
		b.WriteString("\n")
	}

	// Description field (always shown)
	if editing {
		b.WriteString("  ")
		b.WriteString(descInput)
		b.WriteString("\n")
	} else if pkg.Description != "" {
		b.WriteString("  ")
		b.WriteString(StyleDetailKey.Render("Description"))
		b.WriteString(StyleDetailVal.Render(pkg.Description))
		b.WriteString("\n")
	} else {
		b.WriteString("  ")
		b.WriteString(StyleDetailKey.Render("Description"))
		b.WriteString(StyleDim.Render("(none) — press e to add"))
		b.WriteString("\n")
	}

	return b.String()
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

func renderDepsOverlay(pkg model.Package, cursor, width, height int) string {
	var b strings.Builder

	// Title
	title := fmt.Sprintf("  Dependencies — %s", pkg.Name)
	b.WriteString(StyleOverlayTitle.Render(title))
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", 46)))
	b.WriteString("\n")

	hasDeps := len(pkg.DependsOn) > 0
	hasReqBy := len(pkg.RequiredBy) > 0
	total := len(pkg.DependsOn) + len(pkg.RequiredBy)

	if total == 0 {
		b.WriteString("\n")
		b.WriteString(StyleDim.Render("  No dependencies"))
		b.WriteString("\n")
		return renderDepsOverlayFrame(b.String(), width, height, 6)
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
		b.WriteString("\n")
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

	overlayHeight := min(maxVisible+10, height-4)
	return renderDepsOverlayFrame(b.String(), width, height, overlayHeight)
}

func renderDepsOverlayFrame(content string, width, height, overlayHeight int) string {
	overlayWidth := 54

	overlay := StyleOverlay.
		Width(overlayWidth).
		Height(overlayHeight).
		Render(content)

	return placeOverlay(width, height, overlay)
}

func renderPkgHelpOverlay(name string, lines []string, scroll, width, height int) string {
	var b strings.Builder

	overlayWidth := width - 10
	if overlayWidth > 120 {
		overlayWidth = 120
	}
	if overlayWidth < 40 {
		overlayWidth = 40
	}
	contentWidth := overlayWidth - 6

	// Title
	title := fmt.Sprintf("  %s --help", name)
	b.WriteString(StyleOverlayTitle.Render(title))
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", min(contentWidth, overlayWidth-4))))
	b.WriteString("\n")

	if len(lines) == 0 {
		b.WriteString("\n")
		b.WriteString(StyleDim.Render("  No help available"))
		b.WriteString("\n")
	} else {
		visibleLines := height - 10
		if visibleLines < 5 {
			visibleLines = 5
		}

		end := scroll + visibleLines
		if end > len(lines) {
			end = len(lines)
		}

		headingStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
		flagStyle := lipgloss.NewStyle().Foreground(ColorGreen)
		normalStyle := lipgloss.NewStyle().Foreground(ColorText)

		for i := scroll; i < end; i++ {
			line := lines[i]
			// Truncate long lines
			if len(line) > contentWidth {
				line = line[:contentWidth-1] + "…"
			}

			trimmed := strings.TrimSpace(line)

			// Style based on content
			var styled string
			switch {
			case trimmed == "":
				styled = ""
			case isHelpHeading(trimmed):
				styled = headingStyle.Render("  " + line)
			case strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "--"):
				styled = "  " + flagStyle.Render(line)
			default:
				styled = "  " + normalStyle.Render(line)
			}

			b.WriteString(styled)
			b.WriteString("\n")
		}

		// Scroll indicator
		if len(lines) > visibleLines {
			pct := (scroll + visibleLines) * 100 / len(lines)
			if pct > 100 {
				pct = 100
			}
			indicator := fmt.Sprintf("  ─── %d%% ───", pct)
			b.WriteString(StyleDim.Render(indicator))
		}
	}

	overlayHeight := min(height-4, len(lines)+6)
	if overlayHeight < 8 {
		overlayHeight = 8
	}
	if overlayHeight > height-4 {
		overlayHeight = height - 4
	}

	overlay := StyleOverlay.
		Width(overlayWidth).
		Height(overlayHeight).
		Render(b.String())

	return placeOverlay(width, height, overlay)
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
