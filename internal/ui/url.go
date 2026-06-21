package ui

import (
	"net/url"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/neur0map/glazepkg/internal/model"
)

type openURLMsg struct{ err error }

// packageURL returns the most relevant web page for a package, preferring the
// registry or formula page (which shows versions, deps and install info) over
// the upstream homepage. Returns "" when no page is known for the source.
func packageURL(pkg model.Package) string {
	n := pkg.Name
	q := url.QueryEscape(n)
	switch pkg.Source {
	case model.SourceBrew:
		return "https://formulae.brew.sh/formula/" + n
	case model.SourceBrewCask:
		return "https://formulae.brew.sh/cask/" + n
	case model.SourceNpm, model.SourcePnpm, model.SourceBun:
		return "https://www.npmjs.com/package/" + n
	case model.SourceCargo:
		return "https://crates.io/crates/" + n
	case model.SourcePip, model.SourcePipx, model.SourceUv:
		return "https://pypi.org/project/" + n
	case model.SourceGo:
		return "https://pkg.go.dev/" + n
	case model.SourceGem:
		return "https://rubygems.org/gems/" + n
	case model.SourceAUR:
		return "https://aur.archlinux.org/packages/" + n
	case model.SourcePacman:
		return "https://archlinux.org/packages/?q=" + q
	case model.SourceFlatpak:
		return "https://flathub.org/apps/" + n
	case model.SourceSnap:
		return "https://snapcraft.io/" + n
	case model.SourceComposer:
		return "https://packagist.org/packages/" + n
	case model.SourceNuget:
		return "https://www.nuget.org/packages/" + n
	case model.SourceChocolatey:
		return "https://community.chocolatey.org/packages/" + n
	case model.SourceOpam:
		return "https://opam.ocaml.org/packages/" + n
	case model.SourceConda:
		return "https://anaconda.org/search?q=" + q
	case model.SourceScoop:
		return "https://scoop.sh/#/apps?q=" + q
	case model.SourceMaven:
		return "https://mvnrepository.com/search?q=" + q
	case model.SourceLuarocks:
		return "https://luarocks.org/search?q=" + q
	case model.SourceQuicklisp:
		return "https://quickdocs.org/" + n
	}
	return ""
}

// openURLForOS returns the platform command that opens url in the default browser.
func openURLForOS(goos, target string) *exec.Cmd {
	switch goos {
	case "darwin":
		return exec.Command("open", target)
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		return exec.Command("xdg-open", target)
	}
}

// openURLCmd opens target in the default browser without blocking the UI.
func openURLCmd(target string) tea.Cmd {
	return func() tea.Msg {
		return openURLMsg{err: openURLForOS(runtime.GOOS, target).Start()}
	}
}
