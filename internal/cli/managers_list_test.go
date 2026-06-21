package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestManagersOverviewJSON(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fakes := []manager.Manager{
		&fakeManager{
			name: model.SourcePacman, available: true,
			scanFn: func() ([]model.Package, error) {
				return []model.Package{
					fakePackage("vim", "9.0", model.SourcePacman),
					fakePackage("git", "2.43", model.SourcePacman),
				}, nil
			},
		},
		&fakeManager{name: model.SourceApt, available: false},
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"managers", "--json", "--no-cache", "--quiet"}, fakes, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	var env struct {
		Data []managerStat `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	got := map[string]managerStat{}
	for _, s := range env.Data {
		got[s.Name] = s
	}
	if !got["pacman"].Available || got["pacman"].Count != 2 {
		t.Errorf("pacman stat = %+v, want available with count 2", got["pacman"])
	}
	if got["apt"].Available {
		t.Errorf("apt should be unavailable")
	}
}

func TestManagersOverviewAvailableOnly(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fakes := []manager.Manager{
		&fakeManager{name: model.SourcePacman, available: true},
		&fakeManager{name: model.SourceApt, available: false},
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"managers", "--json", "--available", "--no-cache", "--quiet"}, fakes, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var env struct {
		Data []managerStat `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(env.Data) != 1 || env.Data[0].Name != "pacman" {
		t.Errorf("--available should list only pacman, got %+v", env.Data)
	}
}
