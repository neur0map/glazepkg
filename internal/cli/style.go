package cli

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"

	"github.com/neur0map/glazepkg/internal/config"
	"github.com/neur0map/glazepkg/internal/model"
)

// styler renders human-facing output using the active gpk theme. When the
// output stream isn't a terminal (pipes, tests, NO_COLOR, TERM=dumb) every
// method returns plain text, so scripted and captured output stays stable.
type styler struct {
	on  bool
	pal config.Palette
	mgr map[model.Source]string
	r   *lipgloss.Renderer
}

// newStyler builds a styler bound to the saved theme. Color is enabled only
// when stdout is a real TTY and the environment doesn't opt out; when it is,
// a dedicated renderer forces a true-color profile so the theme's hex colors
// render exactly as they do in the TUI.
func newStyler() *styler {
	t := config.ResolveTheme(config.Load().Appearance.Theme)
	s := &styler{
		on:  colorEnabled(),
		pal: t.Palette,
		mgr: managerHexes(t.Palette),
		r:   lipgloss.DefaultRenderer(),
	}
	if s.on {
		s.r = lipgloss.NewRenderer(os.Stdout)
		s.r.SetColorProfile(termenv.TrueColor)
	}
	for name, hex := range t.Managers {
		s.mgr[model.Source(name)] = hex
	}
	return s
}

func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	return isatty.IsTerminal(os.Stdout.Fd())
}

func (s *styler) paint(text, hex string, bold bool) string {
	if !s.on || text == "" {
		return text
	}
	st := s.r.NewStyle().Foreground(lipgloss.Color(hex))
	if bold {
		st = st.Bold(true)
	}
	return st.Render(text)
}

func (s *styler) title(text string) string   { return s.paint(text, s.pal.Blue, true) }
func (s *styler) accent(text string) string  { return s.paint(text, s.pal.Cyan, true) }
func (s *styler) ok(text string) string      { return s.paint(text, s.pal.Green, false) }
func (s *styler) warn(text string) string    { return s.paint(text, s.pal.Yellow, false) }
func (s *styler) bad(text string) string     { return s.paint(text, s.pal.Red, false) }
func (s *styler) dim(text string) string     { return s.paint(text, s.pal.Subtext, false) }
func (s *styler) version(text string) string { return s.paint(text, s.pal.Green, false) }
func (s *styler) num(text string) string     { return s.paint(text, s.pal.Cyan, true) }

// mgrName colors a source with its badge color so managers read the same in
// the CLI as in the TUI.
func (s *styler) mgrName(src model.Source) string {
	hex, ok := s.mgr[src]
	if !ok {
		hex = s.pal.Subtext
	}
	return s.paint(string(src), hex, true)
}

// box wraps lines in a rounded border with a colored title. Falls back to a
// blank-line-padded block when color is off.
func (s *styler) box(title string, lines []string) string {
	body := strings.Join(lines, "\n")
	if !s.on {
		if title != "" {
			return title + "\n" + body
		}
		return body
	}
	inner := body
	if title != "" {
		inner = s.title(title) + "\n" + body
	}
	return s.r.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(s.pal.Surface)).
		Padding(0, 1).
		Render(inner)
}

// managerHexes maps each source to a palette hex, mirroring the TUI badge
// colors so both surfaces stay visually consistent.
func managerHexes(p config.Palette) map[model.Source]string {
	return map[model.Source]string{
		model.SourceBrew:           p.Yellow,
		model.SourceBrewCask:       p.Yellow,
		model.SourcePacman:         p.Blue,
		model.SourceAUR:            p.Cyan,
		model.SourceApt:            p.Green,
		model.SourceDnf:            p.Red,
		model.SourceSnap:           p.Orange,
		model.SourcePip:            p.Purple,
		model.SourcePipx:           p.Purple,
		model.SourceCargo:          p.Orange,
		model.SourceGo:             p.Cyan,
		model.SourceNpm:            p.Red,
		model.SourcePnpm:           p.White,
		model.SourceBun:            p.Yellow,
		model.SourceFlatpak:        p.Blue,
		model.SourceMacPorts:       p.Cyan,
		model.SourcePkgsrc:         p.Green,
		model.SourceOpam:           p.Orange,
		model.SourceGem:            p.Red,
		model.SourcePkg:            p.Blue,
		model.SourceComposer:       p.Purple,
		model.SourceMas:            p.Blue,
		model.SourceApk:            p.Cyan,
		model.SourceNix:            p.Blue,
		model.SourceConda:          p.Green,
		model.SourceLuarocks:       p.Blue,
		model.SourceXbps:           p.Green,
		model.SourcePortage:        p.Purple,
		model.SourceGuix:           p.Yellow,
		model.SourceWinget:         p.Cyan,
		model.SourceChocolatey:     p.Orange,
		model.SourceScoop:          p.Green,
		model.SourceNuget:          p.Purple,
		model.SourcePowerShell:     p.Blue,
		model.SourceWindowsUpdates: p.Red,
		model.SourceMaven:          p.Orange,
		model.SourceUv:             p.Purple,
		model.SourceLocal:          p.Green,
	}
}
