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
- **Universal single-key package upgrades** — press **u** to upgrade the highlighted package with the active manager; privileged managers (apt, dnf, pacman, snap, apk, XBPS) surface a confirmation overlay so the upgrade never runs on a single keypress, and even gem/flatpak/pipx/opam/apk/XBPS/conda/luarocks ship native commands so the flow never leaves `gpk`
- **Custom descriptions** — press `e` in the detail view to annotate any package; persists across sessions
- **Background descriptions** — package summaries load asynchronously and cache for 24 hours
- **Export** — dump your full package list to JSON or text for backup, migration, or dotfile tracking
- **Self-updating** — run `gpk update` to grab the latest release automatically
- **Tokyo Night theme** — carefully designed color palette with per-manager color coding
- **Vim keybindings** — `j`/`k`, `g`/`G`, `Ctrl+d`/`Ctrl+u` — feels like home
- **Zero dependencies** — single static Go binary, no runtime requirements
- **Cross-platform** — works on macOS, Linux, and Windows; skips managers that aren't installed

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

```powershell
# PowerShell (Windows)
$env:PATH += ";$env:USERPROFILE\go\bin"
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

Just run `gpk` — it drops straight into a beautiful table. Navigate with `j`/`k`, switch managers with `Tab`, search with `/`, press `s` to snapshot, `d` to diff, `e` to export, `u` to upgrade the selected package. Press `?` for the full keybind reference.

## Usage

1. Launch `gpk` and pick a manager tab with `Tab`/`Shift+Tab`.
2. Highlight the package you want to upgrade with `j`/`k`.
3. Press `u` to trigger a single-package upgrade for the active manager; privileged managers (apt, dnf, pacman, snap, apk, XBPS) show a confirmation overlay asking for `y`/`Enter` before the command runs so the upgrade never happens on a single keypress, and the flow now includes native support for gem, flatpak, pipx, opam, apk, XBPS, conda, and luarocks.
4. The status bar reports `upgrading <name>...` while the command runs; it ends with `Package upgraded successfully` on success or the terminal error on failure.
5. If the manager cannot upgrade individual packages you see: `This package manager does not support upgrading a single package.` which keeps the UI responsive.

The same flow works on Windows, Linux, and macOS — `u` maps to the correct command for the selected manager automatically.

## Supported Package Managers

| Manager | Platform | What it scans | Descriptions |
|---------|----------|---------------|-------------|
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
| `d` (detail) | View dependencies |
| `h` (detail) | Package help/usage |
| `e` (detail) | Edit description |
| `s` | Save snapshot |
| `u` | Upgrade selected package (privileged managers show a y/n overlay before the command runs) |
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
