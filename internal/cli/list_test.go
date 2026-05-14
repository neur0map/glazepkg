package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func mgrSet() []manager.Manager {
	return []manager.Manager{
		&fakeManager{
			name: model.SourcePacman, available: true,
			scanFn: func() ([]model.Package, error) {
				return []model.Package{
					fakePackage("vim", "9.0", model.SourcePacman),
					fakePackage("git", "2.43", model.SourcePacman),
				}, nil
			},
		},
		&fakeManager{
			name: model.SourceBrew, available: true,
			scanFn: func() ([]model.Package, error) {
				return []model.Package{
					fakePackage("ripgrep", "14.0", model.SourceBrew),
				}, nil
			},
		},
		&fakeManager{
			name: model.SourceApk, available: false, // not on this system
		},
	}
}

func TestListJSON(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"list", "--json", "--no-cache"}, mgrSet(), "test", &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	var env struct {
		Schema int          `json:"schema"`
		Data   []cliPackage `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v\nbody=%s", err, out.String())
	}
	if env.Schema != 1 {
		t.Errorf("schema = %d, want 1", env.Schema)
	}
	if len(env.Data) != 3 {
		t.Errorf("data length = %d, want 3 (2 pacman + 1 brew)", len(env.Data))
	}
}

func TestListJSONFilterByManager(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"list", "--json", "--no-cache", "--manager", "brew"}, mgrSet(), "test", &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	var env struct {
		Data []cliPackage `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(env.Data) != 1 || env.Data[0].Name != "ripgrep" {
		t.Errorf("data = %+v, want [ripgrep]", env.Data)
	}
}

func TestListJSONUnknownManagerErrors(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"list", "--manager", "yum"}, mgrSet(), "test", &out, &errOut)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty on error", out.String())
	}
	if !strings.Contains(errOut.String(), "unknown manager") {
		t.Errorf("stderr = %q, want 'unknown manager'", errOut.String())
	}
}

func TestListHumanOutput(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"list", "--no-cache"}, mgrSet(), "test", &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	body := out.String()
	for _, want := range []string{"vim", "git", "ripgrep"} {
		if !strings.Contains(body, want) {
			t.Errorf("output missing %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "\x1b[") {
		t.Errorf("output contains ANSI escape codes (writer is not a TTY)")
	}
}
