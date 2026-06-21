package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
	"github.com/neur0map/glazepkg/internal/snapshot"
)

func init() {
	subcommands["install"] = runInstall
}

// installPlan is a fully resolved install: the manager to drive, the package
// name, an optional pinned version, and the version/description discovered
// during resolution (for the plan display).
type installPlan struct {
	mgr      manager.Manager
	name     string
	version  string
	availVer string
	desc     string
}

type candidate struct {
	mgr manager.Manager
	pkg model.Package
}

func runInstall(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag    = fs.String("manager", "", "manager to install from; required if a name is available in multiple managers")
		yesFlag    = fs.Bool("yes", false, "skip the confirmation prompt")
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

	st := newStyler()
	r := newPromptReader(stdin)

	var plans []installPlan
	for _, raw := range names {
		name, ver := splitVersionPin(raw)
		plan, code, ok := resolvePlan(name, ver, filtered, st, r, stdout, stderr)
		if !ok {
			return code
		}
		plans = append(plans, plan)
	}

	return executeInstalls(plans, *yesFlag, *dryRunFlag, *quietFlag, st, r, stdout, stderr)
}

// resolvePlan turns one requested name into a concrete plan. When a name lives
// in several managers it asks (interactive) or returns exit-3 (scripted); when
// it lives nowhere it offers "did you mean" suggestions.
func resolvePlan(name, ver string, filtered []manager.Manager, st *styler, r *bufio.Reader, stdout, stderr io.Writer) (installPlan, int, bool) {
	cands := findInstallCandidates(name, filtered)
	switch len(cands) {
	case 0:
		fmt.Fprintf(stderr, "error: %q not found in any manager\n", name)
		if sug := suggestPackages(name, filtered); len(sug) > 0 {
			fmt.Fprintf(stderr, "did you mean: %s\n", strings.Join(sug, ", "))
		}
		return installPlan{}, ExitNegative, false
	case 1:
		return buildPlan(cands[0], ver, stderr)
	default:
		if canPrompt(r) {
			chosen, ok := pickCandidate(name, cands, st, r, stdout)
			if !ok {
				fmt.Fprintln(stderr, "cancelled")
				return installPlan{}, ExitOK, false
			}
			return buildPlan(chosen, ver, stderr)
		}
		srcs := make([]string, len(cands))
		for i, c := range cands {
			srcs[i] = string(c.mgr.Name())
		}
		fmt.Fprintf(stderr, "error: %q is available in %d managers (%s); pick one with --manager\n",
			name, len(cands), strings.Join(srcs, ", "))
		return installPlan{}, ExitAmbiguous, false
	}
}

func buildPlan(c candidate, ver string, stderr io.Writer) (installPlan, int, bool) {
	if ver != "" {
		if _, ok := c.mgr.(manager.VersionedInstaller); !ok {
			fmt.Fprintf(stderr, "error: %s cannot install a specific version\n", c.mgr.Name())
			return installPlan{}, ExitErr, false
		}
	} else if _, ok := c.mgr.(manager.Installer); !ok {
		fmt.Fprintf(stderr, "error: %s does not support install\n", c.mgr.Name())
		return installPlan{}, ExitErr, false
	}
	return installPlan{mgr: c.mgr, name: c.pkg.Name, version: ver, availVer: c.pkg.Version, desc: c.pkg.Description}, ExitOK, true
}

// findInstallCandidates returns one candidate per source that has an exact
// name match, deduplicated and mapped to the canonical installing manager
// (so an AUR result from pacman's search installs via the AUR helper).
func findInstallCandidates(name string, filtered []manager.Manager) []candidate {
	seen := make(map[model.Source]bool)
	var out []candidate
	for _, m := range filtered {
		if !m.Available() {
			continue
		}
		s, ok := m.(manager.Searcher)
		if !ok {
			continue
		}
		results, err := s.Search(name)
		if err != nil {
			continue
		}
		for _, p := range results {
			if p.Name != name {
				continue
			}
			src := p.Source
			if src == "" {
				src = m.Name()
			}
			if seen[src] {
				continue
			}
			canonical := mgrByName(filtered, src)
			if canonical == nil {
				continue
			}
			seen[src] = true
			p.Source = src
			out = append(out, candidate{mgr: canonical, pkg: p})
		}
	}
	return out
}

// suggestPackages searches a prefix of name and ranks the returned names by
// edit distance, so a typo like "ffmpgg" surfaces "ffmpeg".
func suggestPackages(name string, filtered []manager.Manager) []string {
	query := name
	if len(query) > 4 {
		query = query[:4]
	}
	seen := make(map[string]bool)
	var names []string
	for _, m := range filtered {
		if !m.Available() {
			continue
		}
		s, ok := m.(manager.Searcher)
		if !ok {
			continue
		}
		res, err := s.Search(query)
		if err != nil {
			continue
		}
		for _, p := range res {
			if !seen[p.Name] {
				seen[p.Name] = true
				names = append(names, p.Name)
			}
		}
		if len(names) > 400 {
			break
		}
	}
	return suggestNames(name, names, 3, 5)
}

func pickCandidate(name string, cands []candidate, st *styler, r *bufio.Reader, stdout io.Writer) (candidate, bool) {
	fmt.Fprintf(stdout, "%s is available from %d sources:\n", st.accent(name), len(cands))
	for i, c := range cands {
		row := fmt.Sprintf("  %s %s %s", st.num(strconv.Itoa(i+1)+")"), st.mgrName(c.mgr.Name()), st.version(c.pkg.Version))
		if c.pkg.Description != "" {
			row += "  " + st.dim(truncate(c.pkg.Description, 60))
		}
		fmt.Fprintln(stdout, row)
	}
	input, ok := readSelection(st.accent("==> ")+"source to install from [1-"+strconv.Itoa(len(cands))+"]: ", r, stdout)
	if !ok {
		return candidate{}, false
	}
	idxs, err := parseSelection(input, len(cands))
	if err != nil || len(idxs) == 0 {
		return candidate{}, false
	}
	return cands[idxs[0]], true
}

// executeInstalls renders the plan, prompts (unless yes), and runs each
// install. Shared by `gpk install` and the search-and-install picker.
func executeInstalls(plans []installPlan, yes, dryRun, quiet bool, st *styler, r *bufio.Reader, stdout, stderr io.Writer) int {
	if len(plans) == 0 {
		return ExitOK
	}
	fmt.Fprintln(stdout, st.title("Install plan"))
	for _, p := range plans {
		cmd := installCmdFor(p, yes)
		if cmd == nil {
			fmt.Fprintf(stderr, "error: %s cannot install %s\n", p.mgr.Name(), p.name)
			return ExitErr
		}
		target := p.name
		switch {
		case p.version != "":
			target += " " + st.version("@"+p.version)
		case p.availVer != "":
			target += " " + st.version(p.availVer)
		}
		fmt.Fprintf(stdout, "  %s  %s\n", st.mgrName(p.mgr.Name()), target)
		fmt.Fprintln(stdout, "      "+st.dim(displayCmd(cmd)))
		if p.version == "" {
			if pv, ok := p.mgr.(manager.Previewer); ok {
				if deps, err := pv.InstallDeps(p.name); err == nil && len(deps) > 0 {
					fmt.Fprintln(stdout, "      "+st.dim(fmt.Sprintf("pulls in %d: %s", len(deps), truncate(strings.Join(deps, " "), 64))))
				}
			}
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

	grp := nextGroup()
	for _, p := range plans {
		if !quiet {
			fmt.Fprintf(stderr, "installing %s via %s...\n", p.name, p.mgr.Name())
		}
		cmd := installCmdFor(p, yes)
		if err := headlessExec(cmd); err != nil {
			fmt.Fprintf(stderr, "error: install %s failed: %v\n", p.name, err)
			return ExitErr
		}
		invalidateAfterWrite(p.mgr, []model.Package{{Name: p.name, Source: p.mgr.Name()}})
		ver := p.version
		if ver == "" {
			ver = p.availVer
		}
		_ = snapshot.AppendHistory(snapshot.HistoryItem{
			Group: grp, Time: time.Now(), Op: snapshot.OpInstall,
			Source: p.mgr.Name(), Name: p.name, Version: ver,
		})
	}
	return ExitOK
}

func installCmdFor(p installPlan, yes bool) *exec.Cmd {
	if p.version != "" {
		if v, ok := p.mgr.(manager.VersionedInstaller); ok {
			return v.InstallVersionCmd(p.name, p.version)
		}
		return nil
	}
	if yes {
		if ni, ok := p.mgr.(manager.NonInteractiveInstaller); ok {
			if cmd := ni.InstallCmdYes(p.name); cmd != nil {
				return cmd
			}
		}
	}
	if inst, ok := p.mgr.(manager.Installer); ok {
		return inst.InstallCmd(p.name)
	}
	return nil
}

func displayCmd(cmd *exec.Cmd) string {
	return strings.Join(stripSudoStdinFlag(cmd).Args, " ")
}

// splitVersionPin splits "name@version" into its parts. A leading '@' (scoped
// npm packages) is preserved; only a later '@' is treated as the pin.
func splitVersionPin(raw string) (name, version string) {
	if i := strings.LastIndex(raw, "@"); i > 0 {
		return raw[:i], raw[i+1:]
	}
	return raw, ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
