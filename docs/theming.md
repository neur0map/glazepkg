# GlazePKG Theming

GlazePKG has a complete, production-ready theme system. Themes are TOML files
that define a full color palette. The active theme is stored in
`~/.glazepkg/config.toml` and applied on every launch.

---

## Table of Contents

1. [Bundled themes](#bundled-themes)
2. [Switching themes](#switching-themes)
3. [Writing a custom theme](#writing-a-custom-theme)
4. [Theme file format](#theme-file-format)
5. [Snapshots](#snapshots)
6. [TUI integration and keybindings](#tui-integration-and-keybindings)
7. [How the bundled theme system works](#how-the-bundled-theme-system-works)
8. [Rebuilding the binary with updated themes](#rebuilding-the-binary-with-updated-themes)

---

## Bundled themes

Seven themes ship inside the binary (no external files required):

| Name | Type | Origin |
|------|------|--------|
| Tokyo Night | dark | enkia |
| Catppuccin Mocha | dark | catppuccin |
| Dracula | dark | Zeno Rocha |
| Gruvbox Dark | dark | morhetz |
| Nord | dark | Arctic Ice Studio |
| One Dark | dark | Atom |
| Solarized Light | light | Ethan Schoonover |

---

## Switching themes

### TUI (interactive)

Press **`t`** from any view to open the theme picker overlay:

```
╭──────────────────────────────────────╮
│  Themes                              │
│  ────────────────────────────────── │
│    Catppuccin Mocha  [dark]          │
│    Dracula           [dark]          │
│  ▶ Gruvbox Dark      [dark]          │
│  ✓ Tokyo Night       [dark]          │
│    Nord              [dark]          │
│                                      │
│  j/k navigate · Enter select · Esc  │
╰──────────────────────────────────────╯
```

- `j` / `k` — move cursor up/down
- `Enter` — apply the highlighted theme and close
- `t` — cycle to the next theme live without closing
- `Esc` — close without changing

The chosen theme is saved to `~/.glazepkg/config.toml` immediately.

### CLI

```bash
# List all themes (active theme marked with ✓)
gpk themes

# Apply a theme by exact name and launch TUI
gpk --theme "Nord"
gpk --theme "Catppuccin Mocha"
```

---

## Writing a custom theme

Create a `.toml` file in `~/.glazepkg/themes/`. It is loaded at startup and
appears in the TUI picker and `gpk themes` listing alongside bundled themes.
User themes override bundled themes that share the same `name`.

```toml
name        = "My Theme"
type        = "dark"          # "dark" or "light" — used for UI badges
author      = "you"           # optional
description = "My own look"   # optional

# Core palette — all used by the TUI
background  = "#1e1e2e"
foreground  = "#cdd6f4"
accent      = "#cba6f7"       # tabs, titles, selected rows
cursor      = "#f5e0dc"
highlight   = "#b4befe"
border      = "#585b70"       # overlay borders
selection   = "#313244"       # overlay background tint
surface     = "#313244"       # raised surfaces
subtext     = "#6c7086"       # dim text, separators
text        = "#cdd6f4"       # normal body text

# Semantic colours — used for manager badges and diff indicators
blue        = "#89b4fa"
purple      = "#cba6f7"
green       = "#a6e3a1"
red         = "#f38ba8"
yellow      = "#f9e2af"
cyan        = "#89dceb"
orange      = "#fab387"
white       = "#bac2de"

# Any extra keys are accepted and accessible via theme.Color("key")
my_special  = "#ff00ff"
```

All keys are optional — missing ones fall back to `#888888`. The `name` field
is required; files without it are skipped with a warning in the log.

### Invalid files

TOML parse errors or missing `name` fields produce a log warning and are
silently skipped. All other themes continue to load normally.

---

## Theme file format

### Required

| Key | Description |
|-----|-------------|
| `name` | Display name used in the picker and CLI |

### Recommended palette keys

| Key | Used for |
|-----|---------|
| `background` | `ColorBase` — package row backgrounds, selected text fg |
| `foreground` | General readable text |
| `accent` | Titles, active tabs, selected row background, theme picker cursor |
| `cursor` | Spinner, cursors |
| `highlight` | Secondary accent |
| `border` | Overlay border color |
| `selection` | Overlay background tint |
| `surface` | `ColorSurface` — raised panels |
| `subtext` | `ColorSubtext` — separators, dim text, key hints |
| `text` | `ColorText` — normal body text |
| `blue` … `white` | Manager badge colours, diff indicators |

### Optional metadata

| Key | Type | Description |
|-----|------|-------------|
| `type` | `"dark"` / `"light"` | Shown as a badge in the picker |
| `author` | string | Displayed in `gpk themes` |
| `description` | string | Displayed in `gpk themes` |

---

## Snapshots

A theme snapshot captures the entire active palette to a timestamped TOML file
under `~/.glazepkg/theme_snapshots/`. Useful for preserving a state before
experimenting with a new theme.

### CLI

```bash
# Save current theme as a named snapshot
gpk snapshot save "before-redesign"

# List all saved snapshots (newest first)
gpk snapshot list

# Restore a snapshot (applies the theme and launches TUI)
gpk snapshot load ~/.glazepkg/theme_snapshots/before-redesign_20250101T120000.toml
```

### Snapshot file format

```toml
name         = "before-redesign"
saved_at     = 2025-01-01T12:00:00Z
source_theme = "Tokyo Night"

[theme]
name       = "Tokyo Night"
background = "#1a1b26"
# ... full palette
```

Snapshots are ordinary TOML files — edit them by hand if needed. They are
registered in the theme registry as `"<label> (snapshot)"` when loaded so
they appear in the picker like any other theme.

---

## TUI integration and keybindings

The theme system integrates with the TUI in three places:

### 1. Startup

`NewModel()` in `internal/ui/app.go` calls `theme.Load()` then
`ApplyTheme(theme.Active())`. This wires every `ColorX` and `StyleX`
package-level variable in `internal/ui/theme.go` to the palette before the
first render.

### 2. Live switching (`t` key)

Pressing `t` in any view opens the theme picker overlay. Navigating with
`j`/`k` and pressing `Enter` calls `theme.SetActive(name)` which:

1. Updates the in-memory registry pointer.
2. Writes `active_theme = "<name>"` to `~/.glazepkg/config.toml` atomically.
3. Calls `ApplyTheme(t)` to reassign all `ColorX`/`StyleX` vars.
4. Dispatches a `themeChangedMsg` so bubbletea triggers a full re-render on
   the next tick.

### 3. Keybinding summary

| Context | Key | Action |
|---------|-----|--------|
| List view | `t` | Open theme picker |
| Detail view | `t` | Open theme picker |
| Diff view | `t` | Open theme picker |
| Theme picker | `j` / `k` | Navigate |
| Theme picker | `Enter` | Apply and close |
| Theme picker | `t` | Cycle next and stay open |
| Theme picker | `Esc` | Close without applying |

---

## How the bundled theme system works

All `.toml` files under `internal/theme/themes/` are compiled into the
executable at build time using Go's `embed` package:

```go
//go:embed themes/*.toml
var bundledFS embed.FS
```

`theme.Load()` walks this embedded filesystem first, then merges any user
themes from `~/.glazepkg/themes/`. This means:

- The binary works with zero runtime files — no external theme directory needed.
- User themes in `~/.glazepkg/themes/` take precedence over bundled ones with
  the same `name`.
- Removing a user theme file reverts to the bundled version on next launch.

---

## Rebuilding the binary with updated themes

To add or change a **bundled** theme (one compiled into the binary):

1. Edit or add a `.toml` file in `internal/theme/themes/`.
2. Rebuild:

```bash
go build -o gpk ./cmd/gpk/
```

The `go:embed` directive automatically picks up any `*.toml` changes in that
directory. No other changes required.

To add a theme **without rebuilding** (user theme, hot-reloaded at startup):

```bash
mkdir -p ~/.glazepkg/themes/
cp mytheme.toml ~/.glazepkg/themes/
gpk          # theme is available immediately
```

---

## Rainbow title animation

The `GlazePKG` heading at the top of the TUI renders with a subtle,
continuously cycling color gradient. Each character is individually colored
from a curated 8-stop palette:

| Stop | Color | Role |
|------|-------|------|
| 1 | `#7aa2f7` | blue |
| 2 | `#7dcfff` | sky |
| 3 | `#73daca` | teal |
| 4 | `#9ece6a` | green |
| 5 | `#e0af68` | amber |
| 6 | `#ff9e64` | orange |
| 7 | `#f7768e` | rose |
| 8 | `#bb9af7` | purple |

The phase shifts one step every 5 spinner ticks — fast enough to be alive,
slow enough to never feel distracting. The palette is fixed regardless of the
active theme so the title always reads as distinct and premium against any
background.

The `accent` key in a theme controls all other interactive elements (tabs,
selected rows, badges, the spinner) but does **not** override the rainbow
palette.

### Centering

The title is mathematically centered on every render using the terminal width
stored in `m.width` (updated on every `tea.WindowSizeMsg`). This means it
stays perfectly centered when the window is resized, regardless of theme or
font.

The implementation uses `rainbowPalette` (a `[]lipgloss.Color` slice) in
`internal/ui/app.go`. Centering is computed against `titleVisualWidth = 12`
(the printable cell-count of `"✦ GlazePKG ✦"`). All layout math happens in
`renderTitleLine()` and is recalculated on every `tea.WindowSizeMsg`, so
resize is handled automatically.
