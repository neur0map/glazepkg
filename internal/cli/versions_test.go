package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestVersionsSortsNewestFirst(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	pip := &fakeManager{
		name:      model.SourcePip,
		available: true,
		versionsFn: func(name string) ([]string, error) {
			if name != "black" {
				return nil, nil
			}
			return []string{"1.2.0", "1.10.0", "1.9.0"}, nil
		},
	}
	mgrs := []manager.Manager{pip}

	var out, errOut bytes.Buffer
	if code := Dispatch([]string{"versions", "black", "--manager", "pip", "--json"}, mgrs, "test", &out, &errOut, nil); code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	var env struct {
		Data struct {
			Sources []struct {
				Versions []string `json:"versions"`
			} `json:"sources"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	got := env.Data.Sources[0].Versions
	want := []string{"1.10.0", "1.9.0", "1.2.0"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("newest-first sort: got %v, want %v", got, want)
	}

	out.Reset()
	errOut.Reset()
	if code := Dispatch([]string{"versions", "missing", "--manager", "pip"}, mgrs, "test", &out, &errOut, nil); code != ExitNegative {
		t.Errorf("missing versions: exit %d, want %d", code, ExitNegative)
	}
}
