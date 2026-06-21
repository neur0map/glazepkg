package cli

import (
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
	subcommands["remove"] = runRemove
}

func runRemove(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag      = fs.String("manager", "", "manager to remove from; required if package is installed in multiple")
		withDepsFlag = fs.Bool("with-deps", false, "also remove orphaned dependencies (pacman, apt, dnf, xbps)")
		yesFlag      = fs.Bool("yes", false, "skip the y/N confirmation prompt")
		dryRunFlag   = fs.Bool("dry-run", false, "print the command(s) without executing")
		quietFlag    = fs.Bool("quiet", false, "suppress progress on stderr")
		noCacheFlag  = fs.Bool("no-cache", false, "bypass the scan cache; do a fresh live scan")
		jsonFlag     = fs.Bool("json", false, "emit the resolved plan as JSON and exit (no execution)")
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

	names := fs.Args()
	if len(names) == 0 {
		fmt.Fprintln(stderr, "error: remove requires at least one package name")
		return ExitErr
	}

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	// Scan to find which installed manager owns each name.
	pkgs, err := collectPackages(filtered, *noCacheFlag, true, stderr, false)
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	type plan struct {
		pkg     model.Package
		mgr     manager.Manager
		remover manager.Remover
		cmdStr  string
		warning string
	}
	var plans []plan
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
			var srcs []string
			for _, m := range matches {
				srcs = append(srcs, string(m.Source))
			}
			fmt.Fprintf(stderr, "error: %q installed in %d managers (%s); use --manager to pick\n",
				name, len(matches), strings.Join(srcs, ", "))
			return ExitAmbiguous
		}
		pkg := matches[0]
		mgr := mgrByName(filtered, pkg.Source)
		if mgr == nil {
			fmt.Fprintf(stderr, "error: manager %s not in filtered set\n", pkg.Source)
			return ExitErr
		}
		remover, ok := mgr.(manager.Remover)
		if !ok {
			fmt.Fprintf(stderr, "error: %s does not support remove\n", mgr.Name())
			return ExitErr
		}

		var cmd *exec.Cmd
		if *withDepsFlag {
			deep, ok := mgr.(manager.DeepRemover)
			if !ok {
				fmt.Fprintf(stderr, "error: %s does not support --with-deps\n", mgr.Name())
				return ExitErr
			}
			if *yesFlag {
				if ni, ok := mgr.(manager.NonInteractiveDeepRemover); ok {
					cmd = ni.RemoveCmdWithDepsYes(name)
				}
			}
			if cmd == nil {
				cmd = deep.RemoveCmdWithDeps(name)
			}
		} else {
			if *yesFlag {
				if ni, ok := mgr.(manager.NonInteractiveRemover); ok {
					cmd = ni.RemoveCmdYes(name)
				}
			}
			if cmd == nil {
				cmd = remover.RemoveCmd(name)
			}
		}

		if cmd == nil {
			if *withDepsFlag {
				fmt.Fprintf(stderr, "error: %s does not support --with-deps for %q\n", mgr.Name(), name)
			} else {
				fmt.Fprintf(stderr, "error: %s returned no remove command for %q\n", mgr.Name(), name)
			}
			return ExitErr
		}

		cmdDisplay := strings.Join(stripSudoStdinFlag(cmd).Args, " ")

		warning := ""
		if len(pkg.RequiredBy) > 0 {
			req := strings.Join(pkg.RequiredBy, ", ")
			if len(req) > 60 {
				req = req[:60] + "..."
			}
			warning = "  ⚠ required by: " + req
		}

		plans = append(plans, plan{pkg: pkg, mgr: mgr, remover: remover, cmdStr: cmdDisplay, warning: warning})
	}

	if *jsonFlag {
		steps := make([]planStep, 0, len(plans))
		for _, p := range plans {
			steps = append(steps, planStep{Manager: string(p.mgr.Name()), Name: p.pkg.Name, Version: p.pkg.Version, Command: cmdArgs(removeCmdFor(p.mgr, p.pkg.Name, *withDepsFlag, *yesFlag))})
		}
		if err := emitPlanJSON(stdout, version, "remove", steps); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	st := newStyler()
	r := newPromptReader(stdin)

	fmt.Fprintln(stdout, st.title("Remove plan"))
	for _, p := range plans {
		fmt.Fprintf(stdout, "  %s  %s\n", st.mgrName(p.mgr.Name()), p.pkg.Name)
		fmt.Fprintln(stdout, "      "+st.dim(p.cmdStr))
		if p.warning != "" {
			fmt.Fprintln(stdout, "      "+st.warn(strings.TrimSpace(p.warning)))
		}
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
			fmt.Fprintf(stderr, "removing %s via %s...\n", p.pkg.Name, p.mgr.Name())
		}
		var c *exec.Cmd
		if *withDepsFlag {
			if *yesFlag {
				if ni, ok := p.mgr.(manager.NonInteractiveDeepRemover); ok {
					c = ni.RemoveCmdWithDepsYes(p.pkg.Name)
				}
			}
			if c == nil {
				c = p.mgr.(manager.DeepRemover).RemoveCmdWithDeps(p.pkg.Name)
			}
		} else {
			if *yesFlag {
				if ni, ok := p.mgr.(manager.NonInteractiveRemover); ok {
					c = ni.RemoveCmdYes(p.pkg.Name)
				}
			}
			if c == nil {
				c = p.remover.RemoveCmd(p.pkg.Name)
			}
		}
		if err := headlessExec(c); err != nil {
			fmt.Fprintf(stderr, "error: remove %s failed: %v\n", p.pkg.Name, err)
			return ExitErr
		}
		invalidateAfterWrite(p.mgr, []model.Package{p.pkg})
		_ = snapshot.AppendHistory(snapshot.HistoryItem{
			Group: grp, Time: time.Now(), Op: snapshot.OpRemove,
			Source: p.mgr.Name(), Name: p.pkg.Name, Version: p.pkg.Version,
		})
	}
	return ExitOK
}

// mgrByName finds a manager by its Source name within a filtered set.
func mgrByName(filtered []manager.Manager, src model.Source) manager.Manager {
	for _, m := range filtered {
		if m.Name() == src {
			return m
		}
	}
	return nil
}

func removeCmdFor(mgr manager.Manager, name string, withDeps, yes bool) *exec.Cmd {
	if withDeps {
		if yes {
			if ni, ok := mgr.(manager.NonInteractiveDeepRemover); ok {
				if c := ni.RemoveCmdWithDepsYes(name); c != nil {
					return c
				}
			}
		}
		if d, ok := mgr.(manager.DeepRemover); ok {
			return d.RemoveCmdWithDeps(name)
		}
		return nil
	}
	if yes {
		if ni, ok := mgr.(manager.NonInteractiveRemover); ok {
			if c := ni.RemoveCmdYes(name); c != nil {
				return c
			}
		}
	}
	if rm, ok := mgr.(manager.Remover); ok {
		return rm.RemoveCmd(name)
	}
	return nil
}
