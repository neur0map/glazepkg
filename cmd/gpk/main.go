package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/neur0map/glazepkg/internal/theme"
	"github.com/neur0map/glazepkg/internal/ui"
	"github.com/neur0map/glazepkg/internal/updater"
)

const repoURL = "https://github.com/neur0map/glazepkg"

// version holds the current version, resolved at startup from the latest GitHub release tag.
var version = fetchLatestVersion()

// fetchLatestVersion queries the GitHub Releases API for the latest release tag.
// Falls back to "dev" if the request fails or the response cannot be parsed.
func fetchLatestVersion() string {
	const apiURL = "https://api.github.com/repos/neur0map/glazepkg/releases/latest"

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "dev"
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "glazepkg")

	resp, err := client.Do(req)
	if err != nil {
		return "dev"
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return "dev"
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil || release.TagName == "" {
		return "dev"
	}

	return strings.TrimPrefix(release.TagName, "v")
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h", "help":
			printHelp()
			return
		case "--version", "-v", "version":
			fmt.Printf("gpk %s\n", version)
			return
		case "update":
			runUpdate()
			return
		case "themes":
			runListThemes()
			return
		case "--theme", "-T":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: gpk --theme <name>")
				os.Exit(1)
			}
			runSetTheme(strings.Join(os.Args[2:], " "))
			return
		case "snapshot":
			runSnapshot(os.Args[2:])
			return
		}
	}

	// Launch TUI immediately
	p := tea.NewProgram(ui.NewModel(version), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// runListThemes loads the theme registry and prints all available names
func runListThemes() {
	if err := theme.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "theme load error: %v\n", err)
		os.Exit(1)
	}
	active := theme.Active()
	names := theme.ListThemes()
	fmt.Printf("Available themes (%d):\n\n", len(names))
	for _, name := range names {
		t, _ := theme.GetTheme(name)
		marker := "  "
		if name == active.Name {
			marker = "✓ "
		}
		typeSuffix := ""
		if t.Type != "" {
			typeSuffix = " [" + t.Type + "]"
		}
		desc := ""
		if t.Description != "" {
			desc = "  — " + t.Description
		}
		fmt.Printf("  %s%s%s%s\n", marker, name, typeSuffix, desc)
	}
	fmt.Printf("\nActive: %s\n", active.Name)
	fmt.Println("\nTo change: gpk --theme \"<name>\"  or press 't' in the TUI")
}

// runSetTheme applies the named theme, persists the preference, then launches the TUI
func runSetTheme(name string) {
	if err := theme.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "theme load error: %v\n", err)
		os.Exit(1)
	}
	if _, err := theme.SetActive(name); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nRun `gpk themes` to list available theme names.\n")
		os.Exit(1)
	}
	fmt.Printf("Theme set to: %s\n", name)
	fmt.Println("Launching TUI...")

	p := tea.NewProgram(ui.NewModel(version), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// runSnapshot handles the 'gpk snapshot' subcommand family
func runSnapshot(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: gpk snapshot <save [label] | list | load <path>>")
		os.Exit(1)
	}
	if err := theme.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "theme load error: %v\n", err)
		os.Exit(1)
	}
	switch args[0] {
	case "save":
		label := "manual"
		if len(args) > 1 {
			label = strings.Join(args[1:], "_")
		}
		if err := theme.SaveSnapshot(label); err != nil {
			fmt.Fprintf(os.Stderr, "snapshot save error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Theme snapshot saved: %s\n", label)
	case "list":
		paths, err := theme.ListSnapshots()
		if err != nil {
			fmt.Fprintf(os.Stderr, "snapshot list error: %v\n", err)
			os.Exit(1)
		}
		if len(paths) == 0 {
			fmt.Println("No theme snapshots found.")
			return
		}
		fmt.Printf("Theme snapshots (%d):\n\n", len(paths))
		for _, p := range paths {
			fmt.Printf("  %s\n", p)
		}
		fmt.Println("\nTo restore: gpk snapshot load <path>")
	case "load":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: gpk snapshot load <path>")
			os.Exit(1)
		}
		path := args[1]
		if err := theme.LoadSnapshot(path); err != nil {
			fmt.Fprintf(os.Stderr, "snapshot load error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Snapshot loaded and applied.\n")
		fmt.Println("Launching TUI...")
		p := tea.NewProgram(ui.NewModel(version), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown snapshot command: %s\n", args[0])
		os.Exit(1)
	}
}

// runUpdate calls updater.Update with the current version
func runUpdate() {
	fmt.Printf("gpk %s — checking for updates...\n", version)
	newVersion, err := updater.Update(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("updated: %s → %s\n", version, newVersion)
}

// printHelp prints CLI usage
func printHelp() {
	help := `GlazePKG (gpk) — eye-candy package viewer

Usage:
  gpk                        Launch TUI
  gpk update                 Self-update to latest release
  gpk version                Show current version
  gpk themes                 List all available themes
  gpk --theme <name>         Set active theme and launch TUI
  gpk snapshot save [label]  Save a theme snapshot
  gpk snapshot list          List saved theme snapshots
  gpk snapshot load <path>   Restore a theme snapshot
  gpk --help                 Show this help

TUI Keybinds:
  j/k, ↑/↓         Navigate up/down
  g/G              Jump to top/bottom
  Ctrl+d/u         Half page down/up
  PgDn/PgUp        Page down/up
  Tab/Shift+Tab    Cycle manager tabs
  /                Fuzzy search
  Esc              Clear search / close overlay
  Enter            Package details
  u (detail)       Upgrade package
  e (detail)       Edit description
  d (detail)       View dependencies
  h (detail)       Package help/usage
  f                Cycle size filter
  r                Rescan all managers
  s                Save package snapshot
  d                Diff against last snapshot
  e                Export packages
  t                Open theme picker / cycle themes
  ?                Toggle help overlay
  q                Quit

Theme Picker (t):
  j/k              Navigate theme list
  Enter            Apply selected theme
  t                Cycle to next theme (live preview)
  Esc              Close without changing

Data:
  Config      ~/.glazepkg/config.toml
  Themes      ~/.glazepkg/themes/      (custom themes)
  Snapshots   ~/.glazepkg/theme_snapshots/
  Pkg snaps   ~/.local/share/glazepkg/snapshots/
  Exports     ~/.local/share/glazepkg/exports/`

	fmt.Println(help)
}
