<div align="center">

# GlazePKG (`gpk`)

**See every package on your system — one gorgeous terminal dashboard.**

A beautiful TUI that unifies **14 package managers** into a single searchable, snapshotable, diffable view.
Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Zero config. One binary. Just run `gpk`.

[![CI](https://github.com/neur0map/glazepkg/actions/workflows/ci.yml/badge.svg)](https://github.com/neur0map/glazepkg/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/neur0map/glazepkg)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/neur0map/glazepkg)](https://github.com/neur0map/glazepkg/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

![demo](demo.gif)

</div>

---

## Why?

You have `brew`, `pip`, `cargo`, `npm`, `apt`, maybe `flatpak` — all installing software independently. Knowing what's actually on your machine means running 6+ commands across different CLIs with different flags and output formats.

**GlazePKG fixes this.** One command, one view, every package. Track what changed over time with snapshots and diffs. Export everything to JSON for backup or migration.

## Features

- **14 package managers** — brew, pacman, AUR, apt, dnf, snap, pip, pipx, cargo, go, npm, pnpm, bun, flatpak (+ brew dependency tracking)
- **Instant startup** — scans once, caches for 10 days, opens in milliseconds on repeat launches
- **Fuzzy search** — find any package across all managers instantly with `/`
- **Snapshots & diffs** — save your system state, then diff to see what was added, removed, or upgraded
- **Update detection** — packages with available updates show a `↑` indicator (checked every 7 days)
- **Custom descriptions** — press `e` in the detail view to annotate any package; persists across sessions
- **Background descriptions** — package summaries load asynchronously and cache for 24 hours
- **Export** — dump your full package list to JSON or text for backup, migration, or dotfile tracking
- **Self-updating** — run `gpk update` to grab the latest release automatically
- **Tokyo Night theme** — carefully designed color palette with per-manager color coding
- **Vim keybindings** — `j`/`k`, `g`/`G`, `Ctrl+d`/`Ctrl+u` — feels like home
- **Zero dependencies** — single static Go binary, no runtime requirements
- **Cross-platform** — works on macOS and Linux; skips managers that aren't installed

## Install

```bash
go install github.com/neur0map/glazepkg/cmd/gpk@latest
```

If `gpk` is not found after installing, add Go's bin directory to your PATH:

```bash
# bash (~/.bashrc) or zsh (~/.zshrc)
echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.bashrc
source ~/.bashrc
```

```fish
# fish
fish_add_path ~/go/bin
```

Or grab a [pre-built binary](https://github.com/neur0map/glazepkg/releases) for macOS (ARM/Intel) or Linux (x64/ARM).

Or build from source:

```bash
git clone https://github.com/neur0map/glazepkg.git
cd glazepkg && go build ./cmd/gpk
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

## Supported Package Managers

| Manager | Platform | What it scans | Descriptions |
|---------|----------|---------------|-------------|
| **brew** | macOS/Linux | Explicitly installed formulae | batch via JSON |
| **brew-deps** | macOS/Linux | Auto-installed brew dependencies | batch via JSON |
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

- Managers that aren't installed are silently skipped — no errors, no config needed.
- Brew separates explicitly installed formulae from auto-pulled dependencies — deps go in a dedicated **deps** tab.
- Descriptions are fetched in the background and cached for 24 hours.
- Packages with available updates show a `↑` indicator next to their version (checked every 7 days).
- Press `e` in the detail view to add custom descriptions — these persist across sessions and won't be overwritten.

## Keybindings

| Key | Action |
|-----|--------|
| `j`/`k`, `↑`/`↓` | Navigate |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+d` / `Ctrl+u` | Half-page down / up |
| `PgDn` / `PgUp` | Page down / up |
| `Tab` / `Shift+Tab` | Cycle manager tabs |
| `/` | Fuzzy search |
| `Enter` | Package details |
| `e` (detail) | Edit description |
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

## License

[MIT](LICENSE)
