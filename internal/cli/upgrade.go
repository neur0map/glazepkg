package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
	"github.com/neur0map/glazepkg/internal/snapshot"
)

func init() {
	subcommands["upgrade"] = runUpgrade
}

func runUpgrade(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag     = fs.String("manager", "", "manager to upgrade from; required if a name is installed in multiple")
		yesFlag     = fs.Bool("yes", false, "skip the confirmation prompt")
		dryRunFlag  = fs.Bool("dry-run", false, "print the command(s) without executing")
		quietFlag   = fs.Bool("quiet", false, "suppress progress on stderr")
		noCacheFlag = fs.Bool("no-cache", false, "bypass the scan cache; do a fresh live scan")
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
	r := newPromptReader(stdin)

	names := fs.Args()
	if len(names) == 0 {
		return runUpgradeAll(filtered, *yesFlag, *dryRunFlag, *quietFlag, st, r, stdout, stderr)
	}

	pkgs, err := collectPackages(filtered, *noCacheFlag, true, stderr, false)
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	type plan struct {
		pkg model.Package
		mgr manager.Manager
	}
	var plans []plan
	holds := snapshot.LoadHolds()
	for _, name := range names {
		var matches []model.Package
		for _, p := range pkgs {
			if p.Name == name {
				matches = append(matches, p)
			}
		}
		if len(matches) == 0 {
			fmt.Fprintf(stderr, "error: %q is not installed in any searched manager\n", name)
			return ExitNegative
		}
		if len(matches) > 1 {
			srcs := make([]string, len(matches))
			for i, m := range matches {
				srcs[i] = string(m.Source)
			}
			fmt.Fprintf(stderr, "error: %q installed in %d managers (%s); use --manager to pick\n",
				name, len(matches), strings.Join(srcs, ", "))
			return ExitAmbiguous
		}
		if snapshot.IsHeld(holds, matches[0].Source, name) {
			if !*quietFlag {
				fmt.Fprintf(stderr, "%s is held; skipping (run `gpk unhold %s` to upgrade it)\n", name, name)
			}
			continue
		}
		mgr := mgrByName(filtered, matches[0].Source)
		if mgr == nil {
			fmt.Fprintf(stderr, "error: manager %s not in filtered set\n", matches[0].Source)
			return ExitErr
		}
		if upgradeCmdFor(mgr, name, *yesFlag) == nil {
			fmt.Fprintf(stderr, "error: %s cannot upgrade %s\n", mgr.Name(), name)
			return ExitErr
		}
		plans = append(plans, plan{pkg: matches[0], mgr: mgr})
	}
	if len(plans) == 0 {
		fmt.Fprintln(stderr, "nothing to upgrade")
		return ExitOK
	}

	fmt.Fprintln(stdout, st.title("Upgrade plan"))
	for _, p := range plans {
		cmd := upgradeCmdFor(p.mgr, p.pkg.Name, *yesFlag)
		fmt.Fprintf(stdout, "  %s  %s\n", st.mgrName(p.mgr.Name()), p.pkg.Name)
		fmt.Fprintln(stdout, "      "+st.dim(displayCmd(cmd)))
	}

	if *dryRunFlag {
		fmt.Fprintln(stdout, st.dim("(dry-run; nothing executed)"))
		return ExitOK
	}

	if !*yesFlag && !confirm(st.accent("==> proceed?")+" [y/N] ", r, stdout) {
		fmt.Fprintln(stderr, "cancelled")
		return ExitOK
	}

	grp := nextGroup()
	for _, p := range plans {
		if !*quietFlag {
			fmt.Fprintf(stderr, "upgrading %s via %s...\n", p.pkg.Name, p.mgr.Name())
		}
		cmd := upgradeCmdFor(p.mgr, p.pkg.Name, *yesFlag)
		if err := headlessExec(cmd); err != nil {
			fmt.Fprintf(stderr, "error: upgrade %s failed: %v\n", p.pkg.Name, err)
			return ExitErr
		}
		invalidateAfterWrite(p.mgr, []model.Package{p.pkg})
		_ = snapshot.AppendHistory(snapshot.HistoryItem{
			Group: grp, Time: time.Now(), Op: snapshot.OpUpgrade,
			Source: p.mgr.Name(), Name: p.pkg.Name,
		})
	}
	return ExitOK
}

func upgradeCmdFor(mgr manager.Manager, name string, yes bool) *exec.Cmd {
	if yes {
		if ni, ok := mgr.(manager.NonInteractiveUpgrader); ok {
			if cmd := ni.UpgradeCmdYes(name); cmd != nil {
				return cmd
			}
		}
	}
	if up, ok := mgr.(manager.Upgrader); ok {
		return up.UpgradeCmd(name)
	}
	return nil
}

// runUpgradeAll runs each available manager's bulk upgrade command — the
// `gpk upgrade` / `-Syu` "bring everything up to date" path. A failure in one
// manager doesn't stop the others.
func runUpgradeAll(filtered []manager.Manager, yes, dryRun, quiet bool, st *styler, r *bufio.Reader, stdout, stderr io.Writer) int {
	holds := snapshot.LoadHolds()
	var rows []groupedCmd
	for _, m := range filtered {
		if !m.Available() {
			continue
		}
		b, ok := m.(manager.BulkUpgrader)
		if !ok {
			continue
		}
		var cmd *exec.Cmd
		held := snapshot.HeldNames(holds, m.Name())
		if ig, ok := m.(manager.IgnoringBulkUpgrader); ok && len(held) > 0 {
			cmd = ig.UpgradeAllCmdIgnoring(yes, held)
		} else {
			cmd = b.UpgradeAllCmd(yes)
		}
		if cmd == nil {
			continue
		}
		var detail []string
		if len(held) > 0 {
			detail = []string{st.dim("holding: " + strings.Join(held, " "))}
		}
		rows = append(rows, groupedCmd{mgr: m, cmd: cmd, detail: detail})
	}
	if len(rows) == 0 {
		fmt.Fprintln(stderr, "no installed managers support bulk upgrade")
		return ExitOK
	}
	return executeGrouped("Upgrade everything", rows, dryRun, yes, quiet, st, r, stdout, stderr)
}
