package cli

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestAutoremovePrintNone(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fake := &fakeManager{name: model.SourcePacman, available: true}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"autoremove", "--print"}, []manager.Manager{fake}, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Errorf("exit %d, want %d", code, ExitOK)
	}
	if !strings.Contains(out.String(), "no orphaned packages") {
		t.Errorf("stdout = %q, want a no-orphans notice", out.String())
	}
}

func TestAutoremovePrintsOrphans(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fake := &fakeManager{
		name: model.SourcePacman, available: true,
		orphansFn:       func() ([]string, error) { return []string{"orphan-lib"}, nil },
		removeOrphansFn: func(o []string, yes bool) *exec.Cmd { return exec.Command("/bin/true", "-Rns", "orphan-lib") },
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"autoremove", "--print"}, []manager.Manager{fake}, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "orphan-lib") {
		t.Errorf("print should list the orphan:\n%s", out.String())
	}
}

func TestAutoremoveDryRun(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fake := &fakeManager{
		name: model.SourcePacman, available: true,
		orphansFn:       func() ([]string, error) { return []string{"orphan-lib"}, nil },
		removeOrphansFn: func(o []string, yes bool) *exec.Cmd { return exec.Command("/bin/true", "-Rns", "orphan-lib") },
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"autoremove", "--dry-run"}, []manager.Manager{fake}, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "orphan-lib") {
		t.Errorf("plan should include the orphan:\n%s", out.String())
	}
}
