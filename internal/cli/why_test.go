package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestWhyReverseDeps(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	pac := &fakeManager{
		name:      model.SourcePacman,
		available: true,
		scanFn: func() ([]model.Package, error) {
			return []model.Package{
				fakePackage("foo", "1.0", model.SourcePacman),
				fakePackage("bar", "2.0", model.SourcePacman),
			}, nil
		},
		depsFn: func(pkgs []model.Package) map[string][]string {
			return map[string][]string{"foo": {"bar>=2.0"}}
		},
	}
	mgrs := []manager.Manager{pac}

	var out, errOut bytes.Buffer
	if code := Dispatch([]string{"why", "bar", "--quiet"}, mgrs, "test", &out, &errOut, nil); code != ExitOK {
		t.Fatalf("why bar: exit %d, stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "foo") {
		t.Errorf("foo should require bar, got %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := Dispatch([]string{"why", "foo", "--quiet"}, mgrs, "test", &out, &errOut, nil); code != ExitOK {
		t.Fatalf("why foo: exit %d", code)
	}
	if !strings.Contains(out.String(), "safe to remove") {
		t.Errorf("foo should be safe to remove, got %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := Dispatch([]string{"why", "missing", "--quiet"}, mgrs, "test", &out, &errOut, nil); code != ExitNegative {
		t.Errorf("why missing: exit %d, want %d", code, ExitNegative)
	}
}
