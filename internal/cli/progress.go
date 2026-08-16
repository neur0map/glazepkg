package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
)

// gpk reports progress two ways because it shares the terminal two ways.
//
// While a package manager runs, the child owns the terminal — gpk passes its
// own stdin/stdout/stderr straight through so sudo prompts and download bars
// keep working — so gpk cannot see what the child prints and must not animate
// over it. A step logs whole "still running" lines at widening intervals.
//
// While gpk itself works (scanning, checking updates) the subprocesses have
// their output captured, so a self-erasing animated line is safe.

var spinnerFrames = [...]string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// heartbeatDelays widen so a slow-but-healthy upgrade stays readable; the last
// gap repeats forever.
var heartbeatDelays = [...]time.Duration{
	20 * time.Second,
	40 * time.Second,
	1 * time.Minute,
	2 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
}

const spinnerInterval = 120 * time.Millisecond

// step reports that a child holding the terminal is still alive. A nil *step
// is inert.
type step struct {
	stop chan struct{}
	done chan struct{}
}

// startStep begins reporting on a running child. cmdText is the command as the
// user would type it, so a wedged step can be identified and re-run.
func startStep(w io.Writer, st *styler, label, cmdText string) *step {
	if w == nil {
		return nil
	}
	s := &step{stop: make(chan struct{}), done: make(chan struct{})}
	go s.run(w, st, label, cmdText)
	return s
}

func (s *step) run(w io.Writer, st *styler, label, cmdText string) {
	defer close(s.done)
	started := time.Now()
	timer := time.NewTimer(heartbeatDelays[0])
	defer timer.Stop()
	for i := 0; ; i++ {
		select {
		case <-s.stop:
			return
		case <-timer.C:
		}
		hint := ""
		if i == 0 {
			hint = " · Ctrl-C aborts"
		}
		fmt.Fprintf(w, "%s%s %s\n", st.accent(":: "), st.paint(label, st.pal.White, true),
			st.dim("still running ("+humanDuration(time.Since(started))+") — "+cmdText+hint))
		next := heartbeatDelays[len(heartbeatDelays)-1]
		if i+1 < len(heartbeatDelays) {
			next = heartbeatDelays[i+1]
		}
		timer.Reset(next)
	}
}

// finish stops the reporter. Safe on a nil *step and safe to call once.
func (s *step) finish() {
	if s == nil {
		return
	}
	close(s.stop)
	<-s.done
}

// holdInterrupt makes gpk ignore Ctrl-C while a child owns the terminal. The
// signal still reaches the child through the foreground process group; without
// this gpk exits first, skips the failure summary, and leaves a live `pacman`
// writing to a terminal its parent no longer owns. A second Ctrl-C abandons it.
func holdInterrupt(w io.Writer, st *styler, label string) (release func()) {
	sig := make(chan os.Signal, 4)
	signal.Notify(sig, os.Interrupt)
	stop := make(chan struct{})
	go func() {
		for n := 0; ; n++ {
			select {
			case <-stop:
				return
			case <-sig:
			}
			if n > 0 {
				fmt.Fprintf(w, "\n%s %s\n", st.bad("✗"),
					st.dim("abandoning "+label+"; it may still be running in the background"))
				os.Exit(ExitInterrupted)
			}
			fmt.Fprintf(w, "\n%s %s\n", st.warn("interrupt"),
				st.dim("passed to "+label+"; press Ctrl-C again to quit gpk and leave it running"))
		}
	}()
	return func() {
		signal.Stop(sig)
		close(stop)
	}
}

// spinner animates one self-erasing line while gpk itself is working. Safe only
// where no child writes to the terminal; a non-terminal writer degrades to one
// plain line per label. The goroutine owns the line, so caller output goes
// through note rather than straight to the writer.
type spinner struct {
	w      io.Writer
	labels chan string
	notes  chan string
	quit   chan struct{}
	done   chan struct{}
	once   sync.Once
	plain  bool // w isn't a terminal: log lines instead of animating
	st     *styler
}

// startSpinner begins animating label. Returns nil when progress is
// suppressed, which every method tolerates.
func startSpinner(w io.Writer, st *styler, label string) *spinner {
	if w == nil {
		return nil
	}
	s := &spinner{
		w:      w,
		labels: make(chan string, 8),
		notes:  make(chan string, 8),
		quit:   make(chan struct{}),
		done:   make(chan struct{}),
		st:     st,
	}
	if !isTerminalWriter(w) {
		s.plain = true
		close(s.done)
		s.update(label)
		return s
	}
	go s.run(label)
	return s
}

func (s *spinner) run(label string) {
	defer close(s.done)
	started := time.Now()
	tick := time.NewTicker(spinnerInterval)
	defer tick.Stop()
	for frame := 0; ; frame++ {
		select {
		case <-s.quit:
			eraseLine(s.w)
			return
		case l := <-s.labels:
			label = l
		case msg := <-s.notes:
			eraseLine(s.w)
			fmt.Fprintln(s.w, msg)
		case <-tick.C:
		}
		fmt.Fprintf(s.w, "\r\033[2K%s %s %s", s.st.accent(spinnerFrames[frame%len(spinnerFrames)]),
			label, s.st.dim(humanDuration(time.Since(started))))
	}
}

// update switches the announced label, e.g. to the manager now being scanned.
// An empty label announces nothing, so a caller that names its first real step
// immediately doesn't have to emit a placeholder first.
func (s *spinner) update(label string) {
	if s == nil || label == "" {
		return
	}
	if s.plain {
		fmt.Fprintf(s.w, "%s...\n", label)
		return
	}
	select {
	case s.labels <- label:
	case <-s.done:
	default: // a stale label is not worth blocking a scan for
	}
}

// note prints one line without tearing the animation, by handing it to the
// goroutine that owns the terminal line. Falls back to a direct write once the
// animation has stopped and nothing owns that line any more.
func (s *spinner) note(msg string) {
	if s == nil {
		return
	}
	if s.plain {
		fmt.Fprintln(s.w, msg)
		return
	}
	select {
	case s.notes <- msg:
	case <-s.done:
		fmt.Fprintln(s.w, msg)
	}
}

// stop erases the animated line. Safe on a nil *spinner and on repeat calls,
// so callers can both defer it and stop early.
func (s *spinner) stop() {
	if s == nil {
		return
	}
	s.once.Do(func() { close(s.quit) })
	<-s.done
}

// eraseLine clears the current terminal line and parks the cursor at column 0
// so the next real output starts on clean ground.
func eraseLine(w io.Writer) {
	fmt.Fprint(w, "\r\033[2K")
}

// isTerminalWriter reports whether w is a terminal gpk may animate on.
func isTerminalWriter(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	return ok && isatty.IsTerminal(f.Fd())
}

// humanDuration formats an elapsed time for progress output: sub-minute in
// seconds, then m/s, then h/m. Always at most two units, never zero-padded
// beyond what reads naturally in a log line.
func humanDuration(d time.Duration) string {
	switch {
	case d < 10*time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
}
