package cli

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"

	"github.com/neur0map/glazepkg/internal/manager"
)

// groupedCmd is one manager's command in a multi-manager operation (clean,
// autoremove, upgrade-all). detail holds optional lines shown under it.
type groupedCmd struct {
	mgr    manager.Manager
	cmd    *exec.Cmd
	detail []string
}

// executeGrouped renders a themed plan of per-manager commands, prompts unless
// yes, then runs each. A failure in one manager is reported but doesn't abort
// the rest. Callers handle the empty-rows messaging themselves.
func executeGrouped(title string, rows []groupedCmd, dryRun, yes, quiet bool, st *styler, r *bufio.Reader, stdout, stderr io.Writer) int {
	fmt.Fprintln(stdout, st.title(title))
	for _, row := range rows {
		fmt.Fprintf(stdout, "  %s  %s\n", st.mgrName(row.mgr.Name()), st.dim(displayCmd(row.cmd)))
		for _, d := range row.detail {
			fmt.Fprintln(stdout, "      "+st.dim(d))
		}
	}

	if dryRun {
		fmt.Fprintln(stdout, st.dim("(dry-run; nothing executed)"))
		return ExitOK
	}

	if !yes && !confirm(st.accent("==> proceed?")+" [y/N] ", r, stdout) {
		fmt.Fprintln(stderr, "cancelled")
		return ExitOK
	}

	failed := 0
	for _, row := range rows {
		if !quiet {
			fmt.Fprintln(stderr, st.accent(":: ")+st.paint(string(row.mgr.Name()), st.pal.White, true))
		}
		if err := headlessExec(row.cmd); err != nil {
			fmt.Fprintln(stderr, st.bad("✗")+" "+string(row.mgr.Name())+st.dim(" reported an error (details above)"))
			failed++
			continue
		}
		invalidateAfterWrite(row.mgr, nil)
		if !quiet {
			fmt.Fprintln(stderr, st.ok("✓")+" "+st.paint(string(row.mgr.Name()), st.pal.White, true))
		}
	}
	if failed > 0 {
		return ExitErr
	}
	return ExitOK
}
