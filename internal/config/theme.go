package config

import (
	"embed"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/lipgloss"
)

//go:embed themes/*.toml
var embeddedThemes embed.FS

// Palette holds the 12 base colors for a theme.
type Palette struct {
	Base    string `toml:"base"`
	Surface string `toml:"surface"`
	Text    string `toml:"text"`
	Subtext string `toml:"subtext"`
	Blue    string `toml:"blue"`
	Purple  string `toml:"purple"`
	Green   string `toml:"green"`
	Red     string `toml:"red"`
	Yellow  string `toml:"yellow"`
	Cyan    string `toml:"cyan"`
	Orange  string `toml:"orange"`
	White   string `toml:"white"`
}

// ManagerColors holds optional per-manager color overrides.
type ManagerColors map[string]string

// ThemeFile is the TOML structure for a theme file.
type ThemeFile struct {
	Name     string        `toml:"name"`
	Palette  Palette       `toml:"palette"`
	Managers ManagerColors `toml:"managers"`
}

// Theme is a resolved theme ready for use by the UI.
type Theme struct {
	ID       string // filename without extension (e.g. "tokyo-night")
	Name     string // display name (e.g. "Tokyo Night")
	Builtin  bool
	Palette  Palette
	Managers ManagerColors
}

// SystemPalette returns a palette using ANSI terminal colors.
func SystemPalette() Palette {
	return Palette{
		Base:    "0",
		Surface: "8",
		Text:    "7",
		Subtext: "8",
		Blue:    "4",
		Purple:  "5",
		Green:   "2",
		Red:     "1",
		Yellow:  "3",
		Cyan:    "6",
		Orange:  "3",
		White:   "15",
	}
}

// Color converts a palette color string to a lipgloss.Color.
func Color(c string) lipgloss.Color {
	return lipgloss.Color(c)
}

// loadBuiltinThemes loads all embedded theme TOML files.
func loadBuiltinThemes() []Theme {
	entries, err := embeddedThemes.ReadDir("themes")
	if err != nil {
		return nil
	}
	var themes []Theme
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		data, err := embeddedThemes.ReadFile("themes/" + e.Name())
		if err != nil {
			continue
		}
		var tf ThemeFile
		if err := toml.Unmarshal(data, &tf); err != nil {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".toml")
		themes = append(themes, Theme{
			ID:       id,
			Name:     tf.Name,
			Builtin:  true,
			Palette:  tf.Palette,
			Managers: tf.Managers,
		})
	}
	return themes
}

// loadUserThemes loads theme files from the user's themes directory.
func loadUserThemes() []Theme {
	dir := UserThemesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var themes []Theme
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var tf ThemeFile
		if err := toml.Unmarshal(data, &tf); err != nil {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".toml")
		name := tf.Name
		if name == "" {
			name = id
		}
		themes = append(themes, Theme{
			ID:       id,
			Name:     name,
			Builtin:  false,
			Palette:  tf.Palette,
			Managers: tf.Managers,
		})
	}
	return themes
}

// AllThemes returns all available themes: built-in first, then user themes.
// User themes with the same ID as a built-in override it.
func AllThemes() []Theme {
	builtins := loadBuiltinThemes()
	user := loadUserThemes()

	// Index builtins by ID
	byID := make(map[string]int, len(builtins))
	for i, t := range builtins {
		byID[t.ID] = i
	}

	// User themes override builtins with the same ID
	var extras []Theme
	for _, ut := range user {
		if idx, ok := byID[ut.ID]; ok {
			builtins[idx] = ut
			builtins[idx].Builtin = true // keep it in the builtin section
		} else {
			extras = append(extras, ut)
		}
	}

	sort.Slice(builtins, func(i, j int) bool {
		return builtins[i].Name < builtins[j].Name
	})
	sort.Slice(extras, func(i, j int) bool {
		return extras[i].Name < extras[j].Name
	})

	return append(builtins, extras...)
}

// ResolveTheme finds a theme by ID. Returns the system palette theme if
// id is "system", or falls back to tokyo-night if not found.
func ResolveTheme(id string) Theme {
	if id == "system" {
		return Theme{
			ID:      "system",
			Name:    "System (uses terminal colors)",
			Palette: SystemPalette(),
		}
	}

	for _, t := range AllThemes() {
		if t.ID == id {
			return t
		}
	}

	// Fallback to tokyo-night
	for _, t := range AllThemes() {
		if t.ID == "tokyo-night" {
			return t
		}
	}

	// Last resort
	return Theme{
		ID:      "system",
		Name:    "System (uses terminal colors)",
		Palette: SystemPalette(),
	}
}
