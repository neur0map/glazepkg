package manager

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNormalizePipName(t *testing.T) {
	if got := normalizePipName("Flask_SQLAlchemy"); got != "flask-sqlalchemy" {
		t.Errorf("normalizePipName = %q", got)
	}
}

func TestParsePipNameSet(t *testing.T) {
	data := []byte(`[{"name":"Black","version":"24.3.0"},{"name":"requests_oauthlib","version":"1.0"}]`)
	set := parsePipNameSet(data)
	if !set["black"] || !set["requests-oauthlib"] {
		t.Errorf("set missing normalized entries: %v", set)
	}
	if parsePipNameSet([]byte("not json")) != nil {
		t.Error("expected nil on bad json")
	}
}

func TestPipScope(t *testing.T) {
	user := map[string]bool{"black": true}
	if pipScope("Black", user) != "user" {
		t.Error("Black should be user scope")
	}
	if pipScope("setuptools", user) != "global" {
		t.Error("setuptools should be global scope")
	}
	if pipScope("black", nil) != "" {
		t.Error("nil set should yield empty scope")
	}
}

// A Python that ships only pip3 (Homebrew, several distro packages) must still
// be detected and driven (#68).
func TestPipCmdFallsBackToPip3(t *testing.T) {
	dir := t.TempDir()
	stub := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	p := &Pip{}

	t.Setenv("PATH", dir)
	if p.Available() {
		t.Error("no pip at all should not be Available")
	}

	stub("pip3")
	if !p.Available() {
		t.Error("pip3 alone should be Available")
	}
	if got := p.pipCmd(); got != "pip3" {
		t.Errorf("pipCmd = %q, want pip3", got)
	}

	stub("pip")
	if got := p.pipCmd(); got != "pip" {
		t.Errorf("pipCmd = %q, want pip when both exist", got)
	}
}

// Every pip invocation must go through pipCmd, or a pip3-only box breaks on
// whichever call site was missed.
func TestPipCommandsUsePipCmd(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pip3"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_PREFIX", "")

	p := &Pip{}
	for name, cmd := range map[string]*exec.Cmd{
		"install": p.InstallCmd("black"),
		"upgrade": p.UpgradeCmd("black"),
		"remove":  p.RemoveCmd("black"),
	} {
		if cmd.Args[0] != "pip3" {
			t.Errorf("%s command = %v, want pip3 first", name, cmd.Args)
		}
	}
}
