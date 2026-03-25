package ui

import (
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/config"
	"github.com/neur0map/glazepkg/internal/model"
	"github.com/neur0map/glazepkg/internal/theme"
)

// Color palette — mutable, set by ApplyTheme.
var (
	ColorBase    lipgloss.Color
	ColorSurface lipgloss.Color
	ColorText    lipgloss.Color
	ColorSubtext lipgloss.Color
	ColorBlue    lipgloss.Color
	ColorPurple  lipgloss.Color
	ColorGreen   lipgloss.Color
	ColorRed     lipgloss.Color
	ColorYellow  lipgloss.Color
	ColorCyan    lipgloss.Color
	ColorOrange  lipgloss.Color
	ColorWhite   lipgloss.Color
)

// Styles — rebuilt by ApplyTheme.
var (
	StyleTitle         lipgloss.Style
	StyleActiveTab     lipgloss.Style
	StyleInactiveTab   lipgloss.Style
	StyleFilterPrompt  lipgloss.Style
	StyleFilterText    lipgloss.Style
	StyleTableHeader   lipgloss.Style
	StyleSelected      lipgloss.Style
	StyleNormal        lipgloss.Style
	StyleDim           lipgloss.Style
	StyleAdded         lipgloss.Style
	StyleRemoved       lipgloss.Style
	StyleUpgrade       lipgloss.Style
	StyleStatusBar     lipgloss.Style
	StyleDetailKey     lipgloss.Style
	StyleDetailVal     lipgloss.Style
	StyleOverlay       lipgloss.Style
	StyleUpdateBanner  lipgloss.Style
	StyleOverlayTitle  lipgloss.Style
	StyleBadge         lipgloss.Style
)

// ─── Derived style variables ──────────────────────────────────────────────────
//
// Styles are rebuilt in ApplyTheme() so they always reflect the current palette.

var (
	StyleTitle        lipgloss.Style
	StyleActiveTab    lipgloss.Style
	StyleInactiveTab  lipgloss.Style
	StyleFilterPrompt lipgloss.Style
	StyleFilterText   lipgloss.Style
	StyleTableHeader  lipgloss.Style
	StyleSelected     lipgloss.Style
	StyleNormal       lipgloss.Style
	StyleDim          lipgloss.Style
	StyleAdded        lipgloss.Style
	StyleRemoved      lipgloss.Style
	StyleUpgrade      lipgloss.Style
	StyleStatusBar    lipgloss.Style
	StyleDetailKey    lipgloss.Style
	StyleDetailVal    lipgloss.Style
	StyleOverlay      lipgloss.Style
	StyleUpdateBanner lipgloss.Style
	StyleOverlayTitle lipgloss.Style
	StyleOverlayBase  lipgloss.Style
	StyleBadge        lipgloss.Style
)

// ApplyTheme assigns all color and style vars from the given Theme.
// Call at startup and whenever the user switches theme.
func ApplyTheme(t theme.Theme) {
	ColorBase = lipgloss.Color(t.Background)
	ColorSurface = lipgloss.Color(t.Surface)
	ColorText = lipgloss.Color(t.Text)
	ColorSubtext = lipgloss.Color(t.Subtext)
	ColorBlue = lipgloss.Color(t.Blue)
	ColorPurple = lipgloss.Color(t.Purple)
	ColorGreen = lipgloss.Color(t.Green)
	ColorRed = lipgloss.Color(t.Red)
	ColorYellow = lipgloss.Color(t.Yellow)
	ColorCyan = lipgloss.Color(t.Cyan)
	ColorOrange = lipgloss.Color(t.Orange)
	ColorWhite = lipgloss.Color(t.White)

	accent := lipgloss.Color(t.Accent)
	border := lipgloss.Color(t.Border)
	selection := lipgloss.Color(t.Selection)

	// Define a root style that sets the base background/foreground.
	base := lipgloss.NewStyle().Background(ColorBase).Foreground(ColorText)

	StyleTitle = base.Copy().Foreground(accent).Bold(true).Padding(0, 1)

	StyleActiveTab = base.Copy().
		Foreground(accent).Bold(true).Padding(0, 1).Underline(true)

	StyleInactiveTab = base.Copy().
		Foreground(ColorSubtext).Padding(0, 1)

	StyleFilterPrompt = base.Copy().Foreground(ColorCyan)
	StyleFilterText = base.Copy().Foreground(ColorText)

	StyleTableHeader = base.Copy().Foreground(ColorSubtext).Bold(true)

	StyleSelected = base.Copy().
		Foreground(badgeForeground(accent)).Background(accent).Bold(true)

	StyleNormal = base.Copy().Foreground(ColorText)
	StyleDim = base.Copy().Foreground(ColorSubtext)
	StyleAdded = base.Copy().Foreground(ColorGreen)
	StyleRemoved = base.Copy().Foreground(ColorRed)
	StyleUpgrade = base.Copy().Foreground(ColorYellow)
	StyleStatusBar = base.Copy().Foreground(ColorSubtext).Padding(0, 1)
	StyleDetailKey = base.Copy().Foreground(ColorSubtext).Width(18)
	StyleDetailVal = base.Copy().Foreground(ColorText)

	// Custom overlay background logic: ensure the modal contrast is sufficient.
	overlayBg := selection
	if r, ok := contrastRatio(selection, ColorText); ok && r < 3.0 {
		overlayBg = ColorSurface
		if r2, ok := contrastRatio(ColorSurface, ColorText); ok && r2 < 3.0 {
			overlayBg = ColorBase
		}
	}

	ColorOverlay = overlayBg
	StyleOverlayBase = lipgloss.NewStyle().Background(overlayBg).Foreground(ColorText)

	StyleOverlay = base.Copy().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Background(overlayBg).
		Padding(1, 2).
		Foreground(ColorText)

	StyleUpdateBanner = base.Copy().Foreground(ColorYellow).Bold(true)
	StyleOverlayTitle = base.Copy().Foreground(accent).Background(overlayBg).Bold(true)
	StyleBadge = lipgloss.NewStyle().Padding(0, 1).Bold(true) // Badge uses its own background/foreground

	// Theme-picker convenience alias + overlay row styles.
	ColorAccent = accent

	StyleThemeActive = lipgloss.NewStyle().
		Foreground(badgeForeground(accent)).
		Background(accent).
		Padding(0, 1).
		Bold(true)

	StyleThemeItem = StyleOverlayBase.Copy().
		Foreground(ColorText).
		Padding(0, 1)

	rebuildManagerColors()
}

// ─── Manager badge colors ─────────────────────────────────────────────────────

// ManagerColors maps each source to its badge color.
var ManagerColors map[model.Source]lipgloss.Color

// defaultManagerColorMap returns the default manager-to-palette-color mapping.
func defaultManagerColorMap() map[model.Source]lipgloss.Color {
	return map[model.Source]lipgloss.Color{
		model.SourceBrew:           ColorYellow,
		model.SourcePacman:         ColorBlue,
		model.SourceAUR:            ColorCyan,
		model.SourceApt:            ColorGreen,
		model.SourceDnf:            ColorRed,
		model.SourceSnap:           ColorOrange,
		model.SourcePip:            ColorPurple,
		model.SourcePipx:           ColorPurple,
		model.SourceCargo:          ColorOrange,
		model.SourceGo:             ColorCyan,
		model.SourceNpm:            ColorRed,
		model.SourcePnpm:           ColorWhite,
		model.SourceBun:            ColorYellow,
		model.SourceFlatpak:        ColorBlue,
		model.SourceMacPorts:       ColorCyan,
		model.SourcePkgsrc:         ColorGreen,
		model.SourceOpam:           ColorOrange,
		model.SourceGem:            ColorRed,
		model.SourcePkg:            ColorBlue,
		model.SourceComposer:       ColorPurple,
		model.SourceMas:            ColorBlue,
		model.SourceApk:            ColorCyan,
		model.SourceNix:            ColorBlue,
		model.SourceConda:          ColorGreen,
		model.SourceLuarocks:       ColorBlue,
		model.SourceXbps:           ColorGreen,
		model.SourcePortage:        ColorPurple,
		model.SourceGuix:           ColorYellow,
		model.SourceWinget:         ColorCyan,
		model.SourceChocolatey:     ColorOrange,
		model.SourceScoop:          ColorGreen,
		model.SourceNuget:          ColorPurple,
		model.SourcePowerShell:     ColorBlue,
		model.SourceWindowsUpdates: ColorRed,
	}
}

// ApplyTheme sets all palette colors, styles, and manager colors from a theme.
func ApplyTheme(t config.Theme) {
	p := t.Palette
	isSystem := t.ID == "system"

	ColorBase = config.Color(p.Base)
	ColorSurface = config.Color(p.Surface)
	ColorText = config.Color(p.Text)
	ColorSubtext = config.Color(p.Subtext)
	ColorBlue = config.Color(p.Blue)
	ColorPurple = config.Color(p.Purple)
	ColorGreen = config.Color(p.Green)
	ColorRed = config.Color(p.Red)
	ColorYellow = config.Color(p.Yellow)
	ColorCyan = config.Color(p.Cyan)
	ColorOrange = config.Color(p.Orange)
	ColorWhite = config.Color(p.White)

	// Rebuild styles
	StyleTitle = lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true).
		Padding(0, 1)

	StyleActiveTab = lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true).
		Padding(0, 1).
		Underline(true)

	StyleInactiveTab = lipgloss.NewStyle().
		Foreground(ColorSubtext).
		Padding(0, 1)

	StyleFilterPrompt = lipgloss.NewStyle().
		Foreground(ColorCyan)

	StyleFilterText = lipgloss.NewStyle().
		Foreground(ColorText)

	StyleTableHeader = lipgloss.NewStyle().
		Foreground(ColorSubtext).
		Bold(true)

	if isSystem {
		StyleSelected = lipgloss.NewStyle().
			Reverse(true).
			Bold(true)
	} else {
		StyleSelected = lipgloss.NewStyle().
			Foreground(ColorBase).
			Background(ColorBlue).
			Bold(true)
	}

	StyleNormal = lipgloss.NewStyle().
		Foreground(ColorText)

	StyleDim = lipgloss.NewStyle().
		Foreground(ColorSubtext)

	StyleAdded = lipgloss.NewStyle().
		Foreground(ColorGreen)

	StyleRemoved = lipgloss.NewStyle().
		Foreground(ColorRed)

	StyleUpgrade = lipgloss.NewStyle().
		Foreground(ColorYellow)

	StyleStatusBar = lipgloss.NewStyle().
		Foreground(ColorSubtext).
		Padding(0, 1)

	StyleDetailKey = lipgloss.NewStyle().
		Foreground(ColorSubtext).
		Width(18)

	StyleDetailVal = lipgloss.NewStyle().
		Foreground(ColorText)

	StyleOverlay = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSurface).
		Padding(1, 2).
		Foreground(ColorText)

	StyleUpdateBanner = lipgloss.NewStyle().
		Foreground(ColorYellow).
		Bold(true)

	StyleOverlayTitle = lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true)

	StyleBadge = lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)

	// Manager colors: start with defaults, then apply theme overrides
	ManagerColors = defaultManagerColorMap()
	if t.Managers != nil {
		for mgr, hex := range t.Managers {
			ManagerColors[model.Source(mgr)] = config.Color(hex)
		}
	}
}

func init() {
	// Apply Tokyo Night as default so the app works even without config loading.
	ApplyTheme(config.ResolveTheme("tokyo-night"))
}

// init seeds colors from the fallback theme so styles are never zero-value
// even if the caller forgets ApplyTheme().  NewModel() calls it properly.
func init() {
	ApplyTheme(theme.Active())
}

// ─── Badge helpers ────────────────────────────────────────────────────────────

// RenderBadge returns a styled pill for a package source (used in detail view).
func RenderBadge(source model.Source) string {
	color, ok := ManagerColors[source]
	if !ok {
		color = ColorSubtext
	}
	fg := badgeForeground(color)
	return StyleBadge.
		Foreground(ColorBase).
		Background(color).
		Render("  " + string(source) + "  ")
}

// RenderBadgeInline returns a colored source name without background.
func RenderBadgeInline(source model.Source) string {
	color, ok := ManagerColors[source]
	if !ok {
		color = ColorSubtext
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(string(source))
}

// ─── Theme-picker specific styles ─────────────────────────────────────────────

// ColorAccent mirrors the theme Accent color for convenience.
var ColorAccent lipgloss.Color

// ColorOverlay is the background color used for modals/overlays.
var ColorOverlay lipgloss.Color

// StyleThemeActive highlights the currently-active theme in the picker.
var StyleThemeActive lipgloss.Style

// StyleThemeItem is used for non-selected rows in the theme picker.
var StyleThemeItem lipgloss.Style

// badgeForeground picks the most legible text color for a colored badge.
// It chooses between ColorText and ColorBase based on contrast ratio.
func badgeForeground(bg lipgloss.Color) lipgloss.Color {
	return bestContrast(bg, ColorText, ColorBase)
}

func bestContrast(bg, a, b lipgloss.Color) lipgloss.Color {
	ra, oka := contrastRatio(bg, a)
	rb, okb := contrastRatio(bg, b)
	switch {
	case oka && okb:
		if ra >= rb {
			return a
		}
		return b
	case oka:
		return a
	case okb:
		return b
	default:
		return a
	}
}

func contrastRatio(bg, fg lipgloss.Color) (float64, bool) {
	bl, ok := relativeLuminance(string(bg))
	if !ok {
		return 0, false
	}
	fl, ok := relativeLuminance(string(fg))
	if !ok {
		return 0, false
	}
	if bl < fl {
		bl, fl = fl, bl
	}
	return (bl + 0.05) / (fl + 0.05), true
}

func relativeLuminance(hex string) (float64, bool) {
	r, g, b, ok := parseHexColor(hex)
	if !ok {
		return 0, false
	}
	return 0.2126*srgbToLinear(r) + 0.7152*srgbToLinear(g) + 0.0722*srgbToLinear(b), true
}

func srgbToLinear(c float64) float64 {
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func parseHexColor(s string) (float64, float64, float64, bool) {
	if s == "" {
		return 0, 0, 0, false
	}
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) == 8 {
		s = s[:6]
	}
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	r := float64((v>>16)&0xff) / 255.0
	g := float64((v>>8)&0xff) / 255.0
	b := float64(v&0xff) / 255.0
	return r, g, b, true
}
