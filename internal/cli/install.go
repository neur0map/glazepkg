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
	subcommands["install"] = runInstall
}

func runInstall(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = reorderFlagsFirst(args, []string{"manager", "m"})
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag    = fs.String("manager", "", "manager to install from; required if package is available in multiple managers")
		yesFlag    = fs.Bool("yes", false, "skip the y/N confirmation prompt")
		dryRunFlag = fs.Bool("dry-run", false, "print the command(s) without executing")
		quietFlag  = fs.Bool("quiet", false, "suppress progress messages on stderr")
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
		fmt.Fprintln(stderr, "error: install requires at least one package name")
		return ExitErr
	}

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	// Resolve manager + InstallCmd for each name.
	type plan struct {
		pkg string
		mgr manager.Manager
		cmd *strings.Builder // for display
	}
	var plans []plan
	for _, name := range names {
		chosen, err := resolveInstallManager(name, filtered)
		if err != nil {
			return reportResolveErr(err, stderr)
		}

		installer, ok := chosen.(manager.Installer)
		if !ok {
			fmt.Fprintf(stderr, "error: %s does not support install\n", chosen.Name())
			return ExitErr
		}

		var cmd *exec.Cmd
		if *yesFlag {
			if ni, ok := chosen.(manager.NonInteractiveInstaller); ok {
				cmd = ni.InstallCmdYes(name)
			}
		}
		if cmd == nil {
			cmd = installer.InstallCmd(name)
		}
		if cmd == nil {
			fmt.Fprintf(stderr, "error: %s returned no install command for %q\n", chosen.Name(), name)
			return ExitErr
		}
		var disp strings.Builder
		disp.WriteString(strings.Join(stripSudoStdinFlag(cmd).Args, " "))
		plans = append(plans, plan{pkg: name, mgr: chosen, cmd: &disp})
	}

	// Show the plan and (unless --yes) prompt.
	fmt.Fprintln(stdout, "The following commands will run:")
	for _, p := range plans {
		fmt.Fprintf(stdout, "  %s  →  %s\n", p.mgr.Name(), p.cmd.String())
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
			fmt.Fprintf(stderr, "installing %s via %s...\n", p.pkg, p.mgr.Name())
		}
		var cmd *exec.Cmd
		if *yesFlag {
			if ni, ok := p.mgr.(manager.NonInteractiveInstaller); ok {
				cmd = ni.InstallCmdYes(p.pkg)
			}
		}
		if cmd == nil {
			installer := p.mgr.(manager.Installer)
			cmd = installer.InstallCmd(p.pkg)
		}
		if err := headlessExec(cmd); err != nil {
			fmt.Fprintf(stderr, "error: install %s failed: %v\n", p.pkg, err)
			return ExitErr
		}
		invalidateAfterWrite(p.mgr, []model.Package{{Name: p.pkg, Source: p.mgr.Name()}})
	}
	return ExitOK
}

// reportResolveErr maps the sentinel errors from resolveInstallManager to
// exit codes and writes the error to stderr.
func reportResolveErr(err error, stderr io.Writer) int {
	fmt.Fprintf(stderr, "error: %v\n", err)
	switch {
	case errors.Is(err, ErrAmbiguous):
		return ExitAmbiguous
	case errors.Is(err, ErrNotFound):
		return ExitNegative
	default:
		return ExitErr
	}
}
