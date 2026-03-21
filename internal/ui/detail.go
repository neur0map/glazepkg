package ui

import (
	"fmt"
	"strings"

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
		{"Depends on", formatList(pkg.DependsOn)},
		{"Required by", formatList(pkg.RequiredBy)},
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
