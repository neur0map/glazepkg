package cli

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

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
//
// verb is the gpk subcommand behind the run ("upgrade", "clean"), used in the
// retry hints. Every failure is restated in a summary at the end: by then the
// manager that broke is thousands of lines above, and a verdict nobody can
// scroll to is a verdict nobody reads.
func executeGrouped(title, verb string, rows []groupedCmd, dryRun, yes, quiet bool, st *styler, r *bufio.Reader, stdout, stderr io.Writer) int {
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

	if !yes {
		if ok, code := confirmProceed(st, r, stdout, stderr); !ok {
			return code
		}
	}

	var failures []stepFailure
	total := time.Now()
	for i, row := range rows {
		name := string(row.mgr.Name())
		if !quiet {
			fmt.Fprintf(stderr, "%s%s %s\n", st.accent(":: "),
				st.paint(stepPrefix(i, len(rows))+name, st.pal.White, true),
				st.dim("· "+displayCmd(row.cmd)))
		}
		took, err := runStep(row.cmd, name, st, stderr, quiet)
		if err != nil {
			reason := exitReason(err)
			failures = append(failures, stepFailure{name: name, cmd: displayCmd(row.cmd), reason: reason})
			fmt.Fprintf(stderr, "%s %s %s\n", st.bad("✗"), name,
				st.dim("· "+reason+" after "+humanDuration(took)+" — its output is above"))
			continue
		}
		// Autoremove doesn't know which packages went away, so pass nothing
		// and let the whole cache drop.
		invalidateAfterWrite(row.mgr, nil, nil)
		if !quiet {
			fmt.Fprintf(stderr, "%s %s %s\n", st.ok("✓"),
				st.paint(name, st.pal.White, true), st.dim("· "+humanDuration(took)))
		}
	}

	if len(failures) > 0 {
		writeFailureSummary(stderr, st, title, verb, len(rows), failures)
		return ExitErr
	}
	if !quiet && len(rows) > 1 {
		fmt.Fprintf(stderr, "%s %s\n", st.ok("✓"),
			st.dim(fmt.Sprintf("%s done in %s", plural(len(rows), "manager", "managers"), humanDuration(time.Since(total)))))
	}
	return ExitOK
}

// stepFailure is one step that didn't succeed, kept so the run can restate
// every failure once the noise has stopped.
type stepFailure struct {
	name   string // manager, or "package (manager)"
	cmd    string
	reason string
}

// stepPrefix numbers a step when there is more than one, so a run that stops
// halfway says where it stopped.
func stepPrefix(i, n int) string {
	if n < 2 {
		return ""
	}
	return fmt.Sprintf("[%d/%d] ", i+1, n)
}

// writeFailureSummary restates every failure once the run's output has stopped
// scrolling. Printed even under --quiet. The succeeded count matters: one
// habitually broken manager turns every `gpk -Syu` into exit 1, and without it
// a good upgrade with a known straggler looks like a total failure.
func writeFailureSummary(w io.Writer, st *styler, title, verb string, total int, failures []stepFailure) {
	names := make([]string, len(failures))
	for i, f := range failures {
		names[i] = f.name
	}
	fmt.Fprintf(w, "\n%s\n", st.bad(fmt.Sprintf("%s: %d of %d failed", title, len(failures), total)))
	for _, f := range failures {
		fmt.Fprintf(w, "  %s %s  %s\n", st.bad("✗"),
			st.paint(f.name, st.pal.White, true), st.dim(f.reason+" · "+f.cmd))
	}
	if ok := total - len(failures); ok > 0 {
		fmt.Fprintf(w, "  %s\n", st.dim(fmt.Sprintf("%s finished, and their changes are applied", plural(ok, "other manager", "other managers"))))
	}
	fmt.Fprintf(w, "  %s\n", st.dim("retry: gpk "+verb+" --manager "+strings.Join(names, ",")))
	fmt.Fprintf(w, "  %s\n", st.dim("skip:  gpk "+verb+" --manager '!"+strings.Join(names, ",!")+"'"))
}

// plural renders "1 manager" / "3 managers" without the caller branching.
func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
