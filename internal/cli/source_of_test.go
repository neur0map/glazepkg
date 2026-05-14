package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSourceOfFound(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"source-of", "--no-cache", "vim"}, mgrSet(), "test", &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	if strings.TrimSpace(out.String()) != "pacman" {
		t.Errorf("stdout = %q, want 'pacman'", out.String())
	}
}

func TestSourceOfNotFound(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"source-of", "--no-cache", "nonexistent"}, mgrSet(), "test", &out, &errOut)
	if code != ExitNegative {
		t.Errorf("exit %d, want %d", code, ExitNegative)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
}

func TestSourceOfMissingArg(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"source-of"}, mgrSet(), "test", &out, &errOut)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
}

func TestSourceOfTooManyArgs(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"source-of", "vim", "git"}, mgrSet(), "test", &out, &errOut)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
}

func TestSourceOfJSONFound(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"source-of", "--json", "--no-cache", "vim"}, mgrSet(), "test", &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var env struct {
		Data struct {
			Name    string   `json:"name"`
			Sources []string `json:"sources"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if env.Data.Name != "vim" || len(env.Data.Sources) != 1 || env.Data.Sources[0] != "pacman" {
		t.Errorf("data = %+v", env.Data)
	}
}

func TestSourceOfJSONNotFoundEmitsNothing(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"source-of", "--json", "--no-cache", "nonexistent"}, mgrSet(), "test", &out, &errOut)
	if code != ExitNegative {
		t.Fatalf("exit %d", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty on exit 2, got %q", out.String())
	}
}

func TestSourceOfHelpExitsZero(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"source-of", "--help"}, mgrSet(), "test", &out, &errOut)
	if code != ExitOK {
		t.Errorf("exit %d, want %d (stderr=%q)", code, ExitOK, errOut.String())
	}
}
