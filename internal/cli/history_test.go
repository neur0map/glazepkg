package cli

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
	"github.com/neur0map/glazepkg/internal/snapshot"
)

func TestInverseCmd(t *testing.T) {
	fake := &fakeManager{
		name: model.SourcePacman, available: true,
		removeCmdFn:  func(n string) *exec.Cmd { return exec.Command("remove", n) },
		installCmdFn: func(n string) *exec.Cmd { return exec.Command("install", n) },
	}

	// install is undone by removing.
	if cmd := inverseCmd(fake, snapshot.HistoryItem{Op: snapshot.OpInstall, Name: "vim"}, false); cmd == nil || cmd.Args[0] != "remove" {
		t.Errorf("install inverse should remove, got %v", cmd)
	}
	// remove is undone by installing.
	if cmd := inverseCmd(fake, snapshot.HistoryItem{Op: snapshot.OpRemove, Name: "vim"}, false); cmd == nil || cmd.Args[0] != "install" {
		t.Errorf("remove inverse should install, got %v", cmd)
	}
	// a plain upgrade can't be reversed.
	if cmd := inverseCmd(fake, snapshot.HistoryItem{Op: snapshot.OpUpgrade, Name: "vim"}, false); cmd != nil {
		t.Errorf("upgrade inverse should be nil, got %v", cmd)
	}
	// downgrade restores the previous version via a versioned install.
	cmd := inverseCmd(&manager.Pip{}, snapshot.HistoryItem{Op: snapshot.OpDowngrade, Name: "black", PrevVersion: "24.1.0"}, false)
	if cmd == nil || !strings.Contains(strings.Join(cmd.Args, " "), "black==24.1.0") {
		t.Errorf("downgrade inverse should reinstall the prior version, got %v", cmd)
	}
}

func TestUndoExecutesAndPops(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_ = snapshot.SaveHistory([]snapshot.HistoryItem{
		{Group: 1, Op: snapshot.OpInstall, Source: model.SourcePacman, Name: "foo"},
	})
	fake := &fakeManager{
		name: model.SourcePacman, available: true,
		removeCmdFn: func(n string) *exec.Cmd { return exec.Command("/bin/true", "remove", n) },
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"undo", "--yes", "--quiet"}, []manager.Manager{fake}, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	if h := snapshot.LoadHistory(); len(h) != 0 {
		t.Errorf("history should be empty after undo, got %+v", h)
	}
}
