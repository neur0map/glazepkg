package cli

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func upgradeFakeMgrs() []manager.Manager {
	return []manager.Manager{
		&fakeManager{
			name: model.SourcePacman, available: true,
			scanFn: func() ([]model.Package, error) {
				return []model.Package{
					fakePackage("git", "2.43", model.SourcePacman),
					fakePackage("vim", "9.0", model.SourcePacman),
				}, nil
			},
			upgradeCmdFn: func(name string) *exec.Cmd {
				return exec.Command("/bin/true", "upgrade", name)
			},
		},
		&fakeManager{
			name: model.SourceBrew, available: true,
			scanFn: func() ([]model.Package, error) {
				return []model.Package{fakePackage("git", "2.43", model.SourceBrew)}, nil
			},
			upgradeCmdFn: func(name string) *exec.Cmd {
				return exec.Command("/bin/true", "upgrade", name)
			},
		},
	}
}

func TestUpgradeNoArgs(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"upgrade"}, upgradeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
}

func TestUpgradeNotInstalled(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"upgrade", "nonexistent", "--yes", "--quiet"}, upgradeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Errorf("exit %d, want %d", code, ExitNegative)
	}
}

func TestUpgradeAmbiguous(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// git is installed in both pacman and brew.
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"upgrade", "git", "--yes", "--quiet"}, upgradeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitAmbiguous {
		t.Errorf("exit %d, want %d (stderr=%q)", code, ExitAmbiguous, errOut.String())
	}
}

func TestUpgradeExplicitManagerYes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"upgrade", "git", "--manager", "brew", "--yes", "--quiet"}, upgradeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
}

func TestUpgradeDryRun(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"upgrade", "vim", "--dry-run"}, upgradeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Errorf("dry-run notice missing: %q", out.String())
	}
}

func TestUpgradePromptNo(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("n\n")
	code := Dispatch([]string{"upgrade", "vim"}, upgradeFakeMgrs(), "test", &out, &errOut, stdin)
	if code != ExitOK {
		t.Fatalf("cancellation exit %d", code)
	}
	if !strings.Contains(errOut.String(), "cancelled") {
		t.Errorf("expected 'cancelled': %q", errOut.String())
	}
}

func TestUpgradePromptYesExecutes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("y\n")
	code := Dispatch([]string{"upgrade", "vim", "--quiet"}, upgradeFakeMgrs(), "test", &out, &errOut, stdin)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
}

func TestUpgradeUnsupportedManager(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// Manager with scanFn returning a package but no upgradeCmdFn → not an Upgrader.
	noUpgrade := &fakeManager{
		name: model.SourcePacman, available: true,
		scanFn: func() ([]model.Package, error) {
			return []model.Package{fakePackage("hello", "1.0", model.SourcePacman)}, nil
		},
		// no upgradeCmdFn
	}
	// fakeManager implements UpgradeCmd via nil-guard returning nil — caught by
	// the "returned no upgrade command" check.
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"upgrade", "hello", "--yes", "--quiet"}, []manager.Manager{noUpgrade}, "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Errorf("exit %d, want %d (stderr=%q)", code, ExitErr, errOut.String())
	}
}

func TestUpgradeHelpExitsZero(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"upgrade", "--help"}, upgradeFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Errorf("exit %d, want %d", code, ExitOK)
	}
}

func TestUpgradeYesUsesNonInteractive(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var capturedNI string
	var capturedInteractive string
	pacman := &fakeManager{
		name: model.SourcePacman, available: true,
		scanFn: func() ([]model.Package, error) {
			return []model.Package{fakePackage("vim", "9.0", model.SourcePacman)}, nil
		},
		upgradeCmdFn: func(name string) *exec.Cmd {
			capturedInteractive = name
			return exec.Command("/bin/true", "interactive", name)
		},
		upgradeCmdYesFn: func(name string) *exec.Cmd {
			capturedNI = name
			return exec.Command("/bin/true", "noninteractive", name)
		},
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"upgrade", "vim", "--yes", "--quiet"}, []manager.Manager{pacman}, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	if capturedNI != "vim" {
		t.Errorf("expected non-interactive variant to be called with vim, got %q", capturedNI)
	}
	if capturedInteractive != "" {
		t.Errorf("expected interactive variant NOT to be called, but got %q", capturedInteractive)
	}
}
