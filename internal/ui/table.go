package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/model"
)

const badgeWidth = 8 // fixed visual width for all manager badges

func renderPackageTable(pkgs []model.Package, cursor, height, width int, showSize bool, upgradingPkg, removingPkg string, selections map[string]bool) string {
	if len(pkgs) == 0 {
		return StyleDim.Render("\n  No packages found.")
	}

	// Column widths from terminal width
	usable := width - 4
	if usable < 60 {
		usable = 60
	}

	// Fixed columns: 2 padding + name + 2 gap + version + 2 gap + badge(8) + 2 gap + desc
	colName := usable * 25 / 100
	colVer := usable * 12 / 100
	colBadge := badgeWidth + 2 // badge + gaps
	colDesc := usable - colName - colVer - colBadge

	if colName < 15 {
		colName = 15
	}
	if colVer < 10 {
		colVer = 10
	}
	if colDesc < 10 {
		colDesc = 10
	}

	// Header — pad each header label to column width using plain strings
	lastCol := "DESCRIPTION"
	if showSize {
		lastCol = "SIZE"
	}
	header := "  " +
		padCell(StyleTableHeader.Render("PACKAGE"), colName) +
		padCell(StyleTableHeader.Render("VERSION"), colVer) +
		padCell(StyleTableHeader.Render("MANAGER"), colBadge) +
		StyleTableHeader.Render(lastCol)

	// Scrolling viewport
	visibleHeight := height - 4
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	start := 0
	if cursor >= visibleHeight {
		start = cursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(pkgs) {
		end = len(pkgs)
	}

	var lines []string
	lines = append(lines, header)
	lines = append(lines, StyleDim.Render("  "+strings.Repeat("─", min(usable, width-4))))

	updateIndicator := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	sizeStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)

	upgradingStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)

	for i := start; i < end; i++ {
		p := pkgs[i]
		nameStr := p.Name
		if len(selections) > 0 {
			if selections[p.Key()] {
				nameStr = "☑ " + nameStr
			} else {
				nameStr = "☐ " + nameStr
			}
		}
		name := truncate(nameStr, colName-2)
		hasUpdate := p.LatestVersion != "" && p.LatestVersion != p.Version
		ver := truncate(p.Version, colVer-4)
		badge := renderFixedBadge(p.Source)
		isUpgrading := upgradingPkg != "" && p.Name == upgradingPkg
		isRemoving := removingPkg != "" && p.Name == removingPkg

		// Last column: show size when filtering by size, otherwise description
		var lastText string
		if showSize {
			lastText = p.Size
		} else {
			lastText = p.Description
			if len(p.RequiredBy) > 0 {
				reqBy := strings.Join(p.RequiredBy, ", ")
				if lastText != "" {
					lastText = "req by " + reqBy + " — " + lastText
				} else {
					lastText = "req by " + reqBy
				}
			}
		}
		desc := truncate(lastText, colDesc-1)

		if i == cursor {
			verCell := StyleSelected.Render(ver)
			if hasUpdate {
				verCell += " " + updateIndicator.Render("↑")
			}
			lastCell := StyleSelected.Render(desc)
			if showSize && p.Size != "" {
				lastCell = sizeStyle.Render(desc)
			}
			line := "  " +
				padCell(StyleSelected.Render(name), colName) +
				padCell(verCell, colVer) +
				padCell(badge, colBadge) +
				lastCell
			lines = append(lines, line)
		} else if isUpgrading {
			verCell := upgradingStyle.Render(ver)
			if hasUpdate {
				verCell += " " + upgradingStyle.Render("↑")
			}
			lastCell := upgradingStyle.Render(desc)
			line := "  " +
				padCell(upgradingStyle.Render("▸ "+name), colName) +
				padCell(verCell, colVer) +
				padCell(badge, colBadge) +
				lastCell
			lines = append(lines, line)
		} else if isRemoving {
			removingStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
			verCell := removingStyle.Render(ver)
			lastCell := removingStyle.Render(desc)
			line := "  " +
				padCell(removingStyle.Render("✗ "+name), colName) +
				padCell(verCell, colVer) +
				padCell(badge, colBadge) +
				lastCell
			lines = append(lines, line)
		} else {
			verCell := StyleDim.Render(ver)
			if hasUpdate {
				verCell += " " + updateIndicator.Render("↑")
			}
			lastCell := StyleDim.Render(desc)
			if showSize && p.Size != "" {
				lastCell = sizeStyle.Render(desc)
			}
			line := "  " +
				padCell(StyleNormal.Render(name), colName) +
				padCell(verCell, colVer) +
				padCell(badge, colBadge) +
				lastCell
			lines = append(lines, line)
		}
	}

	// Scroll indicator
	total := len(pkgs)
	if total > visibleHeight {
		pct := (cursor + 1) * 100 / total
		indicator := fmt.Sprintf("  %d/%d (%d%%)", cursor+1, total, pct)
		lines = append(lines, StyleDim.Render(indicator))
	}

	return strings.Join(lines, "\n")
}

// padCell pads a styled string to exact visual width.
func padCell(s string, width int) string {
	vis := lipgloss.Width(s)
	if vis >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vis)
}

// renderFixedBadge returns a badge padded to badgeWidth visual characters.
func renderFixedBadge(source model.Source) string {
	color, ok := ManagerColors[source]
	if !ok {
		color = ColorSubtext
	}
	label := string(source)
	// Center the label within badgeWidth by padding inside the badge
	inner := badgeWidth - 2 // subtract the 1-char padding on each side from StyleBadge
	if len(label) < inner {
		pad := inner - len(label)
		left := pad / 2
		right := pad - left
		label = strings.Repeat(" ", left) + label + strings.Repeat(" ", right)
	}
	return StyleBadge.
		Foreground(lipgloss.Color("#1a1b26")).
		Background(color).
		Render(label)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "."
	}
	return s[:max-1] + "…"
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		return t.Format("2006-01-02")
	}
}
