package cli

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestCleanNoSupport(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fake := &fakeManager{name: model.SourcePacman, available: true}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"clean"}, []manager.Manager{fake}, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Errorf("exit %d, want %d", code, ExitOK)
	}
	if !strings.Contains(errOut.String(), "cache cleaning") {
		t.Errorf("stderr = %q, want a no-cleaner notice", errOut.String())
	}
}

func TestCleanDryRun(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fake := &fakeManager{
		name: model.SourcePacman, available: true,
		cleanCacheFn: func(all, yes bool) *exec.Cmd { return exec.Command("/bin/true", "clean") },
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"clean", "--dry-run"}, []manager.Manager{fake}, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "pacman") {
		t.Errorf("plan should list pacman:\n%s", out.String())
	}
}
