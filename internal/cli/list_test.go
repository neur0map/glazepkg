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
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"list", "--json", "--no-cache"}, mgrSet(), "test", &out, &errOut, nil)
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
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"list", "--json", "--no-cache", "--manager", "brew"}, mgrSet(), "test", &out, &errOut, nil)
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
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"list", "--manager", "yum"}, mgrSet(), "test", &out, &errOut, nil)
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
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"list", "--no-cache"}, mgrSet(), "test", &out, &errOut, nil)
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

func TestListHelpExitsZero(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"list", "--help"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Errorf("exit %d, want %d (stderr=%q)", code, ExitOK, errOut.String())
	}
}

func TestListWithManagerFilterDoesNotOverwriteFullScanCache(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	// First call: full scan, populates cache with pacman+brew.
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"list", "--no-cache", "--quiet"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("first call exit %d (stderr=%q)", code, errOut.String())
	}

	// Second call: --manager pacman --no-cache. MUST NOT overwrite the cache.
	out.Reset()
	errOut.Reset()
	code = Dispatch([]string{"list", "--no-cache", "--quiet", "--manager", "pacman"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("filtered call exit %d (stderr=%q)", code, errOut.String())
	}

	// Third call: no flags, must hit cache and still see brew packages.
	out.Reset()
	errOut.Reset()
	code = Dispatch([]string{"list", "--quiet"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("third call exit %d (stderr=%q)", code, errOut.String())
	}
	if !strings.Contains(out.String(), "ripgrep") {
		t.Errorf("cache poisoned: brew packages dropped. stdout=%q", out.String())
	}
}
