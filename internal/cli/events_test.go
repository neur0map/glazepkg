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

// eventFakeMgrs is a pacman-shaped fake whose install, remove and upgrade run
// a shell script standing in for the real manager, so the event stream sees
// real child output, exit codes and signals.
func eventFakeMgrs(script func(verb, name string) string) []manager.Manager {
	run := func(verb string) func(string) *exec.Cmd {
		return func(name string) *exec.Cmd { return exec.Command("sh", "-c", script(verb, name)) }
	}
	return []manager.Manager{&fakeManager{
		name: model.SourcePacman, available: true,
		scanFn: func() ([]model.Package, error) {
			return []model.Package{fakePackage("foo", "1.0-1", model.SourcePacman), fakePackage("bar", "2.0-1", model.SourcePacman)}, nil
		},
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: q, Version: "9.9-1", Source: model.SourcePacman}}, nil
		},
		installCmdFn:    run("install"),
		installCmdYesFn: run("install"),
		removeCmdFn:     run("remove"),
		removeCmdYesFn:  run("remove"),
		upgradeCmdFn:    run("upgrade"),
		upgradeCmdYesFn: run("upgrade"),
	}}
}

type rawEvent map[string]any

func decodeEvents(t *testing.T, out string) []rawEvent {
	t.Helper()
	var evs []rawEvent
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		var e rawEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("stdout line is not JSON: %q (%v)", line, err)
		}
		evs = append(evs, e)
	}
	return evs
}

// eventKinds renders the stream compactly ("plan step:start log progress
// step:done done") so ordering assertions read like the contract.
func eventKinds(evs []rawEvent) string {
	var parts []string
	for _, e := range evs {
		k := e["event"].(string)
		if k == "step" {
			k += ":" + e["phase"].(string)
		}
		parts = append(parts, k)
	}
	return strings.Join(parts, " ")
}

func TestInstallEventsOrdering(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GPK_PRIVILEGE", "")
	mgrs := eventFakeMgrs(func(verb, name string) string {
		return `echo "resolving dependencies..."; echo "warning: a note" >&2; echo "(1/2) installing ` + name + `"; printf '(2/2) installing dep\r'; echo; exit 0`
	})
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "foo", "baz", "-m", "pacman", "--yes", "--events"}, mgrs, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, want 0 (stderr=%q)", code, errOut.String())
	}
	evs := decodeEvents(t, out.String())
	want := "plan " +
		"step:start log log log progress log progress step:done " +
		"step:start log log log progress log progress step:done done"
	if got := eventKinds(evs); got != want {
		t.Fatalf("event order\n got: %s\nwant: %s", got, want)
	}
	plan := evs[0]
	if plan["action"] != "install" {
		t.Errorf("plan action = %v", plan["action"])
	}
	steps := plan["steps"].([]any)
	first := steps[0].(map[string]any)
	if first["manager"] != "pacman" || first["name"] != "foo" || first["version"] != "9.9-1" {
		t.Errorf("plan step 0 = %v", first)
	}
	if evs[3]["line"] != "warning: a note" {
		t.Errorf("stderr line not logged in order: %v", evs[3])
	}
	if p := evs[5]; p["done"] != float64(1) || p["total"] != float64(2) || p["index"] != float64(0) {
		t.Errorf("progress event = %v", p)
	}
	if d := evs[8]; d["exit"] != float64(0) || d["index"] != float64(0) {
		t.Errorf("step done = %v", d)
	}
	if evs[9]["index"] != float64(1) {
		t.Errorf("second step index = %v", evs[9]["index"])
	}
	last := evs[len(evs)-1]
	if last["ok"] != true || last["exit"] != float64(0) || last["error"] != "" {
		t.Errorf("final done = %v", last)
	}
}

func TestInstallEventsStopAtFailure(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GPK_PRIVILEGE", "")
	mgrs := eventFakeMgrs(func(verb, name string) string {
		if name == "foo" {
			return `echo "error: target not found: foo" >&2; exit 1`
		}
		return `echo never`
	})
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "foo", "baz", "-m", "pacman", "-y", "--events"}, mgrs, "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Fatalf("exit %d, want %d", code, ExitErr)
	}
	evs := decodeEvents(t, out.String())
	if got, want := eventKinds(evs), "plan step:start log step:done done"; got != want {
		t.Fatalf("event order\n got: %s\nwant: %s", got, want)
	}
	if evs[3]["exit"] != float64(1) {
		t.Errorf("step exit = %v", evs[3]["exit"])
	}
	last := evs[4]
	if last["ok"] != false || last["exit"] != float64(ExitErr) || !strings.Contains(last["error"].(string), "foo (pacman): exit 1") {
		t.Errorf("final done = %v", last)
	}
}

func TestRemoveEvents(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GPK_PRIVILEGE", "")
	mgrs := eventFakeMgrs(func(verb, name string) string {
		return `echo "(1/1) removing ` + name + `"`
	})
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "bar", "-m", "pacman", "--yes", "--events", "--no-cache"}, mgrs, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	evs := decodeEvents(t, out.String())
	if got, want := eventKinds(evs), "plan step:start log progress step:done done"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	step := evs[0]["steps"].([]any)[0].(map[string]any)
	if evs[0]["action"] != "remove" || step["name"] != "bar" || step["version"] != "2.0-1" {
		t.Errorf("plan = %v", evs[0])
	}
}

func TestRemoveEventsNotInstalledStillCloses(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GPK_PRIVILEGE", "")
	mgrs := eventFakeMgrs(func(verb, name string) string { return "true" })
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"remove", "nope", "-m", "pacman", "--yes", "--events", "--no-cache"}, mgrs, "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Fatalf("exit %d, want %d", code, ExitNegative)
	}
	evs := decodeEvents(t, out.String())
	if len(evs) != 1 || evs[0]["event"] != "done" || evs[0]["exit"] != float64(ExitNegative) ||
		!strings.Contains(evs[0]["error"].(string), `"nope" is not installed`) {
		t.Fatalf("events = %v", evs)
	}
}

func TestUpgradeEvents(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GPK_PRIVILEGE", "")
	mgrs := eventFakeMgrs(func(verb, name string) string {
		return `echo "(1/1) upgrading ` + name + `"`
	})
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"upgrade", "foo", "-m", "pacman", "--yes", "--events", "--no-cache"}, mgrs, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	evs := decodeEvents(t, out.String())
	if got, want := eventKinds(evs), "plan step:start log progress step:done done"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if evs[0]["action"] != "upgrade" {
		t.Errorf("plan = %v", evs[0])
	}
}

func TestEventsDryRunEmitsPlanOnly(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GPK_PRIVILEGE", "")
	mgrs := eventFakeMgrs(func(verb, name string) string { return "exit 7" })
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "foo", "-m", "pacman", "--yes", "--events", "--dry-run"}, mgrs, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if got := eventKinds(decodeEvents(t, out.String())); got != "plan done" {
		t.Fatalf("got %s", got)
	}
}

func TestEventsRejectInteractiveFlags(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	mgrs := eventFakeMgrs(func(verb, name string) string { return "true" })
	for _, args := range [][]string{
		{"install", "foo", "-m", "pacman", "--events"},
		{"install", "foo", "-m", "pacman", "--events", "--yes", "--json"},
		{"install", "foo", "-m", "pacman", "--events", "--yes", "--pick-version"},
	} {
		var out, errOut bytes.Buffer
		if code := Dispatch(args, mgrs, "test", &out, &errOut, nil); code != ExitErr {
			t.Errorf("%v: exit %d, want %d", args, code, ExitErr)
		}
		evs := decodeEvents(t, out.String())
		if len(evs) != 1 || evs[0]["event"] != "done" || evs[0]["ok"] != false {
			t.Errorf("%v: events = %v", args, evs)
		}
	}
}

func TestParseProgress(t *testing.T) {
	cases := []struct {
		line        string
		done, total int
		ok          bool
	}{
		{"(1/3) installing ripgrep", 1, 3, true},
		{"( 2/12) upgrading linux", 2, 12, true},
		{"(1/1) removing foo", 1, 1, true},
		{"(1/3) checking keys in keyring", 0, 0, false},
		{"(2/3) checking package integrity", 0, 0, false},
		{"[3/4] Installing ripgrep-0:14.1.1-1.fc42.x86_64 100% |   1.2 MiB/s", 3, 4, true},
		{"[1/4] Verify package files", 0, 0, false},
		{"  Installing       : ripgrep-13.0.0-1.fc39.x86_64        1/2 ", 1, 2, true},
		{"  Verifying        : ripgrep-13.0.0-1.fc39.x86_64        1/2", 0, 0, false},
		{"Installing 1/2…", 1, 2, true},
		{"Uninstalling 1/1…", 1, 1, true},
		{"(5/3) installing broken", 0, 0, false},
		{"plain output", 0, 0, false},
	}
	for _, c := range cases {
		d, tot, ok := parseProgress(c.line)
		if ok != c.ok || d != c.done || tot != c.total {
			t.Errorf("parseProgress(%q) = %d,%d,%v want %d,%d,%v", c.line, d, tot, ok, c.done, c.total, c.ok)
		}
	}
}

func TestReadLinesSplitsCarriageReturnsAndStripsANSI(t *testing.T) {
	var got []string
	readLines(strings.NewReader("a\r\nb\rc\n\x1b[1;32mgreen\x1b[0m  \n\n   \ntail"), func(s string) { got = append(got, s) })
	want := []string{"a", "b", "c", "green", "tail"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("lines = %q, want %q", got, want)
	}
}

func TestReadLinesSplitsOverlongLines(t *testing.T) {
	var got []string
	readLines(strings.NewReader(strings.Repeat("x", maxLogLine+10)+"\n"), func(s string) { got = append(got, s) })
	if len(got) != 2 || len(got[0]) != maxLogLine || len(got[1]) != 10 {
		t.Errorf("got %d lines (first %d bytes)", len(got), len(got[0]))
	}
}

func TestErrorTapKeepsLastError(t *testing.T) {
	var sink bytes.Buffer
	tap := &errorTap{w: &sink}
	_, _ = tap.Write([]byte("note\nerror: first\nerr"))
	_, _ = tap.Write([]byte("or: second one\n"))
	if tap.last() != "second one" {
		t.Errorf("last = %q", tap.last())
	}
	if sink.String() != "note\nerror: first\nerror: second one\n" {
		t.Errorf("passthrough = %q", sink.String())
	}
}
