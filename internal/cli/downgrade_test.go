package cli

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestDowngradeNotInstalled(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fake := &fakeManager{name: model.SourcePacman, available: true,
		scanFn: func() ([]model.Package, error) { return nil, nil }}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"downgrade", "foo", "--manager", "pacman"}, []manager.Manager{fake}, "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Errorf("exit %d, want %d", code, ExitNegative)
	}
}

func TestDowngradeExplicitVersion(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fake := &fakeManager{
		name: model.SourcePacman, available: true,
		scanFn: func() ([]model.Package, error) {
			return []model.Package{fakePackage("foo", "2.0", model.SourcePacman)}, nil
		},
		installVersionFn: func(n, v string) *exec.Cmd { return exec.Command("/bin/true", n, v) },
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"downgrade", "foo@1.0", "--manager", "pacman", "--dry-run"}, []manager.Manager{fake}, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "1.0") {
		t.Errorf("plan should target version 1.0:\n%s", out.String())
	}
}
