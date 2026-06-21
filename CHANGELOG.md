# Changelog

Notable changes to gpk. Dates are roughly when work landed; the format loosely
follows Keep a Changelog.

## Unreleased

### Added
- Install plans preview the dependencies an install will pull in (pacman), so
  you see what comes along before confirming.
- `gpk managers` — a system overview of every supported manager, which ones are
  detected, and how many packages each holds (with `--json`).
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

- An unknown first argument is now treated as a search query rather than a hard
  error, matching yay.
