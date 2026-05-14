package cli

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func mgrSetWithUpdates() []manager.Manager {
	return []manager.Manager{
		&fakeManager{
			name: model.SourcePacman, available: true,
			scanFn: func() ([]model.Package, error) {
				return []model.Package{
					fakePackage("vim", "9.0", model.SourcePacman),
					fakePackage("git", "2.43", model.SourcePacman),
				}, nil
			},
			checkUpdatesFn: func(pkgs []model.Package) map[string]string {
				return map[string]string{"vim": "9.1"}
			},
		},
	}
}

func mgrSetNoUpdates() []manager.Manager {
	return []manager.Manager{
		&fakeManager{
			name: model.SourcePacman, available: true,
			scanFn: func() ([]model.Package, error) {
				return []model.Package{fakePackage("vim", "9.0", model.SourcePacman)}, nil
			},
			checkUpdatesFn: func(pkgs []model.Package) map[string]string { return nil },
		},
	}
}

func TestOutdatedCountFormat(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"outdated", "--count", "--no-cache"}, mgrSetWithUpdates(), "test", &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	if !regexp.MustCompile(`^[0-9]+\n$`).MatchString(out.String()) {
		t.Errorf("--count output = %q, want /^[0-9]+\\n$/", out.String())
	}
	if strings.TrimSpace(out.String()) != "1" {
		t.Errorf("count = %q, want '1'", strings.TrimSpace(out.String()))
	}
}

func TestOutdatedCountZero(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"outdated", "--count", "--no-cache"}, mgrSetNoUpdates(), "test", &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(out.String()) != "0" {
		t.Errorf("count = %q, want '0'", strings.TrimSpace(out.String()))
	}
}

func TestOutdatedExitCodeWithUpdates(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"outdated", "--exit-code", "--no-cache"}, mgrSetWithUpdates(), "test", &out, &errOut)
	if code != ExitNegative {
		t.Errorf("exit %d, want %d", code, ExitNegative)
	}
}

func TestOutdatedExitCodeNoUpdates(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"outdated", "--exit-code", "--no-cache"}, mgrSetNoUpdates(), "test", &out, &errOut)
	if code != ExitOK {
		t.Errorf("exit %d, want %d", code, ExitOK)
	}
}

func TestOutdatedJSON(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"outdated", "--json", "--no-cache"}, mgrSetWithUpdates(), "test", &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var env struct {
		Schema int `json:"schema"`
		Data   []struct {
			Name    string `json:"name"`
			Current string `json:"current"`
			Latest  string `json:"latest"`
			Source  string `json:"source"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if env.Schema != 1 {
		t.Errorf("schema = %d", env.Schema)
	}
	if len(env.Data) != 1 {
		t.Fatalf("data length = %d, want 1", len(env.Data))
	}
	e := env.Data[0]
	if e.Name != "vim" || e.Current != "9.0" || e.Latest != "9.1" || e.Source != "pacman" {
		t.Errorf("entry = %+v", e)
	}
}

func TestOutdatedJSONEmptyIsEmptyArray(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"outdated", "--json", "--no-cache"}, mgrSetNoUpdates(), "test", &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	// data must be [] not null, even when empty
	if !strings.Contains(out.String(), `"data":[]`) {
		t.Errorf("empty data should serialize as [], got: %s", out.String())
	}
}
