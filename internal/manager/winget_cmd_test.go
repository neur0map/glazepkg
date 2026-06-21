package manager

import (
	"strings"
	"testing"
)

func TestWingetCommandsMatchByName(t *testing.T) {
	w := &Winget{}
	cases := map[string][]string{
		"upgrade": w.UpgradeCmd("Visual Studio Code").Args,
		"install": w.InstallCmd("Visual Studio Code").Args,
		"remove":  w.RemoveCmd("Visual Studio Code").Args,
	}
	for op, args := range cases {
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "--name Visual Studio Code") {
			t.Errorf("%s: expected --name with the package name, got %q", op, joined)
		}
		if strings.Contains(joined, "--id") {
			t.Errorf("%s: should match by --name, not --id, got %q", op, joined)
		}
	}
}
