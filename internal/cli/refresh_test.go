package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestRefreshJSON(t *testing.T) {
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
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"refresh", "--json", "--quiet"}, fakes, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	var env struct {
		Data struct {
			Packages int `json:"packages"`
			Managers int `json:"managers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if env.Data.Packages != 2 || env.Data.Managers != 1 {
		t.Errorf("refresh data = %+v, want 2 packages / 1 manager", env.Data)
	}
}
