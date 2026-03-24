# Theme System Design

## Overview

Add TOML-based theme support with a TUI theme picker. Users can switch between
built-in themes, drop in custom themes, or use "System" mode that inherits
terminal colors.

## File Layout

```
~/.config/glazepkg/
├── config.toml          # general app config
└── themes/              # user custom themes (optional)
    └── my-rice.toml
```

### config.toml

```toml
[appearance]
theme = "tokyo-night"    # or "system"
```

Namespaced under `[appearance]` so future settings get their own sections.

## Theme TOML Format

```toml
name = "catppuccin-mocha"

[palette]
base = "#1e1e2e"
surface = "#313244"
text = "#cdd6f4"
subtext = "#6c7086"
blue = "#89b4fa"
purple = "#cba6f7"
green = "#a6e3a1"
red = "#f38ba8"
yellow = "#f9e2af"
cyan = "#89dceb"
orange = "#fab387"
white = "#bac2de"

[managers]
brew = "#f9e2af"
cargo = "#fab387"
# omitted managers fall back to palette defaults
```

- `[palette]` — overrides the 12 base colors. Omitted keys keep defaults.
- `[managers]` — optional per-manager badge color overrides.
- `name` — display name shown in the theme picker.
- Hex `#rrggbb` format (standard for ricing).

## Built-in Themes (embedded in binary)

1. Tokyo Night (current default)
2. Catppuccin Mocha
3. Gruvbox Dark
4. Dracula
5. Nord

## System Mode

Special "System" option uses ANSI colors 0-15 so the user's terminal palette
takes over. Displayed as `System (uses terminal colors)` in the picker.

## Theme Picker UI

Keybind `t` opens an overlay:

```
  Theme

  ● System (uses terminal colors)
    Tokyo Night
    Catppuccin Mocha
    Gruvbox Dark
    Dracula
    Nord
    ─────────────
    my-rice
```

- `j`/`k` to navigate, `Enter` to apply, `Esc` to cancel
- Built-in themes first, separator, then user themes
- Live preview on cursor move
- `Enter` persists to config.toml, `Esc` reverts
- `●` marks currently active theme

## Implementation

### New dependency

- `github.com/BurntSushi/toml`

### New files

- `internal/config/config.go` — load/save `~/.config/glazepkg/config.toml`
- `internal/config/theme.go` — Theme struct, TOML parsing, embedded themes, merge logic
- `internal/config/themes/*.toml` — 5 embedded built-in theme files

### Changed files

- `internal/ui/theme.go` — palette variables become mutable, initialized from loaded theme
- `internal/ui/app.go` — load config on startup, add `t` keybind, theme picker overlay
- `internal/ui/keys.go` — register `t` binding

### Startup flow

1. Load `config.toml` → read `appearance.theme`
2. Resolve: check user `themes/` dir first, then embedded
3. If "system" → use ANSI colors
4. Apply palette to all style variables

### Theme switch flow

1. `t` pressed → build theme list (embedded + user), open overlay
2. Cursor move → apply theme live (preview)
3. `Enter` → persist to config.toml
4. `Esc` → revert to previous theme
