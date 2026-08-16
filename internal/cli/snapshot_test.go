package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
	"github.com/neur0map/glazepkg/internal/snapshot"
)

func snapshotFakeMgrs(pkgs ...model.Package) []manager.Manager {
	return []manager.Manager{&fakeManager{
		name: model.SourcePacman, available: true,
		scanFn: func() ([]model.Package, error) { return pkgs, nil },
	}}
}

// saveSnapshotAt writes a snapshot with a controlled timestamp so index order
// is deterministic; filenames are timestamp-derived.
func saveSnapshotAt(t *testing.T, at time.Time, pkgs ...model.Package) {
	t.Helper()
	snap := snapshot.New(pkgs)
	snap.Timestamp = at
	if _, err := snapshot.Save(snap); err != nil {
		t.Fatalf("save: %v", err)
	}
}

func TestSnapshotSaveWritesFile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"snapshot", "-q"},
		snapshotFakeMgrs(fakePackage("git", "2.43", model.SourcePacman)), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	paths, err := snapshot.List()
	if err != nil || len(paths) != 1 {
		t.Fatalf("List() = %v, %v; want one snapshot", paths, err)
	}
	if !strings.Contains(out.String(), "1 package") {
		t.Errorf("stdout = %q, want a package count", out.String())
	}
}

func TestSnapshotListNumbersNewestFirst(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := snapshot.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	older := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	saveSnapshotAt(t, older, fakePackage("git", "2.43", model.SourcePacman))
	saveSnapshotAt(t, older.Add(time.Hour), fakePackage("git", "2.44", model.SourcePacman), fakePackage("vim", "9.1", model.SourcePacman))

	rows, err := listSnapshots()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if !rows[0].Taken.After(rows[1].Taken) {
		t.Errorf("index 1 (%v) should be newer than index 2 (%v)", rows[0].Taken, rows[1].Taken)
	}
	if rows[0].Index != 1 || rows[0].Packages != 2 {
		t.Errorf("row 1 = %+v", rows[0])
	}
}

// The point of the feature: pick any two saved snapshots and see what changed
// between them (#64).
func TestSnapshotDiffBetweenTwoSaved(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := snapshot.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	saveSnapshotAt(t, base,
		fakePackage("git", "2.43", model.SourcePacman),
		fakePackage("gone", "1.0", model.SourcePacman))
	saveSnapshotAt(t, base.Add(time.Hour),
		fakePackage("git", "2.44", model.SourcePacman),
		fakePackage("fresh", "0.1", model.SourcePacman))

	var out, errOut bytes.Buffer
	code := Dispatch([]string{"snapshot", "diff", "2", "1", "--json"}, snapshotFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	var env struct {
		Data struct {
			Added    []diffEntryJSON `json:"added"`
			Upgraded []diffEntryJSON `json:"upgraded"`
			Removed  []diffEntryJSON `json:"removed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("bad JSON %q: %v", out.String(), err)
	}
	if len(env.Data.Added) != 1 || env.Data.Added[0].Name != "fresh" {
		t.Errorf("added = %+v, want fresh", env.Data.Added)
	}
	if len(env.Data.Removed) != 1 || env.Data.Removed[0].Name != "gone" {
		t.Errorf("removed = %+v, want gone", env.Data.Removed)
	}
	if len(env.Data.Upgraded) != 1 || env.Data.Upgraded[0].From != "2.43" || env.Data.Upgraded[0].To != "2.44" {
		t.Errorf("upgraded = %+v, want git 2.43 to 2.44", env.Data.Upgraded)
	}
}

// Bare `snapshot diff` reproduces the TUI's `d`: newest saved against live.
func TestSnapshotDiffDefaultsToLive(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := snapshot.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	saveSnapshotAt(t, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		fakePackage("git", "2.43", model.SourcePacman))

	var out, errOut bytes.Buffer
	code := Dispatch([]string{"snapshot", "diff", "-q"},
		snapshotFakeMgrs(fakePackage("git", "2.44", model.SourcePacman)), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "→ now") {
		t.Errorf("diff should be against live state: %q", got)
	}
	if !strings.Contains(got, "2.43") || !strings.Contains(got, "2.44") {
		t.Errorf("upgrade not shown: %q", got)
	}
}

func TestSnapshotDiffRejectsBadSelector(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"snapshot", "diff", "7", "1"}, snapshotFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Fatalf("exit %d, want %d", code, ExitErr)
	}
	if !strings.Contains(errOut.String(), "no snapshots saved") {
		t.Errorf("stderr = %q", errOut.String())
	}
}

// Periodic snapshots are the stated use case, so retention has to work or the
// feature fills the disk.
func TestSnapshotPruneKeepsNewest(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := snapshot.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := range 5 {
		saveSnapshotAt(t, base.Add(time.Duration(i)*time.Hour), fakePackage("git", "2.43", model.SourcePacman))
	}
	var out, errOut bytes.Buffer
	if code := Dispatch([]string{"snapshot", "prune", "--keep", "2"}, snapshotFakeMgrs(), "test", &out, &errOut, nil); code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	rows, err := listSnapshots()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("kept %d snapshots, want 2", len(rows))
	}
	if !rows[0].Taken.Equal(base.Add(4 * time.Hour)) {
		t.Errorf("kept the wrong ones; newest is %v", rows[0].Taken)
	}
	if code := Dispatch([]string{"snapshot", "prune", "--keep", "0"}, snapshotFakeMgrs(), "test", &out, &errOut, nil); code != ExitErr {
		t.Error("--keep 0 would delete everything; must be rejected")
	}
}

func TestSnapshotSaveKeepPrunesInline(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	if err := snapshot.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := range 3 {
		saveSnapshotAt(t, base.Add(time.Duration(i)*time.Hour), fakePackage("git", "2.43", model.SourcePacman))
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"snapshot", "save", "-q", "--keep", "1"},
		snapshotFakeMgrs(fakePackage("git", "2.44", model.SourcePacman)), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	rows, err := listSnapshots()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("kept %d snapshots, want 1", len(rows))
	}
	if _, err := os.Stat(rows[0].Path); err != nil {
		t.Errorf("surviving snapshot unreadable: %v", err)
	}
}

func TestSnapshotUnknownAction(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	if code := Dispatch([]string{"snapshot", "bogus"}, snapshotFakeMgrs(), "test", &out, &errOut, nil); code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
}

func TestSortDiffIsStable(t *testing.T) {
	d := model.Diff{Added: []model.Package{
		{Name: "zsh", Source: model.SourcePacman},
		{Name: "atop", Source: model.SourceBrew},
		{Name: "abc", Source: model.SourcePacman},
	}}
	sortDiff(&d)
	got := []string{d.Added[0].Name, d.Added[1].Name, d.Added[2].Name}
	want := []string{"atop", "abc", "zsh"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted %v, want %v (by source then name)", got, want)
		}
	}
}
