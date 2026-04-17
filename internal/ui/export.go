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

// exportBody returns the two-row format picker with cursor highlight on
// m.exportCursor. Pure content — no title, no frame.
func exportBody(m *Model) string {
	var b strings.Builder
	for i, format := range exportFormats {
		prefix := "  "
		style := StyleNormal
		if i == m.exportCursor {
			prefix = StyleSelected.Render(" > ")
			style = StyleSelected
		}
		b.WriteString(prefix)
		b.WriteString(style.Render(format))
		if i < len(exportFormats)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
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
