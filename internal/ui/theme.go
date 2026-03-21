package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/model"
)

// Tokyo Night color palette
var (
	ColorBase    = lipgloss.Color("#1a1b26")
	ColorSurface = lipgloss.Color("#3b4261")
	ColorText    = lipgloss.Color("#a9b1d6")
	ColorSubtext = lipgloss.Color("#565f89")
	ColorBlue    = lipgloss.Color("#7aa2f7")
	ColorPurple  = lipgloss.Color("#bb9af7")
	ColorGreen   = lipgloss.Color("#9ece6a")
	ColorRed     = lipgloss.Color("#f7768e")
	ColorYellow  = lipgloss.Color("#e0af68")
	ColorCyan    = lipgloss.Color("#7dcfff")
	ColorOrange  = lipgloss.Color("#ff9e64")
	ColorWhite   = lipgloss.Color("#c6c6df")
)

var (
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

	StyleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1b26")).
			Background(ColorBlue).
			Bold(true)

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
)

// ManagerColors maps each source to its badge color.
var ManagerColors = map[model.Source]lipgloss.Color{
	model.SourceBrew:     ColorYellow,
	model.SourceBrewDeps: ColorSubtext,
	model.SourcePacman:  ColorBlue,
	model.SourceAUR:     ColorCyan,
	model.SourceApt:     ColorGreen,
	model.SourceDnf:     ColorRed,
	model.SourceSnap:    ColorOrange,
	model.SourcePip:     ColorPurple,
	model.SourcePipx:    ColorPurple,
	model.SourceCargo:   ColorOrange,
	model.SourceGo:      ColorCyan,
	model.SourceNpm:     ColorRed,
	model.SourcePnpm:    ColorWhite,
	model.SourceBun:     ColorYellow,
	model.SourceFlatpak:  ColorBlue,
	model.SourceMacPorts: ColorCyan,
	model.SourcePkgsrc:   ColorGreen,
	model.SourceOpam:     ColorOrange,
	model.SourceGem:      ColorRed,
	model.SourcePkg:      ColorBlue,
	model.SourceComposer: ColorPurple,
	model.SourceMas:      ColorBlue,
	model.SourceApk:      ColorCyan,
	model.SourceNix:      ColorBlue,
	model.SourceConda:    ColorGreen,
	model.SourceLuarocks: ColorBlue,
	// Windows managers
	model.SourceWinget:         ColorCyan,
	model.SourceChocolatey:     ColorOrange,
	model.SourceScoop:          ColorGreen,
	model.SourceNuget:          ColorPurple,
	model.SourcePowerShell:     ColorBlue,
	model.SourceWindowsUpdates: ColorRed,
}

// RenderBadge returns a styled pill for a package source (used in detail view).
func RenderBadge(source model.Source) string {
	color, ok := ManagerColors[source]
	if !ok {
		color = ColorSubtext
	}
	return StyleBadge.
		Foreground(lipgloss.Color("#1a1b26")).
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
