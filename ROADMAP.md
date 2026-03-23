# GlazePKG Roadmap

gpk shows you everything installed on your system. The next step is letting you do something about it: update, remove, install, bulk-select, and pick a color scheme that doesn't hurt your eyes and doesn't look like a 2010 TUI.

This roadmap is an idea of what I want to do with gpk. It is not set in stone and is subject to change in the small details or the order in which things are shown here. I'm open to suggestions and ideas.

## Package Operations

GPK is read-only right now. That needs to change.

### ~~Update (`u`)~~ — DONE

Press `u` in the package detail view. A confirmation modal shows the exact command that will run, with Yes/No buttons. Privileged managers (apt, dnf, pacman, snap, apk, xbps, chocolatey) show a sudo password field on Linux or an elevated terminal warning on Windows. The upgrade runs in the background with a status notification while the TUI stays interactive. 19 managers support single-package upgrades via the `Upgrader` interface. Post-upgrade rescan refreshes the affected manager's packages.

### Remove (`x`)

Hit `x` to remove a package. Uses `x` instead of `r` because `r` is already rescan and people are used to that. Confirmation prompt before anything runs. If the package has dependents, warn about it before proceeding.

### Multi Select (`m`)

Hit `m' to enter multi-select mode. `space` toggles packages on and off. `u` or `x` applies to everything selected, with one confirmation listing all of them. `m' again or `esc` clears selections and goes back to normal mode. Selections stick when switching tabs.

### Install (`i`)

Hit `i` to open a search overlay. Type a package name and gpk queries every available manager in parallel. Results come back in a table showing package name, version, source, and description. Latest versions sort to the top. Pick one, confirm, and gpk installs it.

Each manager that can search gets a `Searcher` interface method (same pattern as `Describer` and `UpdateChecker`). Managers that can't search just don't show up in results. Searches fire in parallel and results render as they arrive so it feels fast even when some managers are slow.

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
| `/` | Search | Search |
| `Enter` | Details | Details |
| `f` | Size filter | Size filter |
| `s` | Snapshot | Snapshot |
| `d` | Diff | Diff |
| `e` | Export | Export |
| `r` | Rescan | Rescan |
| `?` | Help | Help |
| `q` | Quit | Quit |
| `u` | **Upgrade package** | **Upgrade package** |
| `x` | | Remove package |
| `m` | | Multi select mode |
| `i` | | Install overlay |
| `t` | | Cycle theme |
| `space` | | Toggle selection (multi select) |

## Build Order

1. **Themes** — most isolated change, only touches `theme.go` and persistence. Good first contribution.
2. ~~**Update**~~ — DONE. Command execution, confirmation modal, privilege handling, and background execution patterns are established.
3. **Remove** — same execution pattern as update, adds destructive operation warnings.
4. **Multi select** — UI layer on top of update and remove already working.
5. **Install** — most complex. Needs `Searcher` interface, parallel search, results overlay, version display, and all the execution plumbing from above.

## Conflicts and Open Problems

Stuff that needs to be sorted out before or while building the above.

### ~~Terminal Ownership~~ — RESOLVED

Solved by running commands in a background goroutine with `CombinedOutput()`. The TUI stays alive and interactive during upgrades. Sudo passwords are collected in the confirmation modal and piped to `sudo -S` via stdin. Commands use `exec.CommandContext` so they can be cancelled if the user force-quits with ctrl+c. Remove and install should follow the same pattern.

### ~~Privilege Escalation~~ — RESOLVED

Handled via build-tag-split helpers: `privilegedCmd()` wraps with `sudo -S` on Unix (non-root), pass-through on Windows. Each manager declares its own elevation needs through `privilegedCmd` vs `exec.Command`. The confirmation modal shows a password field when sudo is needed, or an "elevated terminal" warning on Windows (chocolatey). Error output is parsed to extract meaningful messages and strip sudo prompts.

### ~~Cache Invalidation After Write Operations~~ — RESOLVED

Went with option 3: `UpdateCache.Invalidate(keys)` removes the affected manager's entries, then `rescanManager()` rescans just that one manager and merges the results back, preserving cached metadata (descriptions, deps, sizes) from previous entries.

### Manager Command Differences

No two package managers work the same way. Different flags, different output, different behavior for the same operation.

Update:
- `brew upgrade <pkg>`
- `sudo pacman -S <pkg>`
- `sudo apt install --only-upgrade <pkg>`
- `pip install --upgrade <pkg>`
- `cargo install <pkg>` (reinstalls latest)
- `npm update -g <pkg>`

Remove:
- `brew uninstall <pkg>`
- `sudo pacman -Rns <pkg>` vs `sudo pacman -R <pkg>` (with or without orphaned deps)
- `sudo apt remove <pkg>` vs `sudo apt purge <pkg>` (keep or remove config)
- `pip uninstall -y <pkg>` (needs -y or it blocks on a prompt)
- `cargo uninstall <pkg>`
- `npm uninstall -g <pkg>`

Search:
- `brew search <query>`
- `pacman -Ss <query>`
- `apt-cache search <query>`
- `npm search <query>`
- `pip index versions <pkg>` (exact name only, no fuzzy)
- cargo has no search command (crates.io API only)

Some managers prompt for confirmation by default (apt, pip uninstall). Since gpk already confirms, the manager's own prompt is redundant and will cause problems when Bubble Tea has released the terminal. Need to pass the right flags (-y, etc.) to suppress them per manager.

### Version Comparison Across Managers

Install results sort by latest version. But version strings aren't comparable across managers.

- brew: `1.14.1`
- pacman: `1.14.1-1` (pkgrel)
- apt: `1.14.1-2ubuntu1` (distro suffix)
- npm: `1.14.1` (but might be a totally different package with the same name)
- pip: `1.14.1.post1` or `1.14.1rc2` (PEP 440)

Stripping suffixes and comparing major.minor.patch gets most of it right. Fall back to string sort when parsing fails. Document the known edge cases.

Also: same package name across managers doesn't mean same software. `json` on npm and `json` on pip are completely different things. Don't deduplicate across managers. Show everything, let the user pick.

### Multi Select Across Managers

Selecting packages from three different managers and hitting `u` means three separate operations running in sequence. If brew succeeds and pacman fails, the user needs to see exactly what happened. Show what worked, what didn't, and why. Group operations by manager to minimize sudo prompts.

### Flatpak and Snap Scope

Both have system vs user scope. A system installed Flatpak needs sudo to remove, a user installed one doesn't. gpk currently scans both but doesn't track scope. The scan data needs to include this before remove or update can work correctly for these managers.

## Not On This Roadmap

- **Config file for enabling/disabling managers.** gpk already skips managers that aren't installed. No need to add config complexity.
- **Plugin system for community managers.** The Manager interface needs to stabilize around the new Searcher/Installer/Remover methods first. Adding a plugin boundary now means breaking it later.
- **Systemd services, D-Bus, LDAP, enterprise features.** gpk is a user tool. Keeping it that way.

## Contributing

Check [CONTRIBUTING.md](CONTRIBUTING.md) for how the code is organized. Each feature above can be worked on independently. Themes don't touch package operations. Update can ship before remove. The install overlay can start with a single manager before wiring up the rest.

Open an issue or start a discussion if you want to grab something.
