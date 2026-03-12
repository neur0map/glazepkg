package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/neur0map/glazepkg/internal/ui"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help") {
		printHelp()
		return
	}

	p := tea.NewProgram(ui.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	help := `GlazePKG (gpk) — eye-candy package viewer

Usage:
  gpk              Launch TUI
  gpk --help       Show this help

Keybinds:
  j/k, ↑/↓         Navigate up/down
  g/G               Jump to top/bottom
  Ctrl+d/u          Half page down/up
  PgDn/PgUp         Page down/up
  Tab/Shift+Tab     Cycle manager tabs
  /                 Fuzzy search
  Esc               Clear search / close overlay
  Enter             Package details
  r                 Rescan all managers
  s                 Save snapshot
  d                 Diff against last snapshot
  e                 Export packages
  ?                 Toggle help overlay
  q                 Quit

Supported managers:
  brew, pacman, aur, apt, dnf, snap, pip, pipx,
  cargo, go, npm, bun, flatpak

Data:
  Snapshots   ~/.local/share/glazepkg/snapshots/
  Exports     ~/.local/share/glazepkg/exports/
  Desc cache  ~/.local/share/glazepkg/cache/`

	fmt.Println(help)
}
