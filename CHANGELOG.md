# Changelog

Notable changes to gpk. Dates are roughly when work landed; the format loosely
follows Keep a Changelog.

## Unreleased

### Added
- When a name lives in several managers, the source picker (and the scripted
  "available in" message) now lists OS/system managers first, and pressing Enter
  installs the top one — so `gpk install ffmpeg` defaults to the system package,
  not a same-named library on PyPI/npm/crates.io.
- Manager preference: set `[install] prefer = ["aur", "brew"]` in the config and
  gpk resolves a package found in several managers to your top choice instead of
  asking.
- `gpk install <pkg> --pick-version` — choose the version interactively from a
  newest-first list, for managers that can install a specific version.
- `gpk export` / `gpk import` — back up the installed set to a file and restore
  it on another machine (cross-manager, brew-bundle style); import skips what's
  already present and previews the rest.
- `gpk theme [name]` — list the color themes with a live palette swatch, or set
  the active one; shared with the TUI via the config file.
- `gpk refresh` (and `-Sy`) rebuilds the scan cache from a fresh scan, gpk's
  take on syncing databases. A bare `-Sy` no longer errors.
- `--json` plan output on install/remove/upgrade/downgrade: prints the resolved
  steps (manager, name, version, exact command) without running anything, so a
  GUI or script can drive gpk as a backend.
- Shell completion: `gpk completion bash|zsh|fish` prints a script that
  tab-completes subcommands, manager names, and — from the scan cache —
  installed package names for remove/upgrade/info/downgrade/hold.
- Install plans preview the dependencies an install pulls in and the download /
  installed size (pacman), so you see exactly what's coming before confirming.
- `gpk managers` — a system overview of every supported manager: which are
  detected and how many packages each holds, drawn as a colored bar chart (and
  `--json`).
- Holds: `gpk hold <pkg>` pins a package so `gpk upgrade`/`-Syu` leave it alone
  (pacman gets a real `--ignore`); `gpk unhold` and `gpk holds` manage the list,
  and held packages drop out of `gpk outdated`.
- Action history and undo: every install/remove/upgrade/downgrade is logged;
  `gpk history` shows recent actions and `gpk undo` reverses the last one
  (install↔remove, and downgrade restores the prior version).

- pacman/yay short flags across the CLI: `-S`, `-Ss`, `-Si`, `-Syu`, `-Su`,
  `-Sc`/`-Scc`, `-R`, `-Rs`/`-Rns`, `-Q`, `-Qi`, `-Qs`, `-Qu`, `-Qdt`. They map
  onto the plain-word subcommands, so muscle memory from pacman/yay just works.
- `gpk search` — searches every installed manager in parallel, deduplicates by
  name, and shows the manager, version, description, and whether it's already
  installed. `--install`/`-i` turns the results into a numbered picker.
- `gpk <pkg>` with no subcommand searches and offers to install, the way
  `yay <pkg>` does.
- Smart install. When a name lives in several managers, gpk lists them with
  versions and lets you pick instead of bailing out; scripts still get exit 3.
  When a name isn't found anywhere, it suggests the closest matches.
- "Did you mean" for mistyped subcommands (`gpk instal` → `install`).
- Inline manager selectors: `--aur`, `--brew`, `--cask`, … as shorthand for
  `--manager <name>`, combinable with the existing `-m`.
- Version pinning on install: `gpk install black@24.1.0`, for the managers that
  support it (pip, pipx, uv, npm, pnpm, bun, go, gem, composer, conda, apt,
  cargo).
- `gpk upgrade` with no arguments (and `-Syu`) upgrades everything, running each
  manager's native bulk-upgrade command.
- `gpk downgrade <pkg>` — installs an earlier version, with a version picker
  populated from each manager (pacman reads its local cache and reinstalls via
  `pacman -U`; others list remote versions). Accepts `name@version` directly.
- `gpk clean` (`-Sc`/`-Scc`) clears cached downloads across managers.
- `gpk autoremove` (`-Qdt` to just list) removes orphaned dependencies.
- `gpk list <term>` filters the installed list by name or description.
- Themed, color-aware CLI output that follows the TUI palette, with plain text
  preserved for pipes, `NO_COLOR`, and non-terminals.
- Five more managers: AM/AppMan (AppImage), gvm, mise, Quicklisp, and macOS
  `softwareupdate` — bringing the total to 42.

### Changed
- Downgrade lists cached versions newest-first with a cross-format version
  comparator (epochs, pacman pkgrel, apt revisions, pip post-releases, tilde
  pre-releases, leading-v tags), following Debian's ordering.

- An unknown first argument is now treated as a search query rather than a hard
  error, matching yay.
- CLI commands parse tool output under a C locale, so field names read the same
  on non-English systems; interactive prompts still appear in your language.
- `gpk outdated` ends with a hint pointing at `gpk upgrade`, so the list leads
  straight into the action.
- Install resolution searches managers concurrently, so `gpk install <name>`
  without `--manager` returns in a fraction of the time on multi-manager hits.

### Fixed
- `install --manager <m>` works for managers that can install but not search
  (go, pipx, bun, …): an explicitly named manager is trusted instead of failing
  with "not found".
- `gpk` with no subcommand falls back to help when stdout isn't a terminal
  (pipes, scripts, CI) instead of failing with an opaque "could not open a new
  TTY" error.
- Self-update works on symlinked installs (Homebrew and similar), verifies the
  download against the release checksums, and reports "already up to date" as a
  success instead of an error.
- winget install/upgrade/remove match packages by their display name, fixing
  "no installed package found" for packages whose Id differs from their name.
- go upgrade resolves the module path (no more "malformed module path"), and go
  remove deletes the binary on Windows too.
- aur install/upgrade elevate with sudo when no AUR helper is installed.
- nuget is read-only: its global cache holds restored libraries, which can't be
  managed as `dotnet tool` entries.
- TUI: the U key updates every manager with a bulk upgrade; queued operations no
  longer leave a stale notification, a finished batch starts the next queued op,
  and a background removal no longer pulls you out of an unrelated detail view.
- State files (snapshots, notes, config) are written atomically, so an
  interrupted write can't truncate them into silent data loss.
