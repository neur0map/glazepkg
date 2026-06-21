package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/snapshot"
)

func init() {
	subcommands["history"] = runHistory
	subcommands["undo"] = runUndo
}

func runHistory(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonFlag := fs.Bool("json", false, "emit JSON envelope")
	limitFlag := fs.Int("limit", 20, "how many recent actions to show")
	args = reorderFlagsFirst(args, []string{"limit"})
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}

	items := snapshot.LoadHistory()
	if *jsonFlag {
		if items == nil {
			items = []snapshot.HistoryItem{}
		}
		if err := writeEnvelope(stdout, version, items); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	st := newStyler()
	if len(items) == 0 {
		fmt.Fprintln(stdout, st.dim("no history yet"))
		return ExitOK
	}

	fmt.Fprintln(stdout, st.title("Recent actions"))
	shown := 0
	for i := len(items) - 1; i >= 0 && shown < *limitFlag; i-- {
		it := items[i]
		target := it.Name
		if it.Version != "" {
			target += " " + st.version(it.Version)
		}
		fmt.Fprintf(stdout, "  %s  %s  %s  %s\n",
			st.dim(it.Time.Format("2006-01-02 15:04")),
			opStyle(st, it.Op),
			st.mgrName(it.Source),
			target)
		shown++
	}
	return ExitOK
}

func opStyle(st *styler, op snapshot.HistoryOp) string {
	label := padRight(string(op), 9)
	switch op {
	case snapshot.OpInstall:
		return st.ok(label)
	case snapshot.OpRemove:
		return st.bad(label)
	case snapshot.OpDowngrade:
		return st.warn(label)
	default:
		return st.dim(label)
	}
}

func runUndo(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("undo", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		yesFlag    = fs.Bool("yes", false, "skip the confirmation prompt")
		dryRunFlag = fs.Bool("dry-run", false, "print the command(s) without executing")
		quietFlag  = fs.Bool("quiet", false, "suppress progress on stderr")
	)
	fs.BoolVar(yesFlag, "y", false, "alias for --yes")
	fs.BoolVar(quietFlag, "q", false, "alias for --quiet")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}

	items := snapshot.LoadHistory()
	if len(items) == 0 {
		fmt.Fprintln(stderr, "nothing to undo")
		return ExitNegative
	}

	// The last command is the trailing run of items sharing a Group.
	group := items[len(items)-1].Group
	cut := len(items)
	for cut > 0 && items[cut-1].Group == group {
		cut--
	}
	last, remaining := items[cut:], items[:cut]

	st := newStyler()
	type undoStep struct {
		item snapshot.HistoryItem
		cmd  *exec.Cmd
	}
	var steps []undoStep
	for _, it := range last {
		mgr := mgrByName(mgrs, it.Source)
		if mgr == nil {
			continue
		}
		if cmd := inverseCmd(mgr, it, *yesFlag); cmd != nil {
			steps = append(steps, undoStep{item: it, cmd: cmd})
		}
	}
	if len(steps) == 0 {
		fmt.Fprintf(stderr, "the last action (%s) can't be undone automatically\n", last[0].Op)
		return ExitErr
	}

	fmt.Fprintln(stdout, st.title("Undo "+string(last[0].Op)))
	for _, s := range steps {
		fmt.Fprintf(stdout, "  %s  %s\n", st.mgrName(s.item.Source), s.item.Name)
		fmt.Fprintln(stdout, "      "+st.dim(displayCmd(s.cmd)))
	}

	if *dryRunFlag {
		fmt.Fprintln(stdout, st.dim("(dry-run; nothing executed)"))
		return ExitOK
	}

	r := newPromptReader(stdin)
	if !*yesFlag && !confirm(st.accent("==> proceed?")+" [y/N] ", r, stdout) {
		fmt.Fprintln(stderr, "cancelled")
		return ExitOK
	}

	failed := false
	for _, s := range steps {
		if !*quietFlag {
			fmt.Fprintf(stderr, "reverting %s...\n", s.item.Name)
		}
		if err := headlessExec(s.cmd); err != nil {
			fmt.Fprintf(stderr, "error: undo %s failed: %v\n", s.item.Name, err)
			failed = true
			break
		}
		invalidateAfterWrite(mgrByName(mgrs, s.item.Source), nil)
	}
	if failed {
		return ExitErr
	}
	_ = snapshot.SaveHistory(remaining)
	return ExitOK
}

// inverseCmd builds the command that reverses a recorded action, or nil when
// the action can't be undone automatically (notably plain upgrades).
func inverseCmd(mgr manager.Manager, it snapshot.HistoryItem, yes bool) *exec.Cmd {
	switch it.Op {
	case snapshot.OpInstall:
		if yes {
			if ni, ok := mgr.(manager.NonInteractiveRemover); ok {
				if cmd := ni.RemoveCmdYes(it.Name); cmd != nil {
					return cmd
				}
			}
		}
		if rm, ok := mgr.(manager.Remover); ok {
			return rm.RemoveCmd(it.Name)
		}
	case snapshot.OpRemove:
		if it.Version != "" {
			if v, ok := mgr.(manager.VersionedInstaller); ok {
				return v.InstallVersionCmd(it.Name, it.Version)
			}
		}
		if yes {
			if ni, ok := mgr.(manager.NonInteractiveInstaller); ok {
				if cmd := ni.InstallCmdYes(it.Name); cmd != nil {
					return cmd
				}
			}
		}
		if inst, ok := mgr.(manager.Installer); ok {
			return inst.InstallCmd(it.Name)
		}
	case snapshot.OpDowngrade:
		if it.PrevVersion != "" {
			return downgradeCmdFor(mgr, it.Name, it.PrevVersion)
		}
	}
	return nil
}

// nextGroup returns a monotonically increasing group id for a command's
// history items, derived from the wall clock.
func nextGroup() int64 {
	return time.Now().UnixNano()
}
