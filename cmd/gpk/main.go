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
		args := os.Args[1:]
		if opArgs, ok := cli.TranslateOps(args); ok {
			args = opArgs
		}
		os.Exit(cli.Dispatch(args, manager.All(), version, os.Stdout, os.Stderr, os.Stdin))
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
	fmt.Printf("  %s %s\n", cmd("gpk", 24), "Launch the TUI")
	fmt.Printf("  %s %s\n", cmd("gpk <pkg>", 24), "Search and install, the way yay does")
	fmt.Printf("  %s %s\n", cmd("gpk <subcommand> ...", 24), "Run a headless command")
	fmt.Printf("  %s %s\n", cmd("gpk update", 24), "Self-update to latest release")
	fmt.Printf("  %s %s\n", cmd("gpk -h, --help", 24), "Show this help")
	fmt.Println()

	fmt.Printf("%s %s\n", section("COMMANDS"), muted("· read"))
	fmt.Printf("  %s %s\n", cmd("search <query>", 24), "Search packages across every manager")
	fmt.Printf("  %s %s\n", cmd("list [filter]", 24), "List installed packages")
	fmt.Printf("  %s %s\n", cmd("info <pkg>", 24), "Show details for one package")
	fmt.Printf("  %s %s\n", cmd("source-of <pkg>", 24), "Print which manager has a package")
	fmt.Printf("  %s %s\n", cmd("outdated", 24), "List packages with available updates")
	fmt.Printf("  %s %s\n", cmd("installed <pkg>...", 24), "Check if packages are installed (exit 0/2)")
	fmt.Printf("  %s %s\n", cmd("managers", 24), "Show which managers are detected, with counts")
	fmt.Println()

	fmt.Printf("%s %s\n", section("COMMANDS"), muted("· write"))
	fmt.Printf("  %s %s\n", cmd("install <pkg>...", 24), "Install packages (name@version to pin)")
	fmt.Printf("  %s %s\n", cmd("remove <pkg>...", 24), "Remove a package (--with-deps for orphans)")
	fmt.Printf("  %s %s\n", cmd("upgrade [pkg...]", 24), "Upgrade packages, or everything if none given")
	fmt.Printf("  %s %s\n", cmd("downgrade <pkg>", 24), "Install an earlier version (version picker)")
	fmt.Printf("  %s %s\n", cmd("clean", 24), "Clear cached downloads (--all for everything)")
	fmt.Printf("  %s %s\n", cmd("autoremove", 24), "Remove orphaned deps (--print to just list)")
	fmt.Printf("  %s %s\n", cmd("hold/unhold <pkg>", 24), "Pin a package so upgrades skip it")
	fmt.Printf("  %s %s\n", cmd("history", 24), "Show recent actions gpk performed")
	fmt.Printf("  %s %s\n", cmd("undo", 24), "Reverse gpk's last action")
	fmt.Println()

	fmt.Println(section("PACMAN / YAY FLAGS"))
	fmt.Printf("  %s %-12s  %s %s\n", flagText("-S pkg", 10), "install", flagText("-Ss term", 10), "search")
	fmt.Printf("  %s %-12s  %s %s\n", flagText("-Syu", 10), "upgrade all", flagText("-Si pkg", 10), "info")
	fmt.Printf("  %s %-12s  %s %s\n", flagText("-R pkg", 10), "remove", flagText("-Rns pkg", 10), "remove + deps")
	fmt.Printf("  %s %-12s  %s %s\n", flagText("-Q [term]", 10), "list", flagText("-Qi pkg", 10), "info")
	fmt.Printf("  %s %-12s  %s %s\n", flagText("-Qu", 10), "outdated", flagText("-Qdt", 10), "list orphans")
	fmt.Printf("  %s %s\n", flagText("-Sc / -Scc", 10), "clean cached downloads")
	fmt.Println()

	fmt.Println(section("COMMON FLAGS"))
	fmt.Printf("  %s %s\n", flagText("--manager M, -m", 20), "filter manager (e.g. pacman, pacman,aur, !brew)")
	fmt.Printf("  %s %s\n", flagText("--aur, --brew, ...", 20), "shorthand for --manager <name>")
	fmt.Printf("  %s %s\n", flagText("--json", 20), "emit a JSON envelope on stdout")
	fmt.Printf("  %s %s\n", flagText("--no-cache", 20), "bypass the scan/update cache")
	fmt.Printf("  %s %s\n", flagText("--yes, -y", 20), "skip prompts")
	fmt.Printf("  %s %s\n", flagText("--dry-run", 20), "print the command without running it")
	fmt.Printf("  %s %s\n", flagText("--quiet, -q", 20), "suppress progress on stderr")
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

	fmt.Printf("%s %s\n", section("SUPPORTED MANAGERS"), muted(fmt.Sprintf("(%d)", len(manager.All()))))
	fmt.Printf("  %s %s\n", cmd("gpk managers", 22), muted("shows which are detected on your system"))
	fmt.Println(muted("  brew, pacman, aur, apt, dnf, snap, pip, pipx, uv, cargo, go,"))
	fmt.Println(muted("  npm, pnpm, bun, flatpak, macports, pkgsrc, opam, gem, pkg,"))
	fmt.Println(muted("  composer, mas, apk, nix, conda, luarocks, xbps, portage, guix,"))
	fmt.Println(muted("  winget, chocolatey, nuget, powershell, windows-updates, scoop,"))
	fmt.Println(muted("  maven, am, gvm, mise, quicklisp, softwareupdate"))
	fmt.Println()

	fmt.Println(section("DATA PATHS"))
	fmt.Printf("  %s %s\n", cmd("Cache", 12), muted("~/.local/share/glazepkg/cache/"))
	fmt.Printf("  %s %s\n", cmd("Snapshots", 12), muted("~/.local/share/glazepkg/snapshots/"))
	fmt.Printf("  %s %s\n", cmd("Exports", 12), muted("~/.local/share/glazepkg/exports/"))
}
