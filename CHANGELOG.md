# Changelog

Notable changes to gpk. Dates are roughly when work landed; the format loosely
follows Keep a Changelog.

## Unreleased

### Added
- `local` source (Linux) — detects and removes apps installed outside any
  package manager: GUI apps that drop an XDG `.desktop` entry (Zed, Discord,
  Termius…) and standalone CLI binaries dropped into `~/.local/bin` or
  `/usr/local/bin` by a `curl | sh` installer (Claude Code, omp…). Anything a
  system package manager owns, and any `#!` interpreter script (pip/pipx/npm
  entry points, shell wrappers), is deliberately skipped. Shows up as its own
  `local` tab in the TUI and under `gpk -m local`; `gpk -R <app>` removes the
  app's files — desktop entry, launcher, and self-contained install dir —
  leaving user config/data untouched.

## v0.6.0 — 2026-06-21

### Added
- Installing an AUR package without a helper shows its PKGBUILD for review
  before `makepkg` builds it, so you see exactly what runs (yay/paru style).
- `gpk versions <pkg>` — lists a package's installable versions per manager,
  newest-first via the version comparator (`--json` too), so a backend can build
  its own picker without driving the interactive `--pick-version` prompt.
- `gpk why <pkg>` — lists the installed packages that depend on one, answering
  "what needs this, is it safe to remove" (`brew uses` / `pacman -Qi` Required
  By). Reverse deps are derived generically by inverting dependency lists.
- `gpk -S install <pkg>` / `-R remove <pkg>` tolerate the redundant verb after
  the flag, so mixing the two forms does the obvious thing.
- `gpk info` on an installed package now fills in its description and
  dependencies (fetched and cached on demand), reading like `pacman -Qi` /
  `brew info` instead of just name/version/source.
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
- Every write command's run is themed end to end — a `::` progress line and a
  colored `✓`/`✗` per package (install/remove/upgrade/downgrade/clean/
  autoremove) — so a failed step reads clearly instead of trailing off after
  the tool's raw output.

### Changed
- The scan progress bar fills smoothly at 60fps (spring-animated) instead of
  jumping between per-manager completions.
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
- pip/gem/luarocks install to the user (`pip install --user` outside a venv,
  `gem install --user-install`, `luarocks install --local`) so they work on a
  stock system without sudo and without tripping pip's PEP 668 guard.
- `gpk upgrade` (bulk) no longer silently upgrades a held package on managers
  whose bulk command can't exclude it — it skips that manager with a warning
  instead of upgrading what you held (only pacman could honor it before).
- A bareword close to a short subcommand (`gpk less`, `gpk yay`, `gpk helm`) is
  now installed, not rejected as a "did you mean" typo; suggestions stay for
  real near-misses like `gpk instal`.
- `gpk install -- -dashpkg` keeps the `--` separator so dash-prefixed package
  names parse instead of erroring as unknown flags.
- `why --json` / `versions --json` return exit 2 when not installed / no
  versions, matching their human-mode exit codes.
- AUR works without yay/paru installed: search queries the AUR RPC directly,
  and install/upgrade build from source with `makepkg` — so `gpk search`,
  `gpk install <pkg> --aur`, and bare `gpk <aur-pkg>` find and install packages
  like `gpk-bin` on any box with base-devel, no helper required.
- A transient network failure while querying a manager (e.g. the AUR RPC
  timing out) now reports "couldn't reach <manager>" with exit 1 across the
  read commands (`search`, `install`, `info`, `versions`), instead of the
  misleading "not found"/"no versions" (exit 2). The AUR search retries once.
- `install --manager <m>` works for managers that can install but not search
  (go, pipx, bun, …): an explicitly named manager is trusted instead of failing
  with "not found".
- `gpk export --json` is accepted (export always emits the envelope), so the
  advertised `--json` flag now works uniformly on every read command.
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
