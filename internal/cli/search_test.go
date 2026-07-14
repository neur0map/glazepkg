package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/config"
	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestRankSearchRows(t *testing.T) {
	rows := []searchRow{
		{pkg: model.Package{Name: "ripgrep-all"}},
		{pkg: model.Package{Name: "repgrep"}},
		{pkg: model.Package{Name: "ripgrep"}},
	}
	rankSearchRows(rows, "ripgrep")
	if rows[0].pkg.Name != "ripgrep" {
		t.Errorf("exact match should rank first, got %q", rows[0].pkg.Name)
	}
	if rows[1].pkg.Name != "ripgrep-all" {
		t.Errorf("prefix match should rank second, got %q", rows[1].pkg.Name)
	}
}

func TestSearchManagersDedupesToCanonical(t *testing.T) {
	pacman := &fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{
				{Name: "x", Source: model.SourcePacman},
				{Name: "yay", Source: model.SourceAUR},
			}, nil
		},
	}
	aur := &fakeManager{
		name: model.SourceAUR, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "yay", Source: model.SourceAUR}}, nil
		},
	}
	rows, _ := searchManagers([]manager.Manager{pacman, aur}, "x")
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	for _, r := range rows {
		if r.pkg.Name == "yay" && r.mgr.Name() != model.SourceAUR {
			t.Errorf("yay should map to the aur manager, got %s", r.mgr.Name())
		}
	}
}

func TestMarkInstalledRows(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	manager.SaveScanCache([]model.Package{{Name: "x", Source: model.SourcePacman}})
	rows := []searchRow{
		{pkg: model.Package{Name: "x", Source: model.SourcePacman}},
		{pkg: model.Package{Name: "y", Source: model.SourcePacman}},
	}
	markInstalledRows(rows)
	if !rows[0].installed {
		t.Error("x should be marked installed")
	}
	if rows[1].installed {
		t.Error("y should not be marked installed")
	}
}

func TestWriteSearchHumanBottomUp(t *testing.T) {
	rows := []searchRow{
		{pkg: model.Package{Name: "foo", Version: "1.0", Source: model.SourcePacman}},
		{pkg: model.Package{Name: "foobar", Version: "2.0", Source: model.SourcePacman, Description: "more foo"}},
	}
	var buf bytes.Buffer
	writeSearchHuman(&buf, "foo", rows, &styler{}, true)
	out := buf.String()

	best := strings.Index(out, "1  foo")
	worst := strings.Index(out, "2  foobar")
	header := strings.Index(out, `results for "foo"  (2)`)
	if best < 0 || worst < 0 || header < 0 {
		t.Fatalf("missing pieces in output:\n%s", out)
	}
	if worst > best {
		t.Errorf("bottom-up should print #2 before #1:\n%s", out)
	}
	if header < best {
		t.Errorf("bottom-up should print the header below the rows:\n%s", out)
	}
	if desc := strings.Index(out, "more foo"); desc < worst || desc > best {
		t.Errorf("description should stay attached under its own row:\n%s", out)
	}
}

func searchFakeMgrs() []manager.Manager {
	return []manager.Manager{&fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{
				{Name: "foo", Version: "1.0", Source: model.SourcePacman},
				{Name: "foobar", Version: "2.0", Source: model.SourcePacman},
			}, nil
		},
	}}
}

func TestSearchBottomUpFlagAndConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	run := func(args ...string) string {
		t.Helper()
		var out, errOut bytes.Buffer
		code := Dispatch(append([]string{"search"}, args...), searchFakeMgrs(), "test", &out, &errOut, nil)
		if code != ExitOK {
			t.Fatalf("exit %d, stderr=%q", code, errOut.String())
		}
		return out.String()
	}
	bottomUpOrder := func(out string) bool {
		t.Helper()
		best, worst := strings.Index(out, "1  foo"), strings.Index(out, "2  foobar")
		if best < 0 || worst < 0 {
			t.Fatalf("rows missing from output:\n%s", out)
		}
		return worst < best
	}

	if bottomUpOrder(run("foo")) {
		t.Error("default order should be top-down")
	}
	if !bottomUpOrder(run("foo", "--bottomup")) {
		t.Error("--bottomup should reverse the order")
	}
	if !bottomUpOrder(run("foo", "--reverse")) {
		t.Error("--reverse should alias --bottomup")
	}

	if err := config.Save(config.Config{Search: config.SearchConfig{BottomUp: true}}); err != nil {
		t.Fatal(err)
	}
	if !bottomUpOrder(run("foo")) {
		t.Error("bottom_up config should reverse the order")
	}
	if bottomUpOrder(run("foo", "--topdown")) {
		t.Error("--topdown should override the config")
	}
}
