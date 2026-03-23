# GlazePKG Roadmap

gpk shows you everything installed on your system. The next step is letting you do something about it: update, remove, install, bulk-select, and pick a color scheme that doesn't hurt your eyes and doesn't look like a 2010 TUI.

This roadmap is an idea of what I want to do with gpk. It is not set in stone and is subject to change in the small details or the order in which things are shown here. I'm open to suggestions and ideas.

## Package Operations

### ~~Update (`u`)~~ — DONE

Press `u` in the package detail view. A confirmation modal shows the exact command that will run, with Yes/No buttons. Privileged managers show a sudo password field on Linux or an elevated terminal warning on Windows. Runs in the background with a notification while the TUI stays interactive. 19 managers support single-package upgrades via the `Upgrader` interface.

### ~~Remove (`x`)~~ — DONE

Press `x` in the package detail view. Managers that support deep remove (apt, pacman, dnf, xbps) offer a choice between removing the package only or removing it with orphaned dependencies. The modal warns when a package is required by others and flags dependency conflicts when deep remove is selected. 19 managers support removal via the `Remover` interface, 4 support deep remove via `DeepRemover`.

### ~~Search + Install (`i`)~~ — DONE

Press `i` from list view to open a full-screen search view. Searches run in parallel across all installed managers that implement `Searcher` (11 managers). Results are deduplicated by name with the highest version shown. Expand a row to see all sources. Install with confirmation modal. Descriptions carry over from search results after install.

### ~~Multi Select (`m`)~~ — DONE

Press `m` to toggle selection mode. `Space` selects packages. Selections persist across scrolling, tab switching, and fuzzy search. `u` batch upgrades all selected, `x` batch removes them. Operations run one at a time with per-package progress — failures don't stop the remaining packages. Smart sudo batching groups privileged and unprivileged operations, one password covers all privileged commands.

## Future

### Operation queue

Currently only one operation can run at a time. If a batch upgrade is running, you can't queue another operation until it finishes. A queue would let the user start new operations that wait for the current one to complete, with the ability to view and cancel queued items.

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

## Resolved Problems

### Terminal Ownership

Solved by running commands in a background goroutine with `CombinedOutput()`. The TUI stays alive and interactive during all operations. Sudo passwords are collected in the confirmation modal and piped to `sudo -S` via stdin. Commands use `exec.CommandContext` so they can be cancelled if the user force-quits with ctrl+c.

### Privilege Escalation

Handled via build-tag-split helpers: `privilegedCmd()` wraps with `sudo -S` on Unix (non-root), pass-through on Windows. Each manager declares its own elevation needs. The confirmation modal shows a password field when sudo is needed, or an "elevated terminal" warning on Windows.

### Cache Invalidation After Write Operations

`UpdateCache.Invalidate(keys)` removes the affected manager's entries, then `rescanManager()` rescans just that one manager and merges the results back, preserving cached metadata. Descriptions from search results are seeded into the cache so newly installed packages show their descriptions immediately.

## Open Problems

### Version Comparison Across Managers

Version strings aren't comparable across managers (brew `1.14.1`, pacman `1.14.1-1`, apt `1.14.1-2ubuntu1`, pip `1.14.1.post1`). Currently using rough semver parsing with string fallback.

### Flatpak and Snap Scope

Both have system vs user scope. A system installed Flatpak needs sudo to remove, a user installed one doesn't. gpk currently scans both but doesn't track scope.

## Not On This Roadmap

- **Config file for enabling/disabling managers.** gpk already skips managers that aren't installed.
- **Plugin system for community managers.** The Manager interface needs to stabilize first.
- **Systemd services, D-Bus, LDAP, enterprise features.** gpk is a user tool.

## Contributing

Check [CONTRIBUTING.md](CONTRIBUTING.md) for how the code is organized. Each feature above can be worked on independently. Open an issue or start a discussion if you want to grab something.
