package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
	"github.com/neur0map/glazepkg/internal/snapshot"
	vercmp "github.com/neur0map/glazepkg/internal/version"
)

func init() {
	subcommands["downgrade"] = runDowngrade
}

func runDowngrade(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("downgrade", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag    = fs.String("manager", "", "manager to downgrade in")
		yesFlag    = fs.Bool("yes", false, "skip the confirmation prompt")
		dryRunFlag = fs.Bool("dry-run", false, "print the command without executing")
		quietFlag  = fs.Bool("quiet", false, "suppress progress on stderr")
		jsonFlag   = fs.Bool("json", false, "emit the resolved plan as JSON and exit (no execution)")
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

	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "error: downgrade takes exactly one package (optionally name@version)")
		return ExitErr
	}
	name, ver := splitVersionPin(rest[0])

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	st := newStyler()
	r := newPromptReader(stdin)
	if *jsonFlag {
		r = nil // non-interactive: requires name@version
	}

	mgr, curVer, code, ok := resolveDowngradeManager(name, filtered, st, r, stdout, stderr)
	if !ok {
		return code
	}

	if ver == "" {
		ver, code, ok = chooseVersion(name, mgr, st, r, stdout, stderr)
		if !ok {
			return code
		}
	}

	cmd := downgradeCmdFor(mgr, name, ver)
	if cmd == nil {
		fmt.Fprintf(stderr, "error: %s cannot downgrade %s to %s (version unavailable)\n", mgr.Name(), name, ver)
		return ExitErr
	}

	if *jsonFlag {
		step := planStep{Manager: string(mgr.Name()), Name: name, Version: ver, Command: cmdArgs(cmd)}
		if err := emitPlanJSON(stdout, version, "downgrade", []planStep{step}); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	fmt.Fprintln(stdout, st.title("Downgrade plan"))
	fmt.Fprintf(stdout, "  %s  %s %s\n", st.mgrName(mgr.Name()), name, st.version("@"+ver))
	fmt.Fprintln(stdout, "      "+st.dim(displayCmd(cmd)))

	if *dryRunFlag {
		fmt.Fprintln(stdout, st.dim("(dry-run; nothing executed)"))
		return ExitOK
	}

	if !*yesFlag && !confirm(st.accent("==> proceed?")+" [y/N] ", r, stdout) {
		fmt.Fprintln(stderr, "cancelled")
		return ExitOK
	}

	if !*quietFlag {
		fmt.Fprintln(stderr, st.accent(":: ")+"downgrading "+st.paint(name, st.pal.White, true)+st.dim(" to "+ver+" via "+string(mgr.Name())))
	}
	if err := headlessExec(cmd); err != nil {
		fmt.Fprintln(stderr, st.bad("✗")+" "+name+st.dim(" — "+string(mgr.Name())+" reported an error (details above)"))
		return ExitErr
	}
	invalidateAfterWrite(mgr, []model.Package{{Name: name, Source: mgr.Name()}})
	_ = snapshot.AppendHistory(snapshot.HistoryItem{
		Group: nextGroup(), Time: time.Now(), Op: snapshot.OpDowngrade,
		Source: mgr.Name(), Name: name, Version: ver, PrevVersion: curVer,
	})
	if !*quietFlag {
		fmt.Fprintln(stderr, st.ok("✓")+" "+st.paint(name, st.pal.White, true)+st.dim(" downgraded to "+ver))
	}
	return ExitOK
}

// resolveDowngradeManager finds the installed owner of name. When several
// managers own it, it asks (interactive) or returns exit-3 (scripted).
func resolveDowngradeManager(name string, filtered []manager.Manager, st *styler, r *bufio.Reader, stdout, stderr io.Writer) (manager.Manager, string, int, bool) {
	pkgs, err := collectPackages(filtered, false, true, stderr, false)
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return nil, "", ExitErr, false
	}
	var owned []model.Package
	for _, p := range pkgs {
		if p.Name == name {
			owned = append(owned, p)
		}
	}
	switch len(owned) {
	case 0:
		fmt.Fprintf(stderr, "error: %q is not installed\n", name)
		return nil, "", ExitNegative, false
	case 1:
		return mgrByName(filtered, owned[0].Source), owned[0].Version, ExitOK, true
	default:
		if !canPrompt(r) {
			srcs := make([]string, len(owned))
			for i, p := range owned {
				srcs[i] = string(p.Source)
			}
			fmt.Fprintf(stderr, "error: %q installed in %d managers (%s); use --manager\n",
				name, len(owned), strings.Join(srcs, ", "))
			return nil, "", ExitAmbiguous, false
		}
		fmt.Fprintf(stdout, "%s is installed by %d managers:\n", st.accent(name), len(owned))
		for i, p := range owned {
			fmt.Fprintf(stdout, "  %s %s\n", st.num(strconv.Itoa(i+1)+")"), st.mgrName(p.Source))
		}
		input, _ := readSelection(st.accent("==> ")+"manager [1-"+strconv.Itoa(len(owned))+"]: ", r, stdout)
		idxs, perr := parseSelection(input, len(owned))
		if perr != nil || len(idxs) == 0 {
			fmt.Fprintln(stderr, "cancelled")
			return nil, "", ExitOK, false
		}
		chosen := owned[idxs[0]]
		return mgrByName(filtered, chosen.Source), chosen.Version, ExitOK, true
	}
}

func chooseVersion(name string, mgr manager.Manager, st *styler, r *bufio.Reader, stdout, stderr io.Writer) (string, int, bool) {
	lister, ok := mgr.(manager.VersionLister)
	if !ok {
		fmt.Fprintf(stderr, "error: %s can't list versions; pass an explicit one: %s@VERSION\n", mgr.Name(), name)
		return "", ExitErr, false
	}
	versions, err := lister.Versions(name)
	if err != nil || len(versions) == 0 {
		fmt.Fprintf(stderr, "error: no other versions of %q available\n", name)
		return "", ExitNegative, false
	}
	sort.Slice(versions, func(i, j int) bool { return vercmp.Compare(versions[i], versions[j]) > 0 })
	if !canPrompt(r) {
		fmt.Fprintf(stderr, "error: specify a version: %s@VERSION\n", name)
		fmt.Fprintf(stderr, "available: %s\n", strings.Join(versions, ", "))
		return "", ExitErr, false
	}
	fmt.Fprintf(stdout, "%s versions:\n", st.accent(name))
	for i, v := range versions {
		fmt.Fprintf(stdout, "  %s %s\n", st.num(strconv.Itoa(i+1)+")"), st.version(v))
	}
	input, _ := readSelection(st.accent("==> ")+"version [1-"+strconv.Itoa(len(versions))+"]: ", r, stdout)
	idxs, perr := parseSelection(input, len(versions))
	if perr != nil || len(idxs) == 0 {
		fmt.Fprintln(stderr, "cancelled")
		return "", ExitOK, false
	}
	return versions[idxs[0]], ExitOK, true
}

func downgradeCmdFor(mgr manager.Manager, name, version string) *exec.Cmd {
	if d, ok := mgr.(manager.Downgrader); ok {
		if cmd := d.DowngradeCmd(name, version); cmd != nil {
			return cmd
		}
	}
	if v, ok := mgr.(manager.VersionedInstaller); ok {
		return v.InstallVersionCmd(name, version)
	}
	return nil
}
