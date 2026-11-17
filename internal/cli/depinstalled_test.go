package cli

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

// depMgrs is a pacman-shaped fake: the scan lists only the explicit "kitty";
// "libdep" (pulled in as a dependency, required by kitty) and "kitty" itself
// are known to the installed-database lookup.
func depMgrs() []manager.Manager {
	db := map[string]model.Package{
		"kitty":  {Name: "kitty", Version: "0.49.2-1", Source: model.SourcePacman},
		"libdep": {Name: "libdep", Version: "1.0-1", Source: model.SourcePacman, RequiredBy: []string{"kitty", "foot"}},
	}
	return []manager.Manager{&fakeManager{
		name: model.SourcePacman, available: true,
		scanFn: func() ([]model.Package, error) {
			return []model.Package{fakePackage("kitty", "0.49.2-1", model.SourcePacman)}, nil
		},
		lookupInstalledFn: func(names []string) map[string]model.Package {
			out := map[string]model.Package{}
			for _, n := range names {
				if p, ok := db[n]; ok {
					out[n] = p
				}
			}
			return out
		},
		removeCmdYesFn: func(name string) *exec.Cmd { return exec.Command("sh", "-c", "echo removing "+name) },
		removeCmdFn:    func(name string) *exec.Cmd { return exec.Command("sh", "-c", "echo removing "+name) },
	}, &fakeManager{name: model.SourceFlatpak, available: true}}
}

func TestRemoveFindsDependencyWithNamedManager(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GPK_PRIVILEGE", "")
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "libdep", "-m", "pacman", "--yes", "--json", "--no-cache"}, depMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	var env struct {
		Data struct {
			Steps []struct {
				Name       string   `json:"name"`
				Version    string   `json:"version"`
				RequiredBy []string `json:"required_by"`
				Command    []string `json:"command"`
			} `json:"steps"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil || len(env.Data.Steps) != 1 {
		t.Fatalf("plan %q: %v", out.String(), err)
	}
	s := env.Data.Steps[0]
	if s.Name != "libdep" || s.Version != "1.0-1" || strings.Join(s.RequiredBy, ",") != "kitty,foot" {
		t.Errorf("step = %+v", s)
	}
	for _, a := range s.Command {
		if a == "-Rdd" || a == "-dd" || a == "--nodeps" {
			t.Errorf("removal must never be forced: %v", s.Command)
		}
	}
}

func TestRemoveEventsCarryRequiredBy(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GPK_PRIVILEGE", "")
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "libdep", "kitty", "-m", "pacman", "--yes", "--events", "--dry-run", "--no-cache"}, depMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	evs := decodeEvents(t, out.String())
	steps := evs[0]["steps"].([]any)
	dep := steps[0].(map[string]any)
	if rb, ok := dep["required_by"].([]any); !ok || len(rb) != 2 || rb[0] != "kitty" {
		t.Errorf("libdep step = %v", dep)
	}
	if _, ok := steps[1].(map[string]any)["required_by"]; ok {
		t.Errorf("kitty is needed by nothing; required_by should be omitted: %v", steps[1])
	}
}

func TestRemoveWithoutManagerKeepsScanScope(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "libdep", "--yes", "--json", "--no-cache"}, depMgrs(), "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Fatalf("exit %d, want %d", code, ExitNegative)
	}
}

func TestInstalledFindsDependencyWithNamedManager(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	if code := Dispatch([]string{"installed", "libdep", "-m", "pacman", "--json", "--no-cache"}, depMgrs(), "test", &out, &errOut, nil); code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"installed":true`) || !strings.Contains(out.String(), `"required_by":["kitty","foot"]`) {
		t.Errorf("installed JSON = %s", out.String())
	}
	out.Reset()
	if code := Dispatch([]string{"installed", "libdep", "--no-cache", "-q"}, depMgrs(), "test", &out, &errOut, nil); code != ExitNegative {
		t.Errorf("without a manager: exit %d, want %d", code, ExitNegative)
	}
	if code := Dispatch([]string{"installed", "nothere", "-m", "pacman", "--no-cache", "-q"}, depMgrs(), "test", &out, &errOut, nil); code != ExitNegative {
		t.Errorf("missing name: exit %d, want %d", code, ExitNegative)
	}
}

func TestListStaysExplicitOnly(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	if code := Dispatch([]string{"list", "-m", "pacman", "--json", "--no-cache", "-q"}, depMgrs(), "test", &out, &errOut, nil); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(out.String(), "libdep") {
		t.Errorf("list must not pick up dependency packages: %s", out.String())
	}
}
