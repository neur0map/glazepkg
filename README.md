<div align="center">

# GlazePKG (`gpk`)

**See every package on your system — one gorgeous terminal dashboard.**

A beautiful TUI that unifies **41 package managers** into a single searchable, snapshotable, diffable view.
Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Zero config. One binary. Just run `gpk`.

[![CI](https://img.shields.io/github/actions/workflow/status/neur0map/glazepkg/ci.yml?style=for-the-badge)](https://github.com/neur0map/glazepkg/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/neur0map/glazepkg?style=for-the-badge&color=00ADD8)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/neur0map/glazepkg?style=for-the-badge&color=4c1)](https://github.com/neur0map/glazepkg/releases)
[![License: GPL-3.0](https://img.shields.io/badge/license-GPL--3.0-blue?style=for-the-badge)](LICENSE)
[![Downloads](https://img.shields.io/github/downloads/neur0map/glazepkg/total?style=for-the-badge&color=orange)](https://github.com/neur0map/glazepkg/releases)
[![Stars](https://img.shields.io/github/stars/neur0map/glazepkg?style=for-the-badge&color=yellow)](https://github.com/neur0map/glazepkg/stargazers)

![demo](demo.gif?v=2026-05-10)

</div>

---

## Why?

You have `brew`, `pip`, `cargo`, `npm`, `apt`, maybe `flatpak` — all installing software independently. Knowing what's actually on your machine means running 6+ commands across different CLIs with different flags and output formats.

**GlazePKG fixes this.** One command, one view, every package. Track what changed over time with snapshots and diffs. Export everything to JSON for backup or migration.

## Features

- **41 package managers** — brew, pacman, AUR, apt, dnf, snap, pip, pipx, uv, cargo, go, npm, pnpm, bun, flatpak, MacPorts, pkgsrc, opam, gem, pkg, composer, mas, apk, nix, conda/mamba, luarocks, XBPS, Portage, Guix, winget, Chocolatey, Scoop, NuGet, PowerShell modules, Maven, Windows Update, AM/AppMan, gvm, mise, Quicklisp, softwareupdate
- **Instant startup** — scans once, caches for 10 days, opens in milliseconds on repeat launches
- **Fuzzy search** — find any package across all managers instantly with `/`
- **Package operations** — upgrade, remove, search, and install packages without leaving the TUI
- **Full command line** — every action as a typed command, plus the pacman/yay short flags (`-S`, `-Ss`, `-Syu`, `-R`, `-Q`) so it feels familiar from the first keystroke
- **Multi-select** — batch upgrade or remove with smart sudo batching
- **Snapshots & diffs** — save system state, diff to see what changed
- **Update detection** — `↑` indicator for packages with available updates
- **Export** — dump to JSON or text for backup, migration, or dotfile tracking
- **Zero dependencies** — single static Go binary, cross-platform

## Install

### Homebrew (macOS / Linux)

```bash
brew install neur0map/tap/gpk
```

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

<details>
<summary><strong>Windows: seeing a "virus detected" or "Windows protected your PC" warning?</strong></summary>

It's a false alarm. The Windows build isn't signed yet, and antivirus tools often flag fresh, unsigned programs made with Go — the exact same code on macOS and Linux is fine. There's nothing harmful in it ([why this happens](https://go.dev/doc/faq#virus)).

Ways around it:

- **Skip the download and warning entirely** — install with Go:

  ```
  go install github.com/neur0map/glazepkg/cmd/gpk@latest
  ```

- **Confirm the file is the real one.** Every release ships a `checksums.txt`. In PowerShell:

  ```
  Get-FileHash .\gpk-windows-amd64.exe -Algorithm SHA256
  ```

  Check the result matches the line for that file in `checksums.txt`.

- **Let it run.** After checking the file: right-click it → Properties → tick **Unblock** → OK (or run `Unblock-File .\gpk-windows-amd64.exe`). If you get a blue "Windows protected your PC" box, click **More info → Run anyway**.

- **On a work computer?** You might not see a "Run anyway" button — that's your IT department blocking unsigned apps, and we can't change that from our side. Ask them to allow it, or use the Go install above.

</details>

<details>
<summary>Build from source</summary>

```bash
git clone https://github.com/neur0map/glazepkg.git
cd glazepkg && go build ./cmd/gpk
```

If `gpk` is not found after installing via `go install`, add Go's bin directory to your PATH:

```bash
export PATH="$PATH:$HOME/go/bin"
```

</details>

## Quick Start

```
gpk              Launch TUI
gpk update       Self-update to latest release
gpk version      Show current version
gpk --help       Show keybind reference
```

Just run `gpk` — navigate with `j`/`k`, switch managers with `Tab`, search with `/`, press `s` to snapshot, `d` to diff, `e` to export. Press `?` for the full keybind reference.

## Command line

gpk isn't only the dashboard. Every action has a typed command, and the
pacman/yay short flags work too, so you can drive it from the shell without
opening the TUI.

```
gpk firefox               # search and pick something to install (like yay)
gpk -S ffmpeg             # install
gpk -S ffmpeg --aur       # install, scoped to one manager
gpk install black@24.1.0  # install a specific version
gpk -Ss ripgrep           # search every manager at once
gpk -Syu                  # update everything
gpk -R foo                # remove  (-Rns to take orphaned deps too)
gpk downgrade foo         # roll back to an earlier version
gpk -Sc                   # clear cached downloads
gpk autoremove            # remove dependencies nothing needs
gpk -Qi foo               # info    ·  gpk -Q lists everything installed
```

When a package exists in more than one manager, gpk lists them with versions and
lets you choose. Mistype a name and it suggests the closest match. Scope any
command with `--aur`, `--brew`, `-m pacman,aur`, or leave it off and let gpk
find it.

| Plain words | Short flags |
|---|---|
| `gpk install foo` | `gpk -S foo` |
| `gpk remove foo` | `gpk -R foo` |
| `gpk upgrade` | `gpk -Syu` |
| `gpk search foo` | `gpk -Ss foo` |
| `gpk list` | `gpk -Q` |

Output is colored to match your theme on a terminal and falls back to plain text
when piped. Run `gpk --help` for the full command and flag reference.

<details>
<summary><strong>Keybindings</strong></summary>

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
| `Enter` (deps screen) | Open the selected dependency's details |
| `h` (detail) | Package help/usage |
| `m` (detail) | Open man page |
| `o` (detail) | Open package page in browser |
| `e` (detail) | Edit description |
| `i` | Search + install packages |
| `m` | Toggle multi-select mode |
| `Space` (multi-select) | Toggle package selection |
| `s` | Save snapshot |
| `d` | Diff against last snapshot |
| `e` | Export (JSON or text) |
| `r` | Force rescan |
| `U` | System update for the active manager tab |
| `Q` | View / cancel the operation queue |
| `?` | Help overlay |
| `q` | Quit |

</details>

<details>
<summary><strong>Package Operations</strong></summary>

### Upgrade (`u` in detail view)

Open a package with `Enter`, then press `u`. A confirmation modal shows the exact command. Privileged managers (apt, pacman, dnf, snap, apk, xbps) include a password field for sudo. The upgrade runs in the background while you keep using the TUI.

### Remove (`x` in detail view)

Open a package with `Enter`, then press `x`. Managers that support it (apt, pacman, dnf, xbps) offer two modes: remove package only, or remove package with orphaned dependencies. If the package is required by other packages, a warning is shown before proceeding.

### Search + Install (`i`)

Press `i` from the package list to open the search view. Type a query and results stream in from all installed managers in parallel. Results are deduplicated by name — expand a row to see all available sources and versions. Press `i` on a result to install it. Already-installed packages are marked.

### Multi-Select (`m`)

Press `m` to enter selection mode. Use `Space` to toggle packages, navigate and search normally — selections persist across tabs and searches. Press `u` to upgrade all selected or `x` to remove all selected. The confirmation modal groups operations by privilege level so you only enter your password once.

All operations work on macOS, Linux, and Windows. Each manager maps to its correct native command automatically.

</details>

<details>
<summary><strong>Supported Package Managers (41)</strong></summary>

| Manager | Platform | What it scans | Descriptions |
|---------|----------|---------------|-------------|
| **brew** | macOS/Linux | Installed formulae and casks | batch via JSON |
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
| **maven** | Cross-platform | Local Maven artifacts in `~/.m2/repository` | — |
| **windows-updates** | Windows | Pending Windows system updates | — |
| **am** | Linux | AppImage apps via AM/AppMan | inline from scan |
| **gvm** | Cross-platform | Go toolchains installed via gvm | — |
| **mise** | Cross-platform | Tools managed by mise | — |
| **quicklisp** | Cross-platform | Common Lisp libraries via Quicklisp | — |
| **softwareupdate** | macOS | Pending macOS system updates | inline from scan |

- Managers that aren't installed are silently skipped — no errors, no config needed.
- Descriptions are fetched in the background and cached for 24 hours.
- Packages with available updates show a `↑` indicator next to their version (checked every 7 days).
- Press `d` in the detail view to see full dependency tree for any package.
- Press `h` in the detail view to see the package's `--help` output.
- Press `e` in the detail view to add custom descriptions — these persist across sessions and won't be overwritten.

</details>

<details>
<summary><strong>Snapshots & Diffs</strong></summary>

GlazePKG can track how your system changes over time:

1. **Snapshot** (`s`) — saves every package name, version, and source to a timestamped JSON file
2. **Diff** (`d`) — compares your current packages against the last snapshot, showing:
   - **Added** packages (new installs)
   - **Removed** packages (uninstalls)
   - **Upgraded** packages (version changes)

Use this to audit what changed after a `brew upgrade`, track drift across machines, or catch unexpected installs.

</details>

<details>
<summary><strong>Data Storage</strong></summary>

All data lives under `~/.local/share/glazepkg/` (respects `XDG_DATA_HOME`):

| Data | Path | Retention |
|------|------|-----------|
| Scan cache | `cache/scan.json` | 10 days (auto-refresh) |
| Description cache | `cache/descriptions.json` | 24 hours |
| Update cache | `cache/updates.json` | 7 days |
| User notes | `notes.json` | Permanent |
| Snapshots | `snapshots/*.json` | Permanent |
| Exports | `exports/*.json` or `*.txt` | Permanent |

</details>

## Contributing

Want to add a package manager or fix a bug? Check out [CONTRIBUTING.md](CONTRIBUTING.md). Each manager is a single Go file — easy to add.

## Built With

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components
- [Fuzzy](https://github.com/sahilm/fuzzy) — fuzzy matching

## Star History

<a href="https://www.star-history.com/?repos=neur0map%2Fglazepkg&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/image?repos=neur0map/glazepkg&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/image?repos=neur0map/glazepkg&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/image?repos=neur0map/glazepkg&type=date&legend=top-left" />
 </picture>
</a>

## License

[GPL-3.0](LICENSE)
