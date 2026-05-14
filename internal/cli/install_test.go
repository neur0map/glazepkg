package cli

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func installFakeMgrs() []manager.Manager {
	pacman := &fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			if q == "ripgrep" || q == "git" {
				return []model.Package{{Name: q, Source: model.SourcePacman}}, nil
			}
			return nil, nil
		},
		installCmdFn: func(name string) *exec.Cmd {
			// Use /bin/true so the test "install" succeeds without side effects.
			return exec.Command("/bin/true", "install", name)
		},
	}
	brew := &fakeManager{
		name: model.SourceBrew, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			if q == "ripgrep" {
				return []model.Package{{Name: q, Source: model.SourceBrew}}, nil
			}
			return nil, nil
		},
		installCmdFn: func(name string) *exec.Cmd {
			return exec.Command("/bin/true", "install", name)
		},
	}
	return []manager.Manager{pacman, brew}
}

func TestInstallNoArgs(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
}

func TestInstallAmbiguous(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "ripgrep"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitAmbiguous {
		t.Errorf("exit %d, want %d (stderr=%q)", code, ExitAmbiguous, errOut.String())
	}
	if !strings.Contains(errOut.String(), "pacman") || !strings.Contains(errOut.String(), "brew") {
		t.Errorf("stderr should list both managers: %q", errOut.String())
	}
}

func TestInstallNotFound(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "nope-does-not-exist"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Errorf("exit %d, want %d", code, ExitNegative)
	}
}

func TestInstallExplicitManagerYes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "ripgrep", "--manager", "brew", "--yes", "--quiet"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, want %d (stderr=%q)", code, ExitOK, errOut.String())
	}
}

func TestInstallDryRun(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "git", "--dry-run"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, want %d", code, ExitOK)
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Errorf("expected dry-run notice in stdout: %q", out.String())
	}
}

func TestInstallPromptYes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("y\n")
	code := Dispatch([]string{"install", "git", "--quiet"}, installFakeMgrs(), "test", &out, &errOut, stdin)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
}

func TestInstallPromptNo(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("n\n")
	code := Dispatch([]string{"install", "git"}, installFakeMgrs(), "test", &out, &errOut, stdin)
	if code != ExitOK {
		t.Fatalf("exit %d (cancellation should still be 0): stderr=%q", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "cancelled") {
		t.Errorf("stderr should mention cancelled: %q", errOut.String())
	}
}

func TestInstallHelpExitsZero(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "--help"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Errorf("exit %d, want %d", code, ExitOK)
	}
}
