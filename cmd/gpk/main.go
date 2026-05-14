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
	// cmd colors the name. Padding always goes INSIDE the color codes so
	// ANSI escapes don't count toward fmt.Printf width — keeps columns
	// aligned in TTY mode where bold/cyan would otherwise add ~10 bytes
	// per cell and break %-22s.
	cmd := func(name string, width int) string {
		return cyan + fmt.Sprintf("%-*s", width, name) + reset
	}
	flagText := func(s string, width int) string {
		return yellow + fmt.Sprintf("%-*s", width, s) + reset
	}
	muted := func(s string) string {
		return dim + s + reset
	}

	fmt.Printf("%s · %s\n", bold+"gpk"+reset, muted("eye-candy package viewer"))
	fmt.Println()

	fmt.Println(section("USAGE"))
	fmt.Printf("  %s %s\n", cmd("gpk", 22), "Launch the TUI")
	fmt.Printf("  %s %s\n", cmd("gpk <subcommand>", 22), "Run a headless command")
	fmt.Printf("  %s %s\n", cmd("gpk update", 22), "Self-update to latest release")
	fmt.Printf("  %s %s\n", cmd("gpk version", 22), "Show current version")
	fmt.Printf("  %s %s\n", cmd("gpk -h, --help", 22), "Show this help")
	fmt.Println()

	fmt.Printf("%s %s\n", section("HEADLESS"), muted("· read-only"))
	fmt.Printf("  %s %s\n", cmd("list", 22), "List installed packages across all managers")
	fmt.Printf("  %s %s\n", cmd("installed <pkg>...", 22), "Check if packages are installed (exit 0/2)")
	fmt.Printf("  %s %s\n", cmd("info <pkg>", 22), "Show details for one installed package")
	fmt.Printf("  %s %s\n", cmd("source-of <pkg>", 22), "Print which manager has the package")
	fmt.Printf("  %s %s\n", cmd("outdated", 22), "List packages with available updates")
	fmt.Println()

	fmt.Printf("%s %s\n", section("HEADLESS"), muted("· write"))
	fmt.Printf("  %s %s\n", cmd("install <pkg>...", 22), "Install one or more packages")
	fmt.Printf("  %s %s\n", cmd("remove <pkg>...", 22), "Remove a package (--with-deps for orphans)")
	fmt.Printf("  %s %s\n", cmd("upgrade <pkg>...", 22), "Upgrade installed packages to latest")
	fmt.Println()

	fmt.Println(section("COMMON FLAGS"))
	fmt.Printf("  %s %s\n", flagText("--manager M, -m", 18), "filter manager (e.g. pacman, pacman,aur, !brew)")
	fmt.Printf("  %s %s\n", flagText("--json", 18), "emit a JSON envelope on stdout")
	fmt.Printf("  %s %s\n", flagText("--no-cache", 18), "bypass the scan/update cache")
	fmt.Printf("  %s %s\n", flagText("--yes, -y", 18), "skip the y/N prompt (also skips manager's prompt)")
	fmt.Printf("  %s %s\n", flagText("--dry-run", 18), "print the command without running it (writes only)")
	fmt.Printf("  %s %s\n", flagText("--quiet, -q", 18), "suppress progress on stderr")
	fmt.Printf("  %s\n", muted("Run `gpk <subcommand> --help` for the full per-command flag list."))
	fmt.Println()

	fmt.Println(section("EXIT CODES"))
	fmt.Printf("  %s  %s\n", yellow+"0"+reset, "success / clean / yes")
	fmt.Printf("  %s  %s\n", yellow+"1"+reset, "error (bad flag, scan failed, IO)")
	fmt.Printf("  %s  %s\n", yellow+"2"+reset, "meaningful 'no' (not installed, has updates, not found)")
	fmt.Printf("  %s  %s\n", yellow+"3"+reset, "ambiguous (package available in multiple managers; use --manager)")
	fmt.Println()

	fmt.Println(section("TUI KEYBINDS"))
	fmt.Printf("  %s %s\n", cmd("j/k, ↑/↓", 18), "Navigate up/down")
	fmt.Printf("  %s %s\n", cmd("g/G", 18), "Jump to top/bottom")
	fmt.Printf("  %s %s\n", cmd("Ctrl+d/u", 18), "Half page down/up")
	fmt.Printf("  %s %s\n", cmd("Tab/Shift+Tab", 18), "Cycle manager tabs")
	fmt.Printf("  %s %s\n", cmd("/", 18), "Fuzzy search")
	fmt.Printf("  %s %s\n", cmd("Enter", 18), "Package details")
	fmt.Printf("  %s %s\n", cmd("u / x (detail)", 18), "Upgrade / remove the focused package")
	fmt.Printf("  %s %s\n", cmd("i", 18), "Search + install across managers")
	fmt.Printf("  %s %s\n", cmd("s / d", 18), "Save snapshot / diff against last")
	fmt.Printf("  %s %s\n", cmd("e", 18), "Export packages")
	fmt.Printf("  %s %s\n", cmd("r", 18), "Rescan all managers")
	fmt.Printf("  %s %s\n", cmd("t", 18), "Theme picker")
	fmt.Printf("  %s %s\n", cmd("? / q", 18), "Toggle help / quit")
	fmt.Println()

	fmt.Printf("%s %s\n", section("SUPPORTED MANAGERS"), muted("(36)"))
	fmt.Println(muted("  brew, pacman, aur, apt, dnf, snap, pip, pipx, uv, cargo, go,"))
	fmt.Println(muted("  npm, pnpm, bun, flatpak, macports, pkgsrc, opam, gem, pkg,"))
	fmt.Println(muted("  composer, mas, apk, nix, conda, luarocks, xbps, portage, guix,"))
	fmt.Println(muted("  winget, chocolatey, nuget, powershell, windows-updates, scoop, maven"))
	fmt.Println()

	fmt.Println(section("DATA PATHS"))
	fmt.Printf("  %s %s\n", cmd("Cache", 12), muted("~/.local/share/glazepkg/cache/"))
	fmt.Printf("  %s %s\n", cmd("Snapshots", 12), muted("~/.local/share/glazepkg/snapshots/"))
	fmt.Printf("  %s %s\n", cmd("Exports", 12), muted("~/.local/share/glazepkg/exports/"))
}
