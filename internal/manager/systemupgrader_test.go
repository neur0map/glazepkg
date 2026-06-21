package manager

import (
	"strings"
	"testing"
)

func TestSystemUpgradeCmds(t *testing.T) {
	cases := []struct {
		mgr  SystemUpgrader
		want string
	}{
		{&Pacman{}, "pacman -Syu --noconfirm"},
		{&Dnf{}, "dnf upgrade -y"},
		{&Apt{}, "apt-get update && apt-get upgrade -y"},
		{&Brew{}, "brew update && brew upgrade"},
	}
	for _, c := range cases {
		got := strings.Join(c.mgr.SystemUpgradeCmd().Args, " ")
		if !strings.Contains(got, c.want) {
			t.Errorf("%T system upgrade = %q, want it to contain %q", c.mgr, got, c.want)
		}
	}
}
