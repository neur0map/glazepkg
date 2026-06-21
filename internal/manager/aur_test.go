package manager

import (
	"strings"
	"testing"
)

func TestAURBuildCmd(t *testing.T) {
	c := aurBuildCmd("gpk-bin", false)
	if len(c.Args) < 4 || c.Args[0] != "sh" || c.Args[1] != "-c" {
		t.Fatalf("expected `sh -c <script> ...`, got %v", c.Args)
	}
	if c.Args[len(c.Args)-1] != "gpk-bin" {
		t.Errorf("name should be the trailing positional arg, got %v", c.Args)
	}
	script := c.Args[2]
	if !strings.Contains(script, "git clone") || !strings.Contains(script, "makepkg -si") {
		t.Errorf("script missing clone/makepkg: %q", script)
	}
	if !strings.Contains(script, "command -v makepkg") {
		t.Errorf("script should preflight-check makepkg: %q", script)
	}
	if strings.Contains(script, "gpk-bin") {
		t.Error("name must be passed positionally, never interpolated into the script")
	}
	if strings.Contains(script, "--noconfirm") {
		t.Error("non-yes build must not pass --noconfirm")
	}
	if !strings.Contains(aurBuildCmd("foo", true).Args[2], "--noconfirm") {
		t.Error("yes build should pass --noconfirm")
	}
}
