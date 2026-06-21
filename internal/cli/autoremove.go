package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/neur0map/glazepkg/internal/manager"
)

func init() {
	subcommands["autoremove"] = runAutoremove
}

func runAutoremove(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("autoremove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag    = fs.String("manager", "", "comma list of managers (default: all)")
		printFlag  = fs.Bool("print", false, "list orphaned packages without removing them")
		yesFlag    = fs.Bool("yes", false, "skip the confirmation prompt")
		dryRunFlag = fs.Bool("dry-run", false, "print the command(s) without executing")
		quietFlag  = fs.Bool("quiet", false, "suppress progress on stderr")
	)
	fs.BoolVar(yesFlag, "y", false, "alias for --yes")
	fs.BoolVar(quietFlag, "q", false, "alias for --quiet")
	fs.StringVar(mgrFlag, "m", "", "alias for --manager")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	st := newStyler()

	type found struct {
		mgr     manager.Manager
		orphans []string
	}
	var all []found
	for _, m := range filtered {
		if !m.Available() {
			continue
		}
		o, ok := m.(manager.Orphaner)
		if !ok {
			continue
		}
		orphans, _ := o.Orphans()
		all = append(all, found{mgr: m, orphans: orphans})
	}

	if *printFlag {
		any := false
		for _, f := range all {
			for _, name := range f.orphans {
				fmt.Fprintf(stdout, "%s  %s\n", st.mgrName(f.mgr.Name()), name)
				any = true
			}
		}
		if !any {
			fmt.Fprintln(stdout, st.dim("no orphaned packages found"))
		}
		return ExitOK
	}

	var rows []groupedCmd
	for _, f := range all {
		o := f.mgr.(manager.Orphaner)
		cmd := o.RemoveOrphansCmd(f.orphans, *yesFlag)
		if cmd == nil {
			continue
		}
		var detail []string
		if len(f.orphans) > 0 {
			detail = []string{strings.Join(f.orphans, " ")}
		}
		rows = append(rows, groupedCmd{mgr: f.mgr, cmd: cmd, detail: detail})
	}
	if len(rows) == 0 {
		fmt.Fprintln(stderr, "no orphaned packages to remove")
		return ExitOK
	}

	r := newPromptReader(stdin)
	return executeGrouped("Remove orphans", rows, *dryRunFlag, *yesFlag, *quietFlag, st, r, stdout, stderr)
}
