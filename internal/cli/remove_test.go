package cli

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func removeFakeMgrs() []manager.Manager {
	return []manager.Manager{
		&fakeManager{
			name: model.SourcePacman, available: true,
			scanFn: func() ([]model.Package, error) {
				return []model.Package{
					fakePackage("git", "2.43", model.SourcePacman),
					fakePackage("hello", "1.0", model.SourcePacman),
				}, nil
			},
			removeCmdFn: func(name string) *exec.Cmd {
				return exec.Command("/bin/true", "remove", name)
			},
		},
		&fakeManager{
			name: model.SourceBrew, available: true,
			scanFn: func() ([]model.Package, error) {
				return []model.Package{fakePackage("git", "2.43", model.SourceBrew)}, nil
			},
			removeCmdFn: func(name string) *exec.Cmd {
				return exec.Command("/bin/true", "uninstall", name)
			},
		},
	}
}

func TestRemoveNoArgs(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove"}, removeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
}

func TestRemoveNotInstalled(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "nonexistent", "--yes", "--quiet"}, removeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Errorf("exit %d, want %d", code, ExitNegative)
	}
}

func TestRemoveAmbiguous(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// git is installed in both pacman and brew.
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "git", "--yes", "--quiet"}, removeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitAmbiguous {
		t.Errorf("exit %d, want %d (stderr=%q)", code, ExitAmbiguous, errOut.String())
	}
}

func TestRemoveExplicitManagerYes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "git", "--manager", "brew", "--yes", "--quiet"}, removeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
}

func TestRemoveDryRun(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "hello", "--dry-run"}, removeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Errorf("dry-run notice missing: %q", out.String())
	}
}

func TestRemovePromptNo(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("n\n")
	code := Dispatch([]string{"remove", "hello"}, removeFakeMgrs(), "test", &out, &errOut, stdin)
	if code != ExitOK {
		t.Fatalf("cancellation exit %d", code)
	}
	if !strings.Contains(errOut.String(), "cancelled") {
		t.Errorf("expected 'cancelled': %q", errOut.String())
	}
}

func TestRemoveWithDepsUnsupported(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// fakeManager doesn't implement DeepRemover (no removeCmdWithDepsFn field).
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "hello", "--with-deps", "--manager", "pacman", "--yes", "--quiet"}, removeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
	if !strings.Contains(errOut.String(), "with-deps") && !strings.Contains(errOut.String(), "does not support") {
		t.Errorf("expected --with-deps error: %q", errOut.String())
	}
}

func TestRemoveHelpExitsZero(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "--help"}, removeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Errorf("exit %d, want %d", code, ExitOK)
	}
}

func TestRemoveRequiredByWarning(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	pacman := &fakeManager{
		name: model.SourcePacman, available: true,
		scanFn: func() ([]model.Package, error) {
			p := fakePackage("foo", "1.0", model.SourcePacman)
			p.RequiredBy = []string{"bar", "baz"}
			return []model.Package{p}, nil
		},
		removeCmdFn: func(name string) *exec.Cmd {
			return exec.Command("/bin/true", name)
		},
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "foo", "--dry-run"}, []manager.Manager{pacman}, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "required by") {
		t.Errorf("expected required-by warning in stdout: %q", out.String())
	}
}
