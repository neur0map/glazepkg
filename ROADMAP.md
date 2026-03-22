# GlazePKG Roadmap

gpk shows you everything installed on your system. The next step is letting you do something about it: update, remove, install, bulk-select, and pick a color scheme that doesn't hurt your eyes and doesn't look like a 2010 TUI.

This roadmap is an idea of what I want to do with gpk. It is not set in stone and is subject to change in the small details or the order in which things are shown here. I'm open to suggestions and ideas.

## Package Operations

GPK is read-only right now. That needs to change.

### Update (`u`)

Hit `u` on a package with the `↑` indicator. gpk asks for confirmation, runs the update through the native manager (`brew upgrade`, `pacman -S`, `pip install --upgrade`, etc.), and refreshes the entry with the new version when it's done. If there's no update available, nothing happens.

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
| `u` | | Update package |
| `x` | | Remove package |
| `m` | | Multi select mode |
| `i` | | Install overlay |
| `t` | | Cycle theme |
| `space` | | Toggle selection (multi select) |

## Build Order

1. **Themes** — most isolated change, only touches `theme.go` and persistence. Good first contribution.
2. **Update** — introduces the command execution and confirmation patterns that remove and install reuse.
3. **Remove** — same execution pattern as update, adds destructive operation warnings.
4. **Multi select** — UI layer on top of update and remove already working.
5. **Install** — most complex. Needs `Searcher` interface, parallel search, results overlay, version display, and all the execution plumbing from above.

## Conflicts and Open Problems

Stuff that needs to be sorted out before or while building the above.

### Terminal Ownership

Bubble Tea owns the terminal. It eats all keyboard input and controls rendering. Running `sudo pacman -S ripgrep` needs stdin for the password prompt, which Bubble Tea is intercepting.

`tea.Exec` and `tea.ExecProcess` handle this. They release terminal control to the subprocess, let it run with full stdin/stdout, and hand back to gpk when the process exits. All package operations have to go through this path. Plain `exec.Command` while Bubble Tea is running will hang or produce garbage.

This means gpk's UI goes away while the command runs. The user sees raw output from their package manager, then gpk comes back. Might actually be better than an embedded output pane since people already know what their package manager's output looks like. Either way, this needs a deliberate call, not a surprise during implementation.

### Privilege Escalation

sudo handling varies wildly across managers and getting it wrong breaks things.

Need root:
- pacman, apt, dnf, apk, xbps, portage, snap
- nix (depends on single vs multi user install)
- guix (depends on system vs user profile)

Must never use root:
- brew (explicitly warns against it, breaks /usr/local or /opt/homebrew permissions)
- cargo, go, npm, pnpm, bun, pip, pipx, gem, composer, luarocks, opam (user space tools, sudo installs to wrong locations or creates root owned files in $HOME)

Depends on context:
- pip (system pip wants sudo, but --user or venv is almost always the right call)
- flatpak (system vs user scope)

Each manager needs to declare its own elevation policy. On Linux that's sudo. On macOS brew never needs it. On Windows it's UAC, completely different mechanism.

Also need to handle sudo failures cleanly: wrong password, timeout, permission denied. gpk has to show a clear error and not leave the list in a broken state.

### Cache Invalidation After Write Operations

Scan cache lasts 10 days. The second you install or remove something through gpk, that cache is stale. User removes ripgrep, ripgrep is still in the list until next rescan? That's broken.

Three options:
1. Nuke the whole cache after any write and force a full rescan. Simple, slow if doing multiple operations.
2. Patch the cache in place (delete entry on remove, add on install, bump version on update). Fast but fragile, two sources of truth.
3. Invalidate just the affected manager's portion of the cache and rescan that one manager.

Option 3 makes the most sense but the current `scan.json` is a flat list with one timestamp for everything. Needs per manager timestamps or a structural change.

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
