package parsing

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestPacmanSearchParsing(t *testing.T) {
	// pacman -Ss kitty: headers may end with "[installed]" or
	// "[installed: <version>]" and can carry a "(group)" marker too.
	output := `extra/kitty 0.42.1-1 [installed]
    A modern, hackable, featureful, OpenGL-based terminal emulator
extra/kitty-shell-integration 0.42.1-1 (kitty-extras) [installed: 0.42.0-1]
    Shell integration scripts for kitty
extra/kitty-terminfo 0.42.1-1
    Terminfo for kitty, an OpenGL-based terminal emulator
aur/kitty-git 0.42.1.r10.gdeadbee-1
    Terminal emulator, built from git
`
	pkgs := manager.ParsePacmanSearch(output)
	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d", len(pkgs))
	}

	if pkgs[0].Name != "kitty" || pkgs[0].Version != "0.42.1-1" {
		t.Errorf("pkg 0: got %q %q", pkgs[0].Name, pkgs[0].Version)
	}
	if pkgs[0].Source != model.SourcePacman {
		t.Errorf("pkg 0 source = %q, want pacman", pkgs[0].Source)
	}
	if pkgs[0].Description != "A modern, hackable, featureful, OpenGL-based terminal emulator" {
		t.Errorf("pkg 0 desc = %q", pkgs[0].Description)
	}
	if !pkgs[0].Installed {
		t.Error("pkg 0: [installed] marker should set Installed")
	}

	if !pkgs[1].Installed {
		t.Error("pkg 1: [installed: <version>] marker should set Installed")
	}
	if pkgs[1].Version != "0.42.1-1" {
		t.Errorf("pkg 1 version = %q, marker must not leak into it", pkgs[1].Version)
	}

	if pkgs[2].Installed {
		t.Error("pkg 2: no marker, Installed should be false")
	}

	if pkgs[3].Source != model.SourceAUR {
		t.Errorf("pkg 3 source = %q, want aur", pkgs[3].Source)
	}
	if pkgs[3].Installed {
		t.Error("pkg 3: no marker, Installed should be false")
	}
}
