package ui

import (
	"fmt"
	"strings"

	"github.com/neur0map/glazepkg/internal/model"
)

type tabItem struct {
	Label  string
	Source string // "" means ALL (excludes deps), specific source filters to that
	Count  int
}

// depSources are sources hidden from the ALL tab (none currently).
var depSources = map[model.Source]bool{}

func buildTabs(pkgs []model.Package) []tabItem {
	counts := make(map[string]int)
	allCount := 0
	for _, p := range pkgs {
		counts[string(p.Source)]++
		if !depSources[p.Source] {
			allCount++
		}
	}

	tabs := []tabItem{
		{Label: "ALL", Source: "", Count: allCount},
	}

	// Fixed order
	sources := []struct {
		source model.Source
		label  string
	}{
		{model.SourceBrew, "brew"},
		{model.SourcePacman, "pacman"},
		{model.SourceAUR, "aur"},
		{model.SourceApt, "apt"},
		{model.SourceDnf, "dnf"},
		{model.SourceSnap, "snap"},
		{model.SourcePip, "pip"},
		{model.SourcePipx, "pipx"},
		{model.SourceCargo, "cargo"},
		{model.SourceGo, "go"},
		{model.SourceNpm, "npm"},
		{model.SourcePnpm, "pnpm"},
		{model.SourceBun, "bun"},
		{model.SourceFlatpak, "flatpak"},
		{model.SourceMacPorts, "macports"},
		{model.SourcePkgsrc, "pkgsrc"},
		{model.SourceOpam, "opam"},
		{model.SourceGem, "gem"},
		{model.SourcePkg, "pkg"},
		{model.SourceComposer, "composer"},
		{model.SourceMas, "mas"},
		{model.SourceApk, "apk"},
		{model.SourceNix, "nix"},
		{model.SourceConda, "conda"},
		{model.SourceLuarocks, "luarocks"},
		{model.SourceXbps, "xbps"},
		{model.SourcePortage, "portage"},
		{model.SourceGuix, "guix"},
		{model.SourceWinget, "winget"},
		{model.SourceChocolatey, "choco"},
		{model.SourceNuget, "nuget"},
		{model.SourcePowerShell, "pwsh"},
		{model.SourceWindowsUpdates, "winupd"},
		{model.SourceScoop, "scoop"},
	}

	for _, s := range sources {
		if c, ok := counts[string(s.source)]; ok && c > 0 {
			tabs = append(tabs, tabItem{
				Label:  s.label,
				Source: string(s.source),
				Count:  c,
			})
		}
	}

	return tabs
}

func renderTabs(tabs []tabItem, active int) string {
	var parts []string
	for i, t := range tabs {
		label := fmt.Sprintf("%s (%d)", t.Label, t.Count)
		if i == active {
			parts = append(parts, StyleActiveTab.Render(label))
		} else {
			parts = append(parts, StyleInactiveTab.Render(label))
		}
	}
	return strings.Join(parts, "  ")
}
