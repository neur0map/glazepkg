<div align="center">

# GlazePKG (`gpk`)

**See every package on your system — one gorgeous terminal dashboard.**

A beautiful TUI that unifies **34 package managers** into a single searchable, snapshotable, diffable view.
Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Zero config. One binary. Just run `gpk`.

[![CI](https://img.shields.io/github/actions/workflow/status/neur0map/glazepkg/ci.yml?style=for-the-badge)](https://github.com/neur0map/glazepkg/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/neur0map/glazepkg?style=for-the-badge&color=00ADD8)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/neur0map/glazepkg?style=for-the-badge&color=4c1)](https://github.com/neur0map/glazepkg/releases)
[![License: GPL-3.0](https://img.shields.io/badge/license-GPL--3.0-blue?style=for-the-badge)](LICENSE)
[![Downloads](https://img.shields.io/github/downloads/neur0map/glazepkg/total?style=for-the-badge&color=orange)](https://github.com/neur0map/glazepkg/releases)
[![Stars](https://img.shields.io/github/stars/neur0map/glazepkg?style=for-the-badge&color=yellow)](https://github.com/neur0map/glazepkg/stargazers)

![demo](demo.gif)

</div>

---

## Why?

You have `brew`, `pip`, `cargo`, `npm`, `apt`, maybe `flatpak` — all installing software independently. Knowing what's actually on your machine means running 6+ commands across different CLIs with different flags and output formats.

**GlazePKG fixes this.** One command, one view, every package. Track what changed over time with snapshots and diffs. Export everything to JSON for backup or migration.

## Features

- **34 package managers** — brew, pacman, AUR, apt, dnf, snap, pip, pipx, cargo, go, npm, pnpm, bun, flatpak, MacPorts, pkgsrc, opam, gem, pkg, composer, mas, apk, nix, conda/mamba, luarocks, XBPS, Portage, Guix, winget, Chocolatey, Scoop, NuGet, PowerShell modules, Windows Update
- **Instant startup** — scans once, caches for 10 days, opens in milliseconds on repeat launches
- **Size filter** — press `f` to cycle through size filters (< 1 MB, 1–10 MB, 10–100 MB, > 100 MB, has updates); sorted largest-first
- **Fuzzy search** — find any package across all managers instantly with `/`
- **Snapshots & diffs** — save your system state, then diff to see what was added, removed, or upgraded
- **Update detection** — packages with available updates show a `↑` indicator (checked every 7 days)
- **Package operations** — upgrade, remove, and install packages without leaving the TUI. Every operation shows a confirmation with the exact command before running.
- **Search + install** — press `i` to search across all installed managers in parallel, browse deduplicated results, expand to see all sources, and install with one keypress
- **Multi-select** — press `m` to select multiple packages, then batch upgrade or remove them all at once with smart sudo batching
- **Custom descriptions** — press `e` in the detail view to annotate any package; persists across sessions
- **Background descriptions** — package summaries load asynchronously and cache for 24 hours
- **Export** — dump your full package list to JSON or text for backup, migration, or dotfile tracking
- **Self-updating** — run `gpk update` to grab the latest release automatically
- **Tokyo Night theme** — carefully designed color palette with per-manager color coding
- **Vim keybindings** — `j`/`k`, `g`/`G`, `Ctrl+d`/`Ctrl+u` — feels like home
- **Zero dependencies** — single static Go binary, no runtime requirements
- **Cross-platform** — works on macOS, Linux, and Windows; skips managers that aren't installed

## Install

### Arch Linux (AUR)

```bash
yay -S gpk-bin
```

### Go

```bash
go install github.com/neur0map/glazepkg/cmd/gpk@latest
```

### Pre-built binaries

Grab a binary from [releases](https://github.com/neur0map/glazepkg/releases) for macOS (ARM/Intel), Linux (x64/ARM), or Windows (x64/ARM).

### Build from source

```bash
git clone https://github.com/neur0map/glazepkg.git
cd glazepkg && go build ./cmd/gpk
```

If `gpk` is not found after installing via `go install`, add Go's bin directory to your PATH:

```bash
# bash/zsh
export PATH="$PATH:$HOME/go/bin"
```

## Update

```bash
gpk update
```

Self-updates the binary to the latest release. Run `gpk version` to check your current version.

## Quick Start

```
gpk              Launch TUI
gpk update       Self-update to latest release
gpk version      Show current version
gpk --help       Show keybind reference
```

Just run `gpk` — it drops straight into a beautiful table. Navigate with `j`/`k`, switch managers with `Tab`, search with `/`, press `s` to snapshot, `d` to diff, `e` to export. Press `?` for the full keybind reference.

## Package Operations

### Upgrade (`u` in detail view)

Open a package with `Enter`, then press `u`. A confirmation modal shows the exact command. Privileged managers (apt, pacman, dnf, snap, apk, xbps) include a password field for sudo. The upgrade runs in the background while you keep using the TUI.

### Remove (`x` in detail view)

Open a package with `Enter`, then press `x`. Managers that support it (apt, pacman, dnf, xbps) offer two modes: remove package only, or remove package with orphaned dependencies. If the package is required by other packages, a warning is shown before proceeding.

### Search + Install (`i`)

Press `i` from the package list to open the search view. Type a query and results stream in from all installed managers in parallel. Results are deduplicated by name — expand a row to see all available sources and versions. Press `i` on a result to install it. Already-installed packages are marked.

### Multi-Select (`m`)

Press `m` to enter selection mode. Use `Space` to toggle packages, navigate and search normally — selections persist across tabs and searches. Press `u` to upgrade all selected or `x` to remove all selected. The confirmation modal groups operations by privilege level so you only enter your password once.

All operations work on macOS, Linux, and Windows. Each manager maps to its correct native command automatically.

## Supported Package Managers

| Manager | Platform | What it scans | Descriptions |
|---------|----------|---------------|-------------|
| **brew** | macOS/Linux | Installed formulae | batch via JSON |
| **pacman** | Arch | Explicit native packages | `pacman -Qi` |
| **AUR** | Arch | Foreign/AUR packages | `pacman -Qi` |
| **apt** | Debian/Ubuntu | Installed packages | `apt-cache show` |
| **dnf** | Fedora/RHEL | Installed packages | `dnf info` |
| **snap** | Ubuntu/Linux | Snap packages | `snap info` |
| **pip** | Cross-platform | Top-level Python packages | `pip show` |
| **pipx** | Cross-platform | Isolated Python CLI tools | — |
| **cargo** | Cross-platform | Installed Rust binaries | — |
| **go** | Cross-platform | Go binaries in `~/go/bin` | — |
| **npm** | Cross-platform | Global Node.js packages | `npm info` |
| **pnpm** | Cross-platform | Global pnpm packages | `pnpm info` |
| **bun** | Cross-platform | Global Bun packages | — |
| **flatpak** | Linux | Flatpak applications | `flatpak info` |
| **MacPorts** | macOS | Installed ports | `port info` |
| **pkgsrc** | NetBSD/cross-platform | Installed packages | `pkg_info` |
| **opam** | Cross-platform | OCaml packages | `opam show` |
| **gem** | Cross-platform | Ruby gems | `gem info` |
| **pkg** | FreeBSD | Installed packages | inline from scan |
| **composer** | Cross-platform | Global PHP packages | inline from JSON |
| **mas** | macOS | Mac App Store apps | — |
| **apk** | Alpine Linux | Installed packages | `apk info` |
| **nix** | NixOS/cross-platform | Nix profile, nix-env, and NixOS system packages | `nix-env -qa` |
| **conda/mamba** | Cross-platform | Conda environments | — |
| **luarocks** | Cross-platform | Lua rocks | `luarocks show` |
| **XBPS** | Void Linux | Installed packages | `xbps-query` |
| **Portage** | Gentoo | Installed ebuilds via `qlist` | `emerge -s` |
| **Guix** | GNU Guix | Installed packages | `guix show` |
| **winget** | Windows | Windows Package Manager | — |
| **chocolatey** | Windows | Chocolatey packages (v1 + v2) | — |
| **scoop** | Windows | Scoop packages | — |
| **nuget** | Cross-platform | NuGet global package cache | — |
| **powershell** | Cross-platform | PowerShell modules | via scan |
| **windows-updates** | Windows | Pending Windows system updates | — |

- Managers that aren't installed are silently skipped — no errors, no config needed.
- Descriptions are fetched in the background and cached for 24 hours.
- Packages with available updates show a `↑` indicator next to their version (checked every 7 days).
- Press `d` in the detail view to see full dependency tree for any package.
- Press `h` in the detail view to see the package's `--help` output.
- Press `e` in the detail view to add custom descriptions — these persist across sessions and won't be overwritten.

## Keybindings

| Key | Action |
|-----|--------|
| `j`/`k`, `↑`/`↓` | Navigate |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+d` / `Ctrl+u` | Half-page down / up |
| `PgDn` / `PgUp` | Page down / up |
| `Tab` / `Shift+Tab` | Cycle manager tabs |
| `f` | Cycle size filter |
| `/` | Fuzzy search |
| `Enter` | Package details |
| `u` (detail) | Upgrade package |
| `x` (detail) | Remove package |
| `d` (detail) | View dependencies |
| `h` (detail) | Package help/usage |
| `e` (detail) | Edit description |
| `i` | Search + install packages |
| `m` | Toggle multi-select mode |
| `Space` (multi-select) | Toggle package selection |
| `s` | Save snapshot |
| `d` | Diff against last snapshot |
| `e` | Export (JSON or text) |
| `r` | Force rescan |
| `?` | Help overlay |
| `q` | Quit |

## Snapshots & Diffs

GlazePKG can track how your system changes over time:

1. **Snapshot** (`s`) — saves every package name, version, and source to a timestamped JSON file
2. **Diff** (`d`) — compares your current packages against the last snapshot, showing:
   - **Added** packages (new installs)
   - **Removed** packages (uninstalls)
   - **Upgraded** packages (version changes)

Use this to audit what changed after a `brew upgrade`, track drift across machines, or catch unexpected installs.

## Data Storage

All data lives under `~/.local/share/glazepkg/` (respects `XDG_DATA_HOME`):

| Data | Path | Retention |
|------|------|-----------|
| Scan cache | `cache/scan.json` | 10 days (auto-refresh) |
| Description cache | `cache/descriptions.json` | 24 hours |
| Update cache | `cache/updates.json` | 7 days |
| User notes | `notes.json` | Permanent |
| Snapshots | `snapshots/*.json` | Permanent |
| Exports | `exports/*.json` or `*.txt` | Permanent |

## Built With

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components
- [Fuzzy](https://github.com/sahilm/fuzzy) — fuzzy matching

## Contributing

Want to add a package manager or fix a bug? Check out [CONTRIBUTING.md](CONTRIBUTING.md). Each manager is a single Go file — easy to add.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=neur0map/glazepkg&type=Date)](https://star-history.com/#neur0map/glazepkg&Date)

## License

[GPL-3.0](LICENSE)
