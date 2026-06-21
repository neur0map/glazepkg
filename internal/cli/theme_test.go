package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/config"
)

func TestThemeSetPersists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"theme", "dracula"}, nil, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	if got := config.Load().Appearance.Theme; got != "dracula" {
		t.Errorf("saved theme = %q, want dracula", got)
	}
}

func TestThemeUnknownSuggests(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"theme", "drakula"}, nil, "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
	if !strings.Contains(errOut.String(), "dracula") {
		t.Errorf("stderr = %q, want a 'dracula' suggestion", errOut.String())
	}
}

func TestThemeListJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"theme", "--json"}, nil, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var env struct {
		Data struct {
			Active string   `json:"active"`
			Themes []string `json:"themes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(env.Data.Themes) == 0 {
		t.Error("expected at least one theme")
	}
}
