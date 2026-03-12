# GlazePKG Design

**Command:** `gpk`
**Module:** `github.com/neur0map/glazepkg`
**Platforms:** macOS, Linux (no Windows)
**Stack:** Go + Charmbracelet (Bubble Tea, Lip Gloss, Bubbles)
**Theme:** Tokyo Night

## Overview

GlazePKG is a TUI-only multi-platform package manager aggregator. It scans all installed package managers, presents packages in a searchable table with live descriptions, and supports snapshots and diffs.

Single entry point: `gpk` launches the TUI. No subcommands. All actions are keybinds.

## Package Managers (13)

| Manager | Detection command | Description source |
|---|---|---|
| brew | `brew list --formula` | `brew info --json=v2 --installed` (batch) |
| apt | `dpkg-query -W` | `apt-cache show <name>` |
| dnf | `dnf list installed` | `dnf info <name>` |
| snap | `snap list` | `snap info <name>` |
| pacman | `pacman -Qen` | `pacman -Qi <name>` |
| AUR | `pacman -Qm` | `pacman -Qi <name>` |
| pip | `pip list --user` | `pip show <name>` |
| pipx | `pipx list --json` | included in JSON output |
| cargo | `cargo install --list` | `~/.cargo/registry` or binary name |
| go | `~/go/bin/` listing | binary name only |
| npm | `npm list -g` | `npm info <name> description` |
| bun | `bun pm ls -g` | npm registry fallback |
| flatpak | `flatpak list --app` | `flatpak info <name>` |

Managers not installed are silently skipped.

## Project Structure

```
glazepkg/
├── cmd/gpk/main.go              # entry point, TUI only
├── internal/
│   ├── manager/
│   │   ├── manager.go            # Manager interface + All()
│   │   ├── describe.go           # description cache logic
│   │   ├── brew.go               # NEW
│   │   ├── apt.go                # NEW
│   │   ├── dnf.go                # NEW
│   │   ├── snap.go               # NEW
│   │   ├── pacman.go
│   │   ├── aur.go
│   │   ├── pip.go
│   │   ├── pipx.go
│   │   ├── cargo.go
│   │   ├── go.go
│   │   ├── npm.go
│   │   ├── bun.go
│   │   └── flatpak.go
│   ├── model/
│   │   └── package.go            # Package, Snapshot, Diff types
│   ├── snapshot/
│   │   ├── store.go
│   │   └── snapshot.go
│   └── ui/
│       ├── app.go                # root Bubble Tea model
│       ├── table.go              # main table view
│       ├── detail.go             # package detail overlay
│       ├── diff.go               # diff view
│       ├── search.go             # fuzzy finder overlay
│       ├── help.go               # ? help overlay
│       ├── export.go             # export logic (triggered by 'e')
│       ├── theme.go              # Tokyo Night colors
│       └── keys.go               # keybind definitions
└── go.mod
```

## TUI Layout

### Main Table View (launch screen)

```
┌─────────────────────────────────────────────────────────────────────┐
│  ✦ GlazePKG                    [All] brew  npm  pip  cargo  ...    │
├──────────────────────┬──────────┬─────────┬─────────────────────────┤
│  Package             │ Version  │ Manager │ Description             │
├──────────────────────┼──────────┼─────────┼─────────────────────────┤
│▸ bat                 │ 0.24.0   │ brew    │ Cat clone with wings    │
│  bun                 │ 1.1.38   │ brew    │ Fast JS runtime         │
│  curl                │ 8.7.1    │ apt     │ Transfer data with URLs │
│  ...                 │          │         │                         │
├─────────────────────────────────────────────────────────────────────┤
│  254 packages │ /: search  tab: filter  r: rescan  ?: help         │
└─────────────────────────────────────────────────────────────────────┘
```

4 columns: Package, Version, Manager, Description.
Description truncated to fit terminal width.
Manager column uses colored badge pills.
Selected row: blue highlight (#7aa2f7) with white text.

### Keybinds

| Key | Action |
|---|---|
| `j/k` or `↑/↓` | Navigate rows |
| `g/G` | Jump to top/bottom |
| `Ctrl+d/u` | Page down/up |
| `Tab/Shift+Tab` | Cycle manager filter tabs |
| `/` | Fuzzy search (name + description) |
| `Enter` | Package detail view |
| `r` | Rescan all managers |
| `s` | Save snapshot |
| `d` | Diff against last snapshot |
| `e` | Export (format picker) |
| `?` | Help overlay |
| `q` / `Ctrl+c` | Quit |

### Fuzzy Search (`/`)

Inline search bar at top. Matches against package name AND description. Filters table live. Uses `sahilm/fuzzy` or similar Go fuzzy library. `Esc` clears and closes.

### Detail View (`Enter`)

Full-screen overlay: name, version, manager, description, size, dependencies, required-by, install date. `Esc` returns to table.

### Diff View (`d`)

Added (green), removed (red), upgraded (yellow) since last snapshot. `Esc` returns to table.

### Help Overlay (`?`)

Lists all keybinds. Any key dismisses.

## Tokyo Night Theme

```go
ColorBase     = "#1a1b26"  // background
ColorSurface  = "#3b4261"  // borders, inactive
ColorText     = "#a9b1d6"  // primary text
ColorSubtext  = "#565f89"  // dimmed text
ColorBlue     = "#7aa2f7"  // selected row, active tab
ColorPurple   = "#bb9af7"  // manager badges
ColorGreen    = "#9ece6a"  // added packages
ColorRed      = "#f7768e"  // removed packages
ColorYellow   = "#e0af68"  // version text, upgrades
ColorCyan     = "#7dcfff"  // descriptions
ColorOrange   = "#ff9e64"  // warnings, counts
```

## Description Caching

Descriptions cached to `~/.local/share/glazepkg/cache/descriptions.json`.
TTL: 24 hours. Stale entries refreshed on next scan.
Batch fetch where possible (brew supports `--installed` for all at once).

## Data Storage

- Snapshots: `~/.local/share/glazepkg/snapshots/<timestamp>.json`
- Description cache: `~/.local/share/glazepkg/cache/descriptions.json`

## Implementation Priority

1. Rename project to glazepkg, update module path, binary to `gpk`
2. Implement Tokyo Night theme (`theme.go`)
3. Build proper table view with 4 columns
4. Add fuzzy search with `sahilm/fuzzy`
5. Add new managers: brew, apt, dnf, snap
6. Add description fetching per manager + cache
7. Build help overlay
8. Build export picker
9. Polish: manager badge colors, responsive column widths, spinner
