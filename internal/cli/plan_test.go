package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestInstallJSONPlan(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "git", "--manager", "pacman", "--json"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	var env struct {
		Data planEnvelope `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if env.Data.Action != "install" || len(env.Data.Steps) != 1 {
		t.Fatalf("plan = %+v", env.Data)
	}
	step := env.Data.Steps[0]
	if step.Manager != "pacman" || step.Name != "git" || len(step.Command) == 0 {
		t.Errorf("step = %+v", step)
	}
}

func TestInstallJSONAmbiguousExit3(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	// ripgrep resolves to both pacman and brew; --json is non-interactive.
	code := Dispatch([]string{"install", "ripgrep", "--json"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitAmbiguous {
		t.Errorf("exit %d, want %d", code, ExitAmbiguous)
	}
}
