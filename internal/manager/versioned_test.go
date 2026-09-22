package manager

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallVersionCmd(t *testing.T) {
	cases := []struct {
		mgr  VersionedInstaller
		name string
		ver  string
		want string
	}{
		{&Gem{}, "rails", "7.0.0", "gem install --user-install rails -v 7.0.0"},
		{&Npm{}, "typescript", "5.4.0", "npm install -g typescript@5.4.0"},
		{&Cargo{}, "ripgrep", "14.0.0", "cargo install ripgrep --version 14.0.0"},
		{&Go{}, "golang.org/x/tools/gopls", "v0.15.0", "go install golang.org/x/tools/gopls@v0.15.0"},
		{&Apt{}, "vim", "2:9.0", "apt-get install -y vim=2:9.0"},
	}
	for _, c := range cases {
		cmd := c.mgr.InstallVersionCmd(c.name, c.ver)
		got := strings.Join(cmd.Args, " ")
		// apt is wrapped in sudo; compare the tail.
		if !strings.HasSuffix(got, c.want) {
			t.Errorf("InstallVersionCmd(%q,%q) = %q, want suffix %q", c.name, c.ver, got, c.want)
		}
	}
}

// pip uses --user on a system Python (PEP 668) but not inside a venv/conda env.
func TestPipInstallVersionVenvAware(t *testing.T) {
	pip := &Pip{}
	t.Setenv("CONDA_PREFIX", "")
	t.Setenv("VIRTUAL_ENV", "")
	if got := strings.Join(pip.InstallVersionCmd("black", "24.1.0").Args, " "); got != "pip install --user black==24.1.0" {
		t.Errorf("system Python: got %q, want the --user form", got)
	}
	t.Setenv("VIRTUAL_ENV", "/tmp/venv")
	if got := strings.Join(pip.InstallVersionCmd("black", "24.1.0").Args, " "); got != "pip install black==24.1.0" {
		t.Errorf("in venv: got %q, want the plain form", got)
	}
}

func TestUpgradeAllCmd(t *testing.T) {
	cmd := (&Pacman{}).UpgradeAllCmd(true)
	got := strings.Join(cmd.Args, " ")
	if !strings.Contains(got, "pacman -Syu --noconfirm") {
		t.Errorf("pacman UpgradeAllCmd(yes) = %q", got)
	}
}

func TestCargoUpgradeAllNeedsCargoUpdate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	if cmd := (&Cargo{}).UpgradeAllCmd(true); cmd != nil {
		t.Fatalf("without cargo-install-update got %q, want nil", cmd.Args)
	}
	if err := os.WriteFile(filepath.Join(dir, "cargo-install-update"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := (&Cargo{}).UpgradeAllCmd(true)
	if cmd == nil {
		t.Fatal("with cargo-install-update got nil")
	}
	if got := strings.Join(cmd.Args, " "); got != "cargo install-update --all" {
		t.Errorf("cargo UpgradeAllCmd = %q", got)
	}
}

// A project pinning another pnpm via packageManager must not decide which
// global dir gpk operates on.
func TestPnpmRunsFromHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cmds := []*exec.Cmd{
		(&Pnpm{}).UpgradeAllCmd(true),
		(&Pnpm{}).UpgradeCmd("vite"),
		(&Pnpm{}).InstallCmd("vite"),
		(&Pnpm{}).RemoveCmd("vite"),
		(&Pnpm{}).InstallVersionCmd("vite", "8.0.0"),
	}
	for _, cmd := range cmds {
		if cmd.Dir != home {
			t.Errorf("%q runs in %q, want %q", cmd.Args, cmd.Dir, home)
		}
	}
}
