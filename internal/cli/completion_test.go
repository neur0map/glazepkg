package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func runCompletionsKind(t *testing.T, kind string, mgrs []manager.Manager) string {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Dispatch(append([]string{"completions"}, kind), mgrs, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("completions %s exit %d", kind, code)
	}
	return out.String()
}

func TestCompletionsCommands(t *testing.T) {
	out := runCompletionsKind(t, "commands", nil)
	for _, want := range []string{"install", "remove", "search", "hold", "undo"} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("commands completion missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "completions\n") {
		t.Error("the internal 'completions' helper should not be offered")
	}
}

func TestCompletionsManagers(t *testing.T) {
	mgrs := []manager.Manager{&fakeManager{name: model.SourcePacman, available: true}}
	out := runCompletionsKind(t, "managers", mgrs)
	if !strings.Contains(out, "pacman\n") {
		t.Errorf("managers completion missing pacman:\n%s", out)
	}
	if !strings.Contains(out, "cask\n") {
		t.Errorf("managers completion missing alias 'cask':\n%s", out)
	}
}

func TestCompletionsInstalled(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	manager.SaveScanCache([]model.Package{
		{Name: "ripgrep", Source: model.SourcePacman},
		{Name: "fd", Source: model.SourcePacman},
	})
	out := runCompletionsKind(t, "installed", nil)
	if !strings.Contains(out, "ripgrep\n") || !strings.Contains(out, "fd\n") {
		t.Errorf("installed completion missing names:\n%s", out)
	}
}

func TestCompletionScripts(t *testing.T) {
	cases := map[string]string{
		"bash": "complete -F _gpk gpk",
		"zsh":  "compdef _gpk gpk",
		"fish": "__fish_use_subcommand",
	}
	for shell, want := range cases {
		var out, errOut bytes.Buffer
		if code := Dispatch([]string{"completion", shell}, nil, "test", &out, &errOut, nil); code != ExitOK {
			t.Fatalf("completion %s exit %d", shell, code)
		}
		if !strings.Contains(out.String(), want) {
			t.Errorf("%s script missing %q", shell, want)
		}
	}
	var out, errOut bytes.Buffer
	if code := Dispatch([]string{"completion", "fishy"}, nil, "test", &out, &errOut, nil); code != ExitErr {
		t.Errorf("invalid shell exit %d, want %d", code, ExitErr)
	}
}
