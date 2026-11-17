//go:build unix

package cli

import (
	"bytes"
	"sync"
	"syscall"
	"testing"
)

// signalOnLine is a stdout that sends SIGTERM to gpk itself once a log event
// carrying marker goes by, standing in for a GUI pressing Cancel.
type signalOnLine struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	marker string
	sent   bool
}

func (s *signalOnLine) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.buf.Write(p)
	if !s.sent && bytes.Contains(p, []byte(s.marker)) {
		s.sent = true
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}
	return n, err
}

func TestEventsCancelForwardsSignal(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GPK_PRIVILEGE", "")
	mgrs := eventFakeMgrs(func(verb, name string) string {
		return `echo child-ready; exec sleep 30`
	})
	out := &signalOnLine{marker: "child-ready"}
	var errOut bytes.Buffer
	code := Dispatch([]string{"install", "foo", "baz", "-m", "pacman", "--yes", "--events"}, mgrs, "test", out, &errOut, nil)
	if code != ExitInterrupted {
		t.Fatalf("exit %d, want %d (stderr=%q)", code, ExitInterrupted, errOut.String())
	}
	evs := decodeEvents(t, out.buf.String())
	if got, want := eventKinds(evs), "plan step:start log step:done done"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if evs[3]["exit"] != float64(128+int(syscall.SIGTERM)) {
		t.Errorf("step exit = %v, want %d", evs[3]["exit"], 128+int(syscall.SIGTERM))
	}
	last := evs[4]
	if last["ok"] != false || last["exit"] != float64(ExitInterrupted) || last["error"] != "cancelled" {
		t.Errorf("final done = %v", last)
	}
}
