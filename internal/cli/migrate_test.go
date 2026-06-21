package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestParseImport(t *testing.T) {
	if got := parseImport([]byte(`{"data":[{"name":"a","source":"pacman"}]}`)); len(got) != 1 || got[0].Name != "a" || got[0].Source != "pacman" {
		t.Errorf("envelope parse = %+v", got)
	}
	if got := parseImport([]byte(`[{"name":"b","source":"brew"}]`)); len(got) != 1 || got[0].Source != "brew" {
		t.Errorf("array parse = %+v", got)
	}
	got := parseImport([]byte("pacman/git\n# comment\nbrew:vim\nbare\n"))
	want := []importPkg{{Source: "pacman", Name: "git"}, {Source: "brew", Name: "vim"}, {Name: "bare"}}
	if len(got) != 3 {
		t.Fatalf("text parse len = %d, want 3: %+v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("text parse[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestImportSkipsInstalled(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fakes := []manager.Manager{
		&fakeManager{
			name: model.SourcePacman, available: true,
			scanFn: func() ([]model.Package, error) {
				return []model.Package{fakePackage("git", "2.43", model.SourcePacman)}, nil
			},
			installCmdFn: func(n string) *exec.Cmd { return exec.Command("/bin/true", "install", n) },
		},
	}
	file := filepath.Join(t.TempDir(), "pkgs.txt")
	if err := os.WriteFile(file, []byte("pacman/git\npacman/ripgrep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"import", file, "--dry-run"}, fakes, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "ripgrep") {
		t.Errorf("plan should include ripgrep:\n%s", out.String())
	}
	if strings.Contains(out.String(), "install git") {
		t.Errorf("git is installed and should be skipped:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "skipping 1") {
		t.Errorf("stderr should note one skip: %q", errOut.String())
	}
}
