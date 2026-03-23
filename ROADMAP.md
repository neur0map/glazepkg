# GlazePKG Roadmap

gpk shows you everything installed on your system. The next step is letting you do something about it: update, remove, install, bulk-select, and pick a color scheme that doesn't hurt your eyes and doesn't look like a 2010 TUI.

This roadmap is an idea of what I want to do with gpk. It is not set in stone and is subject to change in the small details or the order in which things are shown here. I'm open to suggestions and ideas.

## Package Operations

### ~~Update (`u`)~~ — DONE

Press `u` in the package detail view. A confirmation modal shows the exact command that will run, with Yes/No buttons. Privileged managers (apt, dnf, pacman, snap, apk, xbps, chocolatey) show a sudo password field on Linux or an elevated terminal warning on Windows. The upgrade runs in the background with a status notification while the TUI stays interactive. 19 managers support single-package upgrades via the `Upgrader` interface. Post-upgrade rescan refreshes the affected manager's packages.

### Remove (`x`)

Press `x` in the package detail view. Two new interfaces:

- `Remover` — implements `RemoveCmd(name string) *exec.Cmd` for basic package removal
- `DeepRemover` — implements `RemoveCmdWithDeps(name string) *exec.Cmd` for removal with orphaned dependency cleanup

The confirmation modal adapts to what the manager supports:

- **Basic** (brew, pip, npm, cargo, etc.) — shows the remove command, Yes/No buttons
- **With dependency warning** — if the package has `RequiredBy` entries, shows a yellow warning listing which packages depend on it before letting the user proceed
- **Deep remove** (apt, pacman, dnf, xbps) — adds a mode selector: "Remove package only" vs "Remove package + orphaned deps". When the deep remove mode is selected, the modal shows which deps will be removed and flags any that are still required by other installed packages

Same execution pattern as upgrade: password field for privileged managers, background goroutine, status bar notification, rescan after completion.

### Search + Install (`i`)

Press `i` from list view to enter a full-screen search view. Type a query and gpk searches all installed managers that implement the `Searcher` interface in parallel. Results stream in as each manager responds.

- `Searcher` — implements `Search(query string) ([]model.Package, error)` for querying available packages
- `Installer` — implements `InstallCmd(name string) *exec.Cmd` for installing a package

**Search results table:** deduplicated by package name, showing the highest stable version and its source. Already-installed packages are dimmed with a `✓` marker.

**Expandable rows:** press `Enter` or right arrow on a result to expand and see all available sources and versions for that package. Pick which source to install from.

**Pre-release toggle:** press `p` to toggle between stable-only and "include beta/pre-release" versions. State shows in the status bar.

**Managers with CLI search (first iteration):** brew, apt, pacman, dnf, npm, flatpak, snap, scoop, winget, chocolatey, pip (exact name only).

**Install flow:** select a package, confirmation modal shows the install command, password field for privileged managers, runs in background, rescan on completion.

### Multi Select (`m`)

Press `m` in list view to enter selection mode. `Space` toggles packages on/off with `☑`/`☐` markers. Selections persist across:

- Scrolling and pagination
- Tab switching between manager tabs
- Fuzzy search with `/` (search, toggle, clear search, search again — selections accumulate)

**Batch operations on selections:**

- `u` — upgrade all selected that have updates
- `x` — remove all selected (with dependency warnings for each)
- `m` or `Esc` — exit selection mode, clear selections

**Smart sudo batching:** the confirmation modal groups operations by privilege level:

```
privileged (1 password for all):
  apt: libssl-dev, curl
  snap: firefox

unprivileged (no password needed):
  brew: ripgrep
  pip: requests
```

One password entry covers all privileged commands. Privileged commands run in sequence under one sudo session. Unprivileged commands run in parallel since they don't share state. Results show what succeeded and what failed per package.

**Multi-select in search view:** same `Space` toggle in search results. `Enter` installs all selected.

## Future

### API-based search for managers without CLI search

Managers like cargo (crates.io API), gem (rubygems.org API), opam, conda, and luarocks don't have built-in CLI search or have limited search. Add HTTP-based searchers that query their public registries directly. All free, no auth needed.

### Version selection on install

When installing a package, allow the user to pick a specific version instead of always getting the latest. Useful when the latest version has breaking changes and the user needs an older stable release. The expanded search result row would show available versions as a scrollable list.

### Downgrade

Press a key in detail view to downgrade a package to a previous version. Shows available versions for the installed package and lets the user pick. Not all managers support this (brew does via `brew install pkg@version`, apt via `apt install pkg=version`, pip via `pip install pkg==version`). Managers that don't support it would show the standard "not supported" message.

## Themes

Only Tokyo Night right now. Shipping these built in:

| Theme | Vibe |
|-------|------|
| Tokyo Night | Current default |
| Catppuccin Mocha | Warm pastels, dark bg |
| Gruvbox Dark | Earthy, retro |
| Dracula | High contrast |
| Nord | Muted arctic |
| Solarized Dark | Classic |
| One Dark | Atom style |
| Rose Pine | Soft pinks |

A theme is just 12 color values matching the slots in `theme.go` (Base, Surface, Text, Subtext, Blue, Purple, Green, Red, Yellow, Cyan, Orange, White).

`t` cycles through themes live. Selection persists to `~/.local/share/glazepkg/theme.json`. Custom themes go in `~/.local/share/glazepkg/themes/` as JSON files with the same 12 values.

## Keybinds After All This Lands

| Key | Current | After |
|-----|---------|-------|
| `j`/`k` | Navigate | Navigate |
| `g`/`G` | Top / bottom | Top / bottom |
| `Ctrl+d`/`Ctrl+u` | Half page | Half page |
| `Tab`/`Shift+Tab` | Cycle tabs | Cycle tabs |
| `/` | Search | Search (also works in multi-select) |
| `Enter` | Details | Details / expand search result |
| `f` | Size filter | Size filter |
| `s` | Snapshot | Snapshot |
| `d` | Diff | Diff |
| `e` | Export | Export |
| `r` | Rescan | Rescan |
| `?` | Help | Help |
| `q` | Quit | Quit |
| `u` | **Upgrade (detail)** | **Upgrade (detail + multi-select)** |
| `x` | | **Remove (detail + multi-select)** |
| `m` | | **Multi select mode** |
| `i` | | **Search + install** |
| `p` | | **Toggle pre-release (search view)** |
| `space` | | **Toggle selection (multi-select)** |
| `t` | | Cycle theme |

## Build Order

1. **Themes** — most isolated change, only touches `theme.go` and persistence. Good first contribution.
2. ~~**Update**~~ — DONE. Command execution, confirmation modal, privilege handling, and background execution patterns are established.
3. **Remove** — same execution pattern as update, adds `Remover`/`DeepRemover` interfaces and dependency warnings.
4. **Search + Install** — full-screen search view, `Searcher`/`Installer` interfaces, parallel search, expandable results, pre-release toggle.
5. **Multi select** — UI layer on top of update, remove, and install. Smart sudo batching.
6. **Version selection + Downgrade** — version picker in search and detail views.

## Resolved Problems

### ~~Terminal Ownership~~ — RESOLVED

Solved by running commands in a background goroutine with `CombinedOutput()`. The TUI stays alive and interactive during upgrades. Sudo passwords are collected in the confirmation modal and piped to `sudo -S` via stdin. Commands use `exec.CommandContext` so they can be cancelled if the user force-quits with ctrl+c. Remove and install follow the same pattern.

### ~~Privilege Escalation~~ — RESOLVED

Handled via build-tag-split helpers: `privilegedCmd()` wraps with `sudo -S` on Unix (non-root), pass-through on Windows. Each manager declares its own elevation needs through `privilegedCmd` vs `exec.Command`. The confirmation modal shows a password field when sudo is needed, or an "elevated terminal" warning on Windows (chocolatey). Error output is parsed to extract meaningful messages and strip sudo prompts.

### ~~Cache Invalidation After Write Operations~~ — RESOLVED

Went with option 3: `UpdateCache.Invalidate(keys)` removes the affected manager's entries, then `rescanManager()` rescans just that one manager and merges the results back, preserving cached metadata (descriptions, deps, sizes) from previous entries.

## Open Problems

### Version Comparison Across Managers

Install results sort by latest version. But version strings aren't comparable across managers.

- brew: `1.14.1`
- pacman: `1.14.1-1` (pkgrel)
- apt: `1.14.1-2ubuntu1` (distro suffix)
- npm: `1.14.1` (but might be a totally different package with the same name)
- pip: `1.14.1.post1` or `1.14.1rc2` (PEP 440)

Stripping suffixes and comparing major.minor.patch gets most of it right. Fall back to string sort when parsing fails. Same package name across managers doesn't mean same software — don't deduplicate across managers, show everything, let the user pick.

### Flatpak and Snap Scope

Both have system vs user scope. A system installed Flatpak needs sudo to remove, a user installed one doesn't. gpk currently scans both but doesn't track scope. The scan data needs to include this before remove can work correctly for these managers.

## Not On This Roadmap

- **Config file for enabling/disabling managers.** gpk already skips managers that aren't installed. No need to add config complexity.
- **Plugin system for community managers.** The Manager interface needs to stabilize around the new Searcher/Installer/Remover methods first. Adding a plugin boundary now means breaking it later.
- **Systemd services, D-Bus, LDAP, enterprise features.** gpk is a user tool. Keeping it that way.

## Contributing

Check [CONTRIBUTING.md](CONTRIBUTING.md) for how the code is organized. Each feature above can be worked on independently. Themes don't touch package operations. Remove can ship before search. The search view can start with a single manager before wiring up the rest.

Open an issue or start a discussion if you want to grab something.
