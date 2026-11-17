package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// --events turns a write command (install, remove, upgrade) into an NDJSON
// stream on stdout for GUIs: one plan event, then per step a start event, the
// child's output as log events (stdout and stderr, one per line), progress
// events when the child prints counts gpk can read, and a done event; the
// stream always closes with one final done event. Exit codes are unchanged.

type eventPlanStep struct {
	Manager string `json:"manager"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	// RequiredBy lists installed packages that depend on this one (remove
	// only), so a GUI can warn; the manager refuses to break them.
	RequiredBy []string `json:"required_by,omitempty"`
}

type planEvent struct {
	Event  string          `json:"event"`
	Action string          `json:"action"`
	Steps  []eventPlanStep `json:"steps"`
}

type stepEvent struct {
	Event string `json:"event"`
	Index int    `json:"index"`
	Phase string `json:"phase"`
	Exit  *int   `json:"exit,omitempty"`
}

type logEvent struct {
	Event string `json:"event"`
	Index int    `json:"index"`
	Line  string `json:"line"`
}

type progressEvent struct {
	Event string `json:"event"`
	Index int    `json:"index"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
}

type doneEvent struct {
	Event string `json:"event"`
	OK    bool   `json:"ok"`
	Exit  int    `json:"exit"`
	Error string `json:"error"`
}

// eventRun owns one command's event stream: the writer, the error that will
// close it, and SIGINT/SIGTERM for the whole run so a signal between steps
// still ends with a done event instead of killing gpk mid-stream.
type eventRun struct {
	mu        sync.Mutex
	w         io.Writer
	errMsg    string
	finished  bool
	sigs      chan os.Signal
	cancelled bool
	stderr    *errorTap
}

// startEvents begins an event stream on stdout and returns the stderr writer
// the command should use from now on (it remembers the last "error:" line so
// the done event can carry it).
func startEvents(stdout, stderr io.Writer) (*eventRun, io.Writer) {
	ev := &eventRun{w: stdout, sigs: make(chan os.Signal, 4)}
	signal.Notify(ev.sigs, os.Interrupt, syscall.SIGTERM)
	ev.stderr = &errorTap{w: stderr}
	return ev, ev.stderr
}

func (ev *eventRun) emit(v any) {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	b = append(b, '\n')
	_, _ = ev.w.Write(b)
}

// fail records the error the closing done event reports; the first one wins.
func (ev *eventRun) fail(msg string) {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if ev.errMsg == "" {
		ev.errMsg = msg
	}
}

// interrupted reports whether SIGINT or SIGTERM arrived, draining any that
// came in between steps.
func (ev *eventRun) interrupted() bool {
	for {
		select {
		case <-ev.sigs:
			ev.cancelled = true
		default:
			return ev.cancelled
		}
	}
}

// finish emits the closing done event once and stops catching signals.
func (ev *eventRun) finish(code int) {
	if ev == nil {
		return
	}
	ev.mu.Lock()
	if ev.finished {
		ev.mu.Unlock()
		return
	}
	ev.finished = true
	msg := ev.errMsg
	ev.mu.Unlock()
	signal.Stop(ev.sigs)
	if code != ExitOK && msg == "" {
		msg = ev.stderr.last()
	}
	if code != ExitOK && msg == "" {
		msg = fmt.Sprintf("exit %d", code)
	}
	if code == ExitOK {
		msg = ""
	}
	ev.emit(doneEvent{Event: "done", OK: code == ExitOK, Exit: code, Error: msg})
}

// eventStep is one command of an event-driven plan.
type eventStep struct {
	plan  eventPlanStep
	cmd   *exec.Cmd
	after func() // bookkeeping once the step succeeded (caches, history)
}

// runEventPlan emits the plan and runs each step, stopping at the first
// failure unless keepGoing (bulk upgrades try every manager). Returns the
// command's exit code: ExitErr on a failed step, ExitInterrupted when a
// signal cancelled the run.
func runEventPlan(ev *eventRun, action string, steps []eventStep, dryRun, keepGoing bool, stderr io.Writer) int {
	for _, s := range steps {
		if _, err := withPrivilege(s.cmd); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			ev.fail(err.Error())
			return ExitErr
		}
	}
	pe := planEvent{Event: "plan", Action: action, Steps: make([]eventPlanStep, len(steps))}
	for i, s := range steps {
		pe.Steps[i] = s.plan
	}
	ev.emit(pe)
	if dryRun {
		return ExitOK
	}

	failed := false
	for i, s := range steps {
		if ev.interrupted() {
			ev.fail("cancelled")
			return ExitInterrupted
		}
		code, err := runEventStep(ev, i, s.cmd)
		if ev.interrupted() {
			ev.fail("cancelled")
			return ExitInterrupted
		}
		if err != nil || code != 0 {
			label := s.plan.Name
			if label == "" {
				label = s.plan.Manager
			} else {
				label += " (" + s.plan.Manager + ")"
			}
			reason := fmt.Sprintf("exit %d", code)
			if err != nil {
				reason = err.Error()
			}
			fmt.Fprintf(stderr, "error: %s: %s\n", label, reason)
			ev.fail(label + ": " + reason)
			if !keepGoing {
				return ExitErr
			}
			failed = true
			continue
		}
		if s.after != nil {
			s.after()
		}
	}
	if failed {
		return ExitErr
	}
	return ExitOK
}

// stepDrainGrace bounds how long a finished step waits for its output pipe to
// close: a daemon the child left behind can hold the write end open forever.
var stepDrainGrace = 2 * time.Second

// runEventStep runs one command with its stdout and stderr merged into log
// events, forwarding SIGINT/SIGTERM to it. It returns the child's exit code
// (128+signal when a signal killed it) and an error only when the command
// couldn't start.
func runEventStep(ev *eventRun, index int, cmd *exec.Cmd) (int, error) {
	exit := func(code int) {
		ev.emit(stepEvent{Event: "step", Index: index, Phase: "done", Exit: &code})
	}
	ev.emit(stepEvent{Event: "step", Index: index, Phase: "start"})

	cmd, err := withPrivilege(cmd)
	if err != nil {
		exit(127)
		return 127, err
	}
	// The process environment already has the C locale (UseStableLocale), so
	// log lines and progress counts come out in a form gpk can read.
	pr, pw, err := os.Pipe()
	if err != nil {
		exit(127)
		return 127, err
	}
	cmd.Stdin = nil
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		ev.emit(logEvent{Event: "log", Index: index, Line: err.Error()})
		exit(127)
		return 127, err
	}
	pw.Close()

	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		readLines(pr, func(line string) {
			ev.emit(logEvent{Event: "log", Index: index, Line: line})
			if done, total, ok := parseProgress(line); ok {
				ev.emit(progressEvent{Event: "progress", Index: index, Done: done, Total: total})
			}
		})
	}()

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	for {
		select {
		case sig := <-ev.sigs:
			ev.cancelled = true
			// A root child (pkexec) may refuse the signal; it then runs to
			// completion and the run still ends cancelled.
			_ = cmd.Process.Signal(sig)
			continue
		case err = <-waitErr:
		}
		break
	}

	select {
	case <-readerDone:
	case <-time.After(stepDrainGrace):
		pr.Close()
		<-readerDone
	}
	pr.Close()

	code := 0
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			exit(127)
			return 127, err
		}
		code = ee.ExitCode()
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			code = 128 + int(ws.Signal())
		}
	}
	exit(code)
	return code, nil
}

// maxLogLine caps one log event; a longer run of bytes without a line break
// is split rather than stalling the child on a full pipe.
const maxLogLine = 16 * 1024

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b[()][0-9A-Za-z]|\x1b[=>]`)

// readLines calls fn for every non-empty line in r, splitting on \n and \r
// (progress bars redraw with \r) and dropping terminal escape sequences.
func readLines(r io.Reader, fn func(string)) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 4096), maxLogLine)
	sc.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
			return i + 1, data[:i], nil
		}
		if len(data) >= maxLogLine {
			return len(data), data, nil
		}
		if atEOF && len(data) > 0 {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	for sc.Scan() {
		line := strings.TrimRight(ansiEscape.ReplaceAllString(sc.Text(), ""), " \t")
		if strings.TrimSpace(line) != "" {
			fn(line)
		}
	}
	// Keep draining on a scanner error so the child never blocks on the pipe.
	_, _ = io.Copy(io.Discard, r)
}

var (
	// pacman (and yay/paru, which run it): "(1/3) installing ripgrep". Only the
	// transaction verbs count; the check/download phases restart their own
	// numbering and would make the count run backwards.
	pacmanProgress = regexp.MustCompile(`^\(\s*(\d+)/(\d+)\)\s+(installing|upgrading|reinstalling|downgrading|removing)\b`)
	// dnf5: "[3/4] Installing ripgrep-0:14.1.1-1.fc42.x86_64 100% | ...".
	dnf5Progress = regexp.MustCompile(`^\[\s*(\d+)/(\d+)\]\s+(Installing|Upgrading|Reinstalling|Downgrading|Removing|Erasing)\b`)
	// dnf4: "  Installing   : ripgrep-13.0.0-1.fc39.x86_64      1/1".
	dnf4Progress = regexp.MustCompile(`^\s*(Installing|Upgrading|Reinstalling|Downgrading|Erasing|Removing)\s*:.*\s(\d+)/(\d+)\s*$`)
	// flatpak: "Installing 1/2…", "Updating 1/1…", "Uninstalling 1/1…".
	flatpakProgress = regexp.MustCompile(`^(Installing|Updating|Uninstalling)\s+(\d+)/(\d+)`)
)

// parseProgress reads a done/total count from one line of package-manager
// output.
func parseProgress(line string) (done, total int, ok bool) {
	var d, t string
	if m := pacmanProgress.FindStringSubmatch(line); m != nil {
		d, t = m[1], m[2]
	} else if m := dnf5Progress.FindStringSubmatch(line); m != nil {
		d, t = m[1], m[2]
	} else if m := dnf4Progress.FindStringSubmatch(line); m != nil {
		d, t = m[2], m[3]
	} else if m := flatpakProgress.FindStringSubmatch(line); m != nil {
		d, t = m[2], m[3]
	} else {
		return 0, 0, false
	}
	done, err1 := strconv.Atoi(d)
	total, err2 := strconv.Atoi(t)
	if err1 != nil || err2 != nil || total <= 0 || done < 0 || done > total {
		return 0, 0, false
	}
	return done, total, true
}

// errorTap passes writes through and remembers the last "error: ..." line, so
// a command that fails before its plan still closes the stream with a reason.
type errorTap struct {
	w       io.Writer
	mu      sync.Mutex
	partial []byte
	lastErr string
}

func (t *errorTap) Write(p []byte) (int, error) {
	t.mu.Lock()
	t.partial = append(t.partial, p...)
	for {
		i := bytes.IndexByte(t.partial, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimSpace(ansiEscape.ReplaceAllString(string(t.partial[:i]), ""))
		t.partial = t.partial[i+1:]
		if rest, ok := strings.CutPrefix(line, "error: "); ok {
			t.lastErr = rest
		}
	}
	t.mu.Unlock()
	return t.w.Write(p)
}

func (t *errorTap) last() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastErr
}
