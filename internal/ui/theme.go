package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/config"
	"github.com/neur0map/glazepkg/internal/model"
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
		model.SourceMaven:          ColorOrange,
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

// RenderBadge returns a styled pill for a package source (used in detail view).
func RenderBadge(source model.Source) string {
	color, ok := ManagerColors[source]
	if !ok {
		color = ColorSubtext
	}
	return StyleBadge.
		Foreground(ColorBase).
		Background(color).
		Render(fmt.Sprintf(" %s ", source))
}

// RenderBadgeInline returns a colored source name without background (for inline use).
func RenderBadgeInline(source model.Source) string {
	color, ok := ManagerColors[source]
	if !ok {
		color = ColorSubtext
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(string(source))
}
