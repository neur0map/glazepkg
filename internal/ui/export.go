package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

var exportFormats = []string{"Text", "JSON"}

func renderExportOverlay(cursor, width, height int) string {
	var b strings.Builder
	b.WriteString(StyleOverlayTitle.Render("  Export Packages"))
	b.WriteString(StyleOverlayBase.Render("\n"))
	b.WriteString(StyleOverlayBase.Copy().Foreground(ColorSubtext).Render("  " + strings.Repeat("─", 30)))
	b.WriteString(StyleOverlayBase.Render("\n\n"))

	for i, format := range exportFormats {
		prefix := StyleOverlayBase.Render("  ")
		style := StyleOverlayBase.Copy().Foreground(ColorText)
		if i == cursor {
			prefix = StyleSelected.Render(" > ")
			style = StyleSelected
		}
		b.WriteString(prefix)
		b.WriteString(style.Render(format))
		b.WriteString(StyleOverlayBase.Render("\n"))
	}

	b.WriteString(StyleOverlayBase.Render("\n"))
	b.WriteString(StyleOverlayBase.Copy().Foreground(ColorSubtext).Render("  Enter: export  Esc: cancel"))

	content := b.String()
	overlay := StyleOverlay.
		Width(36).
		Height(8).
		Render(content)

	return placeOverlay(width, height, overlay)
}

func exportPackages(pkgs []model.Package, format int) (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(base, "glazepkg", "exports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05")

	switch format {
	case 1: // JSON
		path := filepath.Join(dir, timestamp+".json")
		data, err := json.MarshalIndent(pkgs, "", "  ")
		if err != nil {
			return "", err
		}
		return path, os.WriteFile(path, data, 0o644)
	default: // Text
		path := filepath.Join(dir, timestamp+".txt")
		groups := make(map[string][]model.Package)
		for _, p := range pkgs {
			groups[string(p.Source)] = append(groups[string(p.Source)], p)
		}
		sources := make([]string, 0, len(groups))
		for s := range groups {
			sources = append(sources, s)
		}
		sort.Strings(sources)

		var b strings.Builder
		for _, source := range sources {
			fmt.Fprintf(&b, "# %s\n", source)
			for _, p := range groups[source] {
				if p.Version != "" {
					fmt.Fprintf(&b, "%s==%s\n", p.Name, p.Version)
				} else {
					fmt.Fprintln(&b, p.Name)
				}
			}
			b.WriteString("\n")
		}
		return path, os.WriteFile(path, []byte(b.String()), 0o644)
	}
}
