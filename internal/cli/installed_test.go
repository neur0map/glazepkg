package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestInstalledAllPresent(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"installed", "--no-cache", "--quiet", "vim", "git"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
}

func TestInstalledOneMissing(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"installed", "--no-cache", "--quiet", "vim", "nonexistent"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Fatalf("exit %d, want %d", code, ExitNegative)
	}
}

func TestInstalledZeroArgsErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"installed"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
}

func TestInstalledJSONShape(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"installed", "--json", "--no-cache", "--quiet", "vim", "nonexistent"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Fatalf("exit %d", code)
	}
	var env struct {
		Schema int `json:"schema"`
		Data   []struct {
			Name      string       `json:"name"`
			Installed bool         `json:"installed"`
			Matches   []cliPackage `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v\nbody=%s", err, out.String())
	}
	if len(env.Data) != 2 {
		t.Fatalf("data length = %d, want 2", len(env.Data))
	}
	if !env.Data[0].Installed || env.Data[0].Name != "vim" {
		t.Errorf("entry 0 = %+v", env.Data[0])
	}
	if env.Data[1].Installed || env.Data[1].Name != "nonexistent" {
		t.Errorf("entry 1 = %+v", env.Data[1])
	}
}

func TestInstalledManagerFilter(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// vim is in pacman; with --manager brew it should be reported as missing.
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"installed", "--manager", "brew", "--no-cache", "--quiet", "vim"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Errorf("exit %d, want %d", code, ExitNegative)
	}
}

func TestInstalledReportsMissingOnStderr(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// Without --quiet, missing packages get listed on stderr.
	var out, errOut bytes.Buffer
	_ = Dispatch([]string{"installed", "--no-cache", "nonexistent"}, mgrSet(), "test", &out, &errOut, nil)
	if !strings.Contains(errOut.String(), "nonexistent") {
		t.Errorf("stderr = %q, want substring 'nonexistent'", errOut.String())
	}
}

func TestInstalledFlagAfterPositional(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// User-style: subcommand, package, then flag. Must work the same as
	// flags-first.
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"installed", "vim", "--json", "--no-cache", "--quiet"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("schema")) {
		t.Errorf("expected JSON envelope on stdout, got %q", out.String())
	}
}

func TestInstalledHelpExitsZero(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"installed", "--help"}, mgrSet(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Errorf("exit %d, want %d (stderr=%q)", code, ExitOK, errOut.String())
	}
}
