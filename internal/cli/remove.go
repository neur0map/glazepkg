package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func init() {
	subcommands["remove"] = runRemove
}

func runRemove(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = reorderFlagsFirst(args, []string{"manager", "m"})
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag      = fs.String("manager", "", "manager to remove from; required if package is installed in multiple")
		withDepsFlag = fs.Bool("with-deps", false, "also remove orphaned dependencies (pacman, apt, dnf, xbps)")
		yesFlag      = fs.Bool("yes", false, "skip the y/N confirmation prompt")
		dryRunFlag   = fs.Bool("dry-run", false, "print the command(s) without executing")
		quietFlag    = fs.Bool("quiet", false, "suppress progress on stderr")
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
	pkgs, err := collectPackages(filtered, false, true, stderr, false /* never cache during write */)
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
			cmd = deep.RemoveCmdWithDeps(name)
		} else {
			cmd = remover.RemoveCmd(name)
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

	// Print plan and prompt.
	fmt.Fprintln(stdout, "The following commands will run:")
	for _, p := range plans {
		fmt.Fprintf(stdout, "  %s  →  %s\n", p.mgr.Name(), p.cmdStr)
		if p.warning != "" {
			fmt.Fprintln(stdout, p.warning)
		}
	}

	if *dryRunFlag {
		fmt.Fprintln(stdout, "(dry-run; nothing executed)")
		return ExitOK
	}

	if !*yesFlag {
		if !confirmAction("Proceed? [y/N] ", stdin, stdout) {
			fmt.Fprintln(stderr, "cancelled")
			return ExitOK
		}
	}

	// Execute. Stop on first failure.
	for _, p := range plans {
		if !*quietFlag {
			fmt.Fprintf(stderr, "removing %s via %s...\n", p.pkg.Name, p.mgr.Name())
		}
		var c *exec.Cmd
		if *withDepsFlag {
			c = p.mgr.(manager.DeepRemover).RemoveCmdWithDeps(p.pkg.Name)
		} else {
			c = p.remover.RemoveCmd(p.pkg.Name)
		}
		if err := headlessExec(c); err != nil {
			fmt.Fprintf(stderr, "error: remove %s failed: %v\n", p.pkg.Name, err)
			return ExitErr
		}
		invalidateAfterWrite(p.mgr, []model.Package{p.pkg})
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
