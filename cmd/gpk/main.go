package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/neur0map/glazepkg/internal/ui"
	"github.com/neur0map/glazepkg/internal/updater"
)

// Set via -ldflags at build time.
var version = "dev"

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
		}
	}

	p := tea.NewProgram(ui.NewModel(version), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runUpdate() {
	fmt.Printf("gpk %s — checking for updates...\n", version)

	newVersion, err := updater.Update(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("updated: %s → %s\n", version, newVersion)
}

func printHelp() {
	help := `GlazePKG (gpk) — eye-candy package viewer

Usage:
  gpk              Launch TUI
  gpk update       Self-update to latest release
  gpk version      Show current version
  gpk --help       Show this help

Keybinds:
  j/k, ↑/↓         Navigate up/down
  g/G              Jump to top/bottom
  Ctrl+d/u         Half page down/up
  PgDn/PgUp        Page down/up
  Tab/Shift+Tab    Cycle manager tabs
  /                Fuzzy search
  Esc              Clear search / close overlay
  Enter            Package details
  u (detail)       Upgrade package
  x (detail)       Remove package
  e (detail)       Edit description
  r                Rescan all managers
  s                Save snapshot
  i                Search + install packages
  d                Diff against last snapshot
  e                Export packages
  ?                Toggle help overlay
  q                Quit

Supported managers:
  brew, pacman, aur, apt, dnf, snap, pip, pipx,
  cargo, go, npm, bun, flatpak

Data:
  Snapshots   ~/.local/share/glazepkg/snapshots/
  Exports     ~/.local/share/glazepkg/exports/
  Desc cache  ~/.local/share/glazepkg/cache/`

	fmt.Println(help)
}
