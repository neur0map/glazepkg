package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"

	"github.com/neur0map/glazepkg/internal/cli"
	"github.com/neur0map/glazepkg/internal/manager"
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
		// Any other first arg is treated as a subcommand attempt.
		// cli.Dispatch handles unknown names with a clean error + ExitErr,
		// so users typing `gpk install foo` get a real message instead of
		// the TUI failing with a TTY error.
		os.Exit(cli.Dispatch(os.Args[1:], manager.All(), version, os.Stdout, os.Stderr, os.Stdin))
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
	tty := isatty.IsTerminal(os.Stdout.Fd())
	var bold, cyan, yellow, dim, reset string
	if tty {
		bold = "\033[1m"
		cyan = "\033[36m"
		yellow = "\033[33m"
		dim = "\033[2m"
		reset = "\033[0m"
	}

	section := func(name string) string {
		return bold + name + reset
	}
	cmd := func(name string) string {
		return cyan + name + reset
	}
	flagText := func(s string) string {
		return yellow + s + reset
	}
	muted := func(s string) string {
		return dim + s + reset
	}

	fmt.Printf("%s · %s\n", bold+"gpk"+reset, muted("eye-candy package viewer"))
	fmt.Println()

	fmt.Println(section("USAGE"))
	fmt.Printf("  %-22s %s\n", cmd("gpk"), "Launch the TUI")
	fmt.Printf("  %-22s %s\n", cmd("gpk <subcommand>"), "Run a headless command")
	fmt.Printf("  %-22s %s\n", cmd("gpk update"), "Self-update to latest release")
	fmt.Printf("  %-22s %s\n", cmd("gpk version"), "Show current version")
	fmt.Printf("  %-22s %s\n", cmd("gpk -h, --help"), "Show this help")
	fmt.Println()

	fmt.Printf("%s %s\n", section("HEADLESS"), muted("· read-only"))
	fmt.Printf("  %-22s %s\n", cmd("list"), "List installed packages across all managers")
	fmt.Printf("  %-22s %s\n", cmd("installed <pkg>..."), "Check if packages are installed (exit 0/2)")
	fmt.Printf("  %-22s %s\n", cmd("info <pkg>"), "Show details for one installed package")
	fmt.Printf("  %-22s %s\n", cmd("source-of <pkg>"), "Print which manager has the package")
	fmt.Printf("  %-22s %s\n", cmd("outdated"), "List packages with available updates")
	fmt.Println()

	fmt.Printf("%s %s\n", section("HEADLESS"), muted("· write"))
	fmt.Printf("  %-22s %s\n", cmd("install <pkg>..."), "Install one or more packages")
	fmt.Printf("  %-22s %s\n", cmd("remove <pkg>..."), "Remove a package (--with-deps for orphans)")
	fmt.Printf("  %-22s %s\n", cmd("upgrade <pkg>..."), "Upgrade installed packages to latest")
	fmt.Println()

	fmt.Println(section("COMMON FLAGS"))
	fmt.Printf("  %s   filter manager (e.g. pacman, pacman,aur, !brew)\n", flagText("--manager M, -m"))
	fmt.Printf("  %s              emit a JSON envelope on stdout\n", flagText("--json"))
	fmt.Printf("  %s          bypass the scan/update cache\n", flagText("--no-cache"))
	fmt.Printf("  %s         skip the y/N prompt for writes (also skips manager's prompt)\n", flagText("--yes, -y"))
	fmt.Printf("  %s           print the command without running it (writes only)\n", flagText("--dry-run"))
	fmt.Printf("  %s       suppress progress on stderr\n", flagText("--quiet, -q"))
	fmt.Printf("  %s\n", muted("Run `gpk <subcommand> --help` for the full per-command flag list."))
	fmt.Println()

	fmt.Println(section("EXIT CODES"))
	fmt.Printf("  %s  %s\n", yellow+"0"+reset, "success / clean / yes")
	fmt.Printf("  %s  %s\n", yellow+"1"+reset, "error (bad flag, scan failed, IO)")
	fmt.Printf("  %s  %s\n", yellow+"2"+reset, "meaningful 'no' (not installed, has updates, not found)")
	fmt.Printf("  %s  %s\n", yellow+"3"+reset, "ambiguous (package available in multiple managers; use --manager)")
	fmt.Println()

	fmt.Println(section("TUI KEYBINDS"))
	fmt.Printf("  %-18s %s\n", cmd("j/k, ↑/↓"), "Navigate up/down")
	fmt.Printf("  %-18s %s\n", cmd("g/G"), "Jump to top/bottom")
	fmt.Printf("  %-18s %s\n", cmd("Ctrl+d/u"), "Half page down/up")
	fmt.Printf("  %-18s %s\n", cmd("Tab/Shift+Tab"), "Cycle manager tabs")
	fmt.Printf("  %-18s %s\n", cmd("/"), "Fuzzy search")
	fmt.Printf("  %-18s %s\n", cmd("Enter"), "Package details")
	fmt.Printf("  %-18s %s\n", cmd("u / x (detail)"), "Upgrade / remove the focused package")
	fmt.Printf("  %-18s %s\n", cmd("i"), "Search + install across managers")
	fmt.Printf("  %-18s %s\n", cmd("s / d"), "Save snapshot / diff against last")
	fmt.Printf("  %-18s %s\n", cmd("e"), "Export packages")
	fmt.Printf("  %-18s %s\n", cmd("r"), "Rescan all managers")
	fmt.Printf("  %-18s %s\n", cmd("t"), "Theme picker")
	fmt.Printf("  %-18s %s\n", cmd("? / q"), "Toggle help / quit")
	fmt.Println()

	fmt.Printf("%s %s\n", section("SUPPORTED MANAGERS"), muted("(36)"))
	fmt.Println(muted("  brew, pacman, aur, apt, dnf, snap, pip, pipx, uv, cargo, go,"))
	fmt.Println(muted("  npm, pnpm, bun, flatpak, macports, pkgsrc, opam, gem, pkg,"))
	fmt.Println(muted("  composer, mas, apk, nix, conda, luarocks, xbps, portage, guix,"))
	fmt.Println(muted("  winget, chocolatey, nuget, powershell, windows-updates, scoop, maven"))
	fmt.Println()

	fmt.Println(section("DATA PATHS"))
	fmt.Printf("  %-12s %s\n", cmd("Cache"), muted("~/.local/share/glazepkg/cache/"))
	fmt.Printf("  %-12s %s\n", cmd("Snapshots"), muted("~/.local/share/glazepkg/snapshots/"))
	fmt.Printf("  %-12s %s\n", cmd("Exports"), muted("~/.local/share/glazepkg/exports/"))
}
