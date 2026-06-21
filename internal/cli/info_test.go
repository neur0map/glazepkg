package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestInfoFound(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"info", "--no-cache", "vim"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	body := out.String()
	if !strings.Contains(body, "vim") || !strings.Contains(body, "9.0") || !strings.Contains(body, "pacman") {
		t.Errorf("output missing expected fields: %s", body)
	}
}

func TestInfoNotFound(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"info", "--no-cache", "nonexistent"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Errorf("exit %d, want %d", code, ExitNegative)
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty on exit 2, got %q", out.String())
	}
}

func TestInfoMissingArg(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"info"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
}

func TestInfoTooManyArgs(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"info", "vim", "git"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
}

func TestInfoJSONFound(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"info", "--json", "--no-cache", "vim"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var env struct {
		Data cliPackage `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if env.Data.Name != "vim" || env.Data.Version != "9.0" {
		t.Errorf("data = %+v", env.Data)
	}
}

func TestInfoManagerFilterMiss(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// vim is pacman; restricting to brew should yield exit 2.
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"info", "--manager", "brew", "--no-cache", "vim"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Errorf("exit %d, want %d", code, ExitNegative)
	}
}

func TestInfoHelpExitsZero(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"info", "--help"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Errorf("exit %d, want %d (stderr=%q)", code, ExitOK, errOut.String())
	}
}

func TestInfoEnrichesInstalled(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fake := &fakeManager{
		name: model.SourcePacman, available: true,
		scanFn: func() ([]model.Package, error) {
			return []model.Package{fakePackage("foo", "1.0", model.SourcePacman)}, nil
		},
		describeFn: func(pkgs []model.Package) map[string]string {
			return map[string]string{"foo": "a foo package"}
		},
		depsFn: func(pkgs []model.Package) map[string][]string {
			return map[string][]string{"foo": {"libbar"}}
		},
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"info", "foo", "--manager", "pacman", "--no-cache"}, []manager.Manager{fake}, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	body := out.String()
	if !strings.Contains(body, "a foo package") {
		t.Errorf("info should show the fetched description:\n%s", body)
	}
	if !strings.Contains(body, "libbar") {
		t.Errorf("info should show the fetched dependencies:\n%s", body)
	}
}
