package cli

import (
	"os/exec"
	"testing"
)

func TestDisplayCmdAURBuild(t *testing.T) {
	build := exec.Command("sh", "-c", `set -e; git clone ...; makepkg -si`, "sh", "gpk-bin")
	if got, want := displayCmd(build), "makepkg -si gpk-bin  (build from AUR)"; got != want {
		t.Errorf("AUR build display = %q, want %q", got, want)
	}
	plain := exec.Command("pacman", "-S", "git")
	if got := displayCmd(plain); got != "pacman -S git" {
		t.Errorf("plain display = %q, want 'pacman -S git'", got)
	}
}
