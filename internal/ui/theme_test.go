package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/theme"
)

// TestApplyTheme_ColorsUpdated verifies that ApplyTheme actually rewires the
// package-level color vars so every downstream style sees the new palette.
func TestApplyTheme_ColorsUpdated(t *testing.T) {
	if err := theme.Load(); err != nil {
		t.Fatalf("theme.Load: %v", err)
	}

	customTheme := theme.Theme{
		Name:       "Test Theme",
		Background: "#aabbcc",
		Foreground: "#112233",
		Accent:     "#ff0000",
		Surface:    "#001122",
		Subtext:    "#334455",
		Text:       "#556677",
		Blue:       "#0000ff",
		Purple:     "#aa00aa",
		Green:      "#00ff00",
		Red:        "#ff0000",
		Yellow:     "#ffff00",
		Cyan:       "#00ffff",
		Orange:     "#ff8800",
		White:      "#ffffff",
		Border:     "#998877",
		Selection:  "#223344",
		Cursor:     "#aabbcc",
		Highlight:  "#bbccdd",
	}

	ApplyTheme(customTheme)

	if ColorBase != lipgloss.Color("#aabbcc") {
		t.Errorf("ColorBase = %q, want #aabbcc", ColorBase)
	}
	if ColorBlue != lipgloss.Color("#0000ff") {
		t.Errorf("ColorBlue = %q, want #0000ff", ColorBlue)
	}
	if ColorAccent != lipgloss.Color("#ff0000") {
		t.Errorf("ColorAccent = %q, want #ff0000", ColorAccent)
	}
}

// TestApplyTheme_StylesNonNil ensures that every exported style var is populated
// after ApplyTheme runs — i.e. no nil/zero-value styles that would render as
// plain text.
func TestApplyTheme_StylesNonNil(t *testing.T) {
	if err := theme.Load(); err != nil {
		t.Fatalf("theme.Load: %v", err)
	}
	ApplyTheme(theme.Active())

	styles := map[string]lipgloss.Style{
		"StyleTitle":        StyleTitle,
		"StyleActiveTab":    StyleActiveTab,
		"StyleInactiveTab":  StyleInactiveTab,
		"StyleSelected":     StyleSelected,
		"StyleNormal":       StyleNormal,
		"StyleDim":          StyleDim,
		"StyleAdded":        StyleAdded,
		"StyleRemoved":      StyleRemoved,
		"StyleUpgrade":      StyleUpgrade,
		"StyleOverlay":      StyleOverlay,
		"StyleOverlayTitle": StyleOverlayTitle,
		"StyleBadge":        StyleBadge,
		"StyleThemeActive":  StyleThemeActive,
		"StyleThemeItem":    StyleThemeItem,
	}

	// All we can practically check is that Render doesn't panic.
	for name, s := range styles {
		result := s.Render("test")
		if result == "" {
			t.Errorf("%s.Render returned empty string", name)
		}
	}
}

// TestManagerColorsPopulated confirms that ManagerColors is non-nil and
// contains entries after ApplyTheme.
func TestManagerColorsPopulated(t *testing.T) {
	if err := theme.Load(); err != nil {
		t.Fatalf("theme.Load: %v", err)
	}
	ApplyTheme(theme.Active())

	if len(ManagerColors) == 0 {
		t.Error("ManagerColors is empty after ApplyTheme")
	}
}

// TestApplyTheme_LightThemeBackground verifies that a light theme's background
// color is correctly applied (not forced to a hard-coded dark value).
func TestApplyTheme_LightThemeBackground(t *testing.T) {
	light := theme.Theme{
		Name:       "Light",
		Type:       "light",
		Background: "#fdf6e3",
		Foreground: "#657b83",
		Accent:     "#268bd2",
		Surface:    "#eee8d5",
		Subtext:    "#93a1a1",
		Text:       "#657b83",
		Blue:       "#268bd2",
		Purple:     "#6c71c4",
		Green:      "#859900",
		Red:        "#dc322f",
		Yellow:     "#b58900",
		Cyan:       "#2aa198",
		Orange:     "#cb4b16",
		White:      "#fdf6e3",
		Border:     "#93a1a1",
		Selection:  "#eee8d5",
		Cursor:     "#cb4b16",
		Highlight:  "#6c71c4",
	}

	ApplyTheme(light)

	if ColorBase != lipgloss.Color("#fdf6e3") {
		t.Errorf("light theme ColorBase = %q, want #fdf6e3", ColorBase)
	}
}
