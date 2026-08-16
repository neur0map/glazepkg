package cli

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestHumanDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{1500 * time.Millisecond, "1.5s"},
		{25 * time.Second, "25s"},
		{95 * time.Second, "1m35s"},
		{time.Hour + 7*time.Minute, "1h07m"},
	}
	for _, c := range cases {
		if got := humanDuration(c.in); got != c.want {
			t.Errorf("humanDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A step whose child outlives the first heartbeat delay must announce itself,
// naming the command so a wedged manager can be identified and re-run. This is
// the whole point: gpk hands the terminal to the child and can't see its
// output, so silence has to be broken from gpk's side.
func TestStepAnnouncesLongRunningChild(t *testing.T) {
	orig := heartbeatDelays
	heartbeatDelays = [...]time.Duration{10 * time.Millisecond, 10 * time.Millisecond,
		10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond}
	defer func() { heartbeatDelays = orig }()

	var out bytes.Buffer
	s := startStep(&out, newStyler(), "mise", "mise upgrade")
	time.Sleep(60 * time.Millisecond)
	s.finish()

	got := out.String()
	if !strings.Contains(got, "still running") {
		t.Fatalf("no progress notice for a long child: %q", got)
	}
	if !strings.Contains(got, "mise upgrade") {
		t.Errorf("notice omits the command: %q", got)
	}
	if !strings.Contains(got, "Ctrl-C aborts") {
		t.Errorf("first notice should say how to abort: %q", got)
	}
	if n := strings.Count(got, "still running"); n < 2 {
		t.Errorf("notice repeated %d times, want it to keep reporting", n)
	}
}

// A step that finishes quickly must stay silent: no notice, ever, for the
// commands that make up almost every run.
func TestStepSilentForShortChild(t *testing.T) {
	var out bytes.Buffer
	s := startStep(&out, newStyler(), "pacman", "pacman -Syu")
	s.finish()
	if out.Len() != 0 {
		t.Errorf("short step wrote %q, want silence", out.String())
	}
}

func TestStepNilIsInert(t *testing.T) {
	var s *step
	s.finish() // must not panic
}

func TestExitReason(t *testing.T) {
	err := exec.Command("/bin/sh", "-c", "exit 7").Run()
	if got := exitReason(err); got != "exit 7" {
		t.Errorf("exitReason = %q, want %q", got, "exit 7")
	}
	if got := exitReason(errors.New("no such file")); got != "no such file" {
		t.Errorf("exitReason = %q, want the raw error", got)
	}
}

// runStep reports the child's error and a duration, and leaves the writer
// untouched for a fast command.
func TestRunStepReturnsChildOutcome(t *testing.T) {
	var out bytes.Buffer
	took, err := runStep(exec.Command("/bin/sh", "-c", "exit 3"), "pacman", newStyler(), &out, false)
	if err == nil {
		t.Fatal("expected the child's failure to surface")
	}
	if exitReason(err) != "exit 3" {
		t.Errorf("exitReason = %q, want exit 3", exitReason(err))
	}
	if took <= 0 {
		t.Errorf("duration = %v, want a positive measurement", took)
	}
}

func TestRunStepNilCommand(t *testing.T) {
	var out bytes.Buffer
	if _, err := runStep(nil, "none", newStyler(), &out, false); err == nil {
		t.Error("a nil command must report an error, not panic")
	}
}

// A non-terminal writer must not receive escape sequences: piped and captured
// output has to stay greppable.
func TestSpinnerFallsBackToPlainLines(t *testing.T) {
	var out bytes.Buffer
	sp := startSpinner(&out, newStyler(), "")
	sp.update("scanning pacman")
	sp.note("warning: aur scan failed: boom")
	sp.stop()

	got := out.String()
	if strings.Contains(got, "\033[") {
		t.Errorf("plain writer got escape sequences: %q", got)
	}
	if !strings.Contains(got, "scanning pacman...") {
		t.Errorf("missing scan line: %q", got)
	}
	if !strings.Contains(got, "aur scan failed") {
		t.Errorf("missing warning: %q", got)
	}
}

func TestSpinnerNilIsInert(t *testing.T) {
	var sp *spinner
	sp.update("x")
	sp.note("y")
	sp.stop() // must not panic
}

func TestSpinnerStopIsRepeatable(t *testing.T) {
	var out bytes.Buffer
	sp := startSpinner(&out, newStyler(), "")
	sp.stop()
	sp.stop() // a caller may both stop early and defer stop
}

// The end-of-run summary is the fix for a failure buried thousands of lines up:
// it must name the manager, its status, the command, how much of the run did
// succeed, and how to retry or skip.
func TestFailureSummaryIsActionable(t *testing.T) {
	var out bytes.Buffer
	writeFailureSummary(&out, newStyler(), "Upgrade everything", "upgrade", 3, []stepFailure{
		{name: "pnpm", cmd: "pnpm update -g", reason: "exit 1"},
	})
	got := out.String()
	for _, want := range []string{
		"1 of 3 failed",
		"pnpm",
		"exit 1",
		"pnpm update -g",
		"2 other managers finished",
		"gpk upgrade --manager pnpm",
		"'!pnpm'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("summary missing %q:\n%s", want, got)
		}
	}
}

// One broken manager must not hide the fact that the others worked, and the
// verdict must still be a failure.
func TestUpgradeAllPartialFailureReportsBoth(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	good := &fakeManager{
		name: model.SourceNpm, available: true,
		upgradeAllCmdFn: func(bool) *exec.Cmd { return exec.Command("/bin/true") },
	}
	bad := &fakeManager{
		name: model.SourcePnpm, available: true,
		upgradeAllCmdFn: func(bool) *exec.Cmd { return exec.Command("/bin/sh", "-c", "exit 1") },
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"upgrade", "--yes"}, []manager.Manager{good, bad}, "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Fatalf("exit %d, want %d", code, ExitErr)
	}
	got := errOut.String()
	if !strings.Contains(got, "1 of 2 failed") {
		t.Errorf("no partial-failure verdict: %q", got)
	}
	if !strings.Contains(got, "1 other manager finished") {
		t.Errorf("verdict doesn't say the rest applied: %q", got)
	}
}

// `gpk -Syu` with nothing able to answer the prompt used to print "cancelled"
// and exit 0 — a scripted upgrade that silently did nothing and claimed
// success. It must be an error instead.
func TestUpgradeAllUnanswerablePromptFails(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	bulk := &fakeManager{
		name: model.SourcePacman, available: true,
		upgradeAllCmdFn: func(bool) *exec.Cmd { return exec.Command("/bin/true") },
	}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"upgrade"}, []manager.Manager{bulk}, "test", &out, &errOut, strings.NewReader(""))
	if code != ExitErr {
		t.Fatalf("exit %d, want %d (stderr=%q)", code, ExitErr, errOut.String())
	}
	if !strings.Contains(errOut.String(), "--yes") {
		t.Errorf("error should point at --yes: %q", errOut.String())
	}
}

func TestConfirmDistinguishesNoFromNoAnswer(t *testing.T) {
	var out bytes.Buffer
	if got := confirm("? ", newPromptReader(strings.NewReader("y\n")), &out); got != answerYes {
		t.Errorf("y = %v, want answerYes", got)
	}
	if got := confirm("? ", newPromptReader(strings.NewReader("n\n")), &out); got != answerNo {
		t.Errorf("n = %v, want answerNo", got)
	}
	if got := confirm("? ", newPromptReader(strings.NewReader("")), &out); got != answerUnavailable {
		t.Errorf("EOF = %v, want answerUnavailable", got)
	}
	if got := confirm("? ", nil, &out); got != answerUnavailable {
		t.Errorf("nil reader = %v, want answerUnavailable", got)
	}
}

func TestStepPrefix(t *testing.T) {
	if got := stepPrefix(0, 1); got != "" {
		t.Errorf("single step prefix = %q, want empty", got)
	}
	if got := stepPrefix(1, 3); got != "[2/3] " {
		t.Errorf("prefix = %q, want %q", got, "[2/3] ")
	}
}
