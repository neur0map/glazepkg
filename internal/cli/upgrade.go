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
	subcommands["upgrade"] = runUpgrade
}

func runUpgrade(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = reorderFlagsFirst(args, []string{"manager", "m"})
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag    = fs.String("manager", "", "manager to upgrade from; required if package is installed in multiple")
		yesFlag    = fs.Bool("yes", false, "skip the y/N confirmation prompt")
		dryRunFlag = fs.Bool("dry-run", false, "print the command(s) without executing")
		quietFlag  = fs.Bool("quiet", false, "suppress progress on stderr")
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

	names := fs.Args()
	if len(names) == 0 {
		fmt.Fprintln(stderr, "error: upgrade requires at least one package name")
		return ExitErr
	}

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	pkgs, err := collectPackages(filtered, *noCacheFlag, true, stderr, false)
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	type plan struct {
		pkg      model.Package
		mgr      manager.Manager
		upgrader manager.Upgrader
		cmd      *exec.Cmd
		cmdStr   string
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
		upgrader, ok := mgr.(manager.Upgrader)
		if !ok {
			fmt.Fprintf(stderr, "error: %s does not support upgrade\n", mgr.Name())
			return ExitErr
		}
		cmd := upgrader.UpgradeCmd(name)
		if cmd == nil {
			fmt.Fprintf(stderr, "error: %s returned no upgrade command for %q\n", mgr.Name(), name)
			return ExitErr
		}
		cmdStr := strings.Join(stripSudoStdinFlag(cmd).Args, " ")
		plans = append(plans, plan{pkg: pkg, mgr: mgr, upgrader: upgrader, cmd: cmd, cmdStr: cmdStr})
	}

	fmt.Fprintln(stdout, "The following commands will run:")
	for _, p := range plans {
		fmt.Fprintf(stdout, "  %s  →  %s\n", p.mgr.Name(), p.cmdStr)
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

	for _, p := range plans {
		if !*quietFlag {
			fmt.Fprintf(stderr, "upgrading %s via %s...\n", p.pkg.Name, p.mgr.Name())
		}
		cmd := p.upgrader.UpgradeCmd(p.pkg.Name)
		if err := headlessExec(cmd); err != nil {
			fmt.Fprintf(stderr, "error: upgrade %s failed: %v\n", p.pkg.Name, err)
			return ExitErr
		}
		invalidateAfterWrite(p.mgr, []model.Package{p.pkg})
	}
	return ExitOK
}
