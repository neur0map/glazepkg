package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/neur0map/glazepkg/internal/manager"
)

func init() {
	subcommands["clean"] = runClean
}

func runClean(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag    = fs.String("manager", "", "comma list of managers (default: all)")
		allFlag    = fs.Bool("all", false, "remove every cached download, not just the stale ones")
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

	var rows []groupedCmd
	for _, m := range filtered {
		if !m.Available() {
			continue
		}
		c, ok := m.(manager.CacheCleaner)
		if !ok {
			continue
		}
		if cmd := c.CleanCacheCmd(*allFlag, *yesFlag); cmd != nil {
			rows = append(rows, groupedCmd{mgr: m, cmd: cmd})
		}
	}
	if len(rows) == 0 {
		fmt.Fprintln(stderr, "no installed managers support cache cleaning")
		return ExitOK
	}

	st := newStyler()
	r := newPromptReader(stdin)
	return executeGrouped("Clean caches", rows, *dryRunFlag, *yesFlag, *quietFlag, st, r, stdout, stderr)
}
