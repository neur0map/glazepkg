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
	"sync"
	"time"

	"github.com/neur0map/glazepkg/internal/config"
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
		jsonFlag   = fs.Bool("json", false, "emit the resolved plan as JSON and exit (no execution)")
		pickFlag   = fs.Bool("pick-version", false, "choose the version to install interactively")
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
	resolveR := r
	if *jsonFlag {
		resolveR = nil // JSON mode is non-interactive: ambiguity returns exit 3
	}
	prefer := config.Load().Install.Prefer

	var plans []installPlan
	for _, raw := range names {
		name, ver := splitVersionPin(raw)
		plan, code, ok := resolvePlan(name, ver, filtered, prefer, st, resolveR, stdout, stderr)
		if !ok {
			return code
		}
		if *pickFlag && plan.version == "" {
			if _, ok := plan.mgr.(manager.VersionedInstaller); !ok {
				if !*quietFlag {
					fmt.Fprintf(stderr, "%s can't install a chosen version; using latest\n", plan.mgr.Name())
				}
			} else if v, code, ok := chooseVersion(plan.name, plan.mgr, st, resolveR, stdout, stderr); ok {
				plan.version = v
			} else {
				return code
			}
		}
		plans = append(plans, plan)
	}

	if *jsonFlag {
		steps := make([]planStep, 0, len(plans))
		for _, p := range plans {
			ver := p.version
			if ver == "" {
				ver = p.availVer
			}
			steps = append(steps, planStep{Manager: string(p.mgr.Name()), Name: p.name, Version: ver, Command: cmdArgs(installCmdFor(p, *yesFlag))})
		}
		if err := emitPlanJSON(stdout, version, "install", steps); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	return executeInstalls(plans, *yesFlag, *dryRunFlag, *quietFlag, st, r, stdout, stderr)
}

// resolvePlan turns one requested name into a concrete plan. When a name lives
// in several managers it asks (interactive) or returns exit-3 (scripted); when
// it lives nowhere it offers "did you mean" suggestions.
func resolvePlan(name, ver string, filtered []manager.Manager, prefer []string, st *styler, r *bufio.Reader, stdout, stderr io.Writer) (installPlan, int, bool) {
	// An explicitly named manager that can't search is trusted: the user said
	// where to get it, so build the plan directly instead of failing to "find" it.
	if len(filtered) == 1 {
		m := filtered[0]
		if !m.Available() {
			fmt.Fprintf(stderr, "error: %s is not available on this system\n", m.Name())
			return installPlan{}, ExitErr, false
		}
		if _, ok := m.(manager.Searcher); !ok {
			return buildPlan(candidate{mgr: m, pkg: model.Package{Name: name, Source: m.Name()}}, ver, stderr)
		}
	}
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
		sort.SliceStable(cands, func(i, j int) bool {
			pi, pj := preferRank(cands[i].mgr.Name(), prefer), preferRank(cands[j].mgr.Name(), prefer)
			if pi != pj {
				return pi < pj
			}
			return preferRank(cands[i].mgr.Name(), defaultPreference) < preferRank(cands[j].mgr.Name(), defaultPreference)
		})
		if len(prefer) > 0 && preferRank(cands[0].mgr.Name(), prefer) < len(prefer) {
			fmt.Fprintf(stderr, "%s: using %s (preferred)\n", name, cands[0].mgr.Name())
			return buildPlan(cands[0], ver, stderr)
		}
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

func preferRank(src model.Source, prefer []string) int {
	for i, p := range prefer {
		if p == string(src) {
			return i
		}
	}
	return len(prefer)
}

// defaultPreference orders managers when a name lives in several and the user
// hasn't set their own preference: the OS/system managers resolve ahead of the
// language ecosystems, since a bare `gpk install ffmpeg` almost always means the
// system package, not a same-named library on PyPI or npm.
var defaultPreference = []string{
	"pacman", "aur", "apt", "dnf", "xbps", "apk", "portage", "guix", "nix",
	"brew", "brew-cask", "flatpak", "snap", "macports", "pkgsrc", "pkg", "mas",
	"scoop", "chocolatey", "winget", "am",
	"go", "cargo", "npm", "pnpm", "bun", "pip", "pipx", "uv", "gem", "composer",
	"conda", "luarocks", "opam", "nuget", "maven", "powershell", "mise", "gvm", "quicklisp",
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
	type res struct {
		from model.Source
		pkgs []model.Package
	}
	var wg sync.WaitGroup
	ch := make(chan res, len(filtered))
	for _, m := range filtered {
		s, ok := m.(manager.Searcher)
		if !ok || !m.Available() {
			continue
		}
		wg.Add(1)
		go func(s manager.Searcher, from model.Source) {
			defer wg.Done()
			pkgs, err := s.Search(name)
			if err != nil {
				pkgs = nil
			}
			ch <- res{from: from, pkgs: pkgs}
		}(s, m.Name())
	}
	go func() { wg.Wait(); close(ch) }()

	bySource := make(map[model.Source]candidate)
	native := make(map[model.Source]bool)
	for r := range ch {
		for _, p := range r.pkgs {
			if p.Name != name {
				continue
			}
			src := p.Source
			if src == "" {
				src = r.from
			}
			canonical := mgrByName(filtered, src)
			if canonical == nil {
				continue
			}
			// Prefer the result from the source's own manager (e.g. AUR's own
			// search over an aur row surfaced by pacman) so output is stable.
			if _, have := bySource[src]; have && !(r.from == src && !native[src]) {
				continue
			}
			p.Source = src
			bySource[src] = candidate{mgr: canonical, pkg: p}
			native[src] = r.from == src
		}
	}

	var out []candidate
	for _, m := range filtered {
		if c, ok := bySource[m.Name()]; ok {
			out = append(out, c)
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
	input, ok := readSelection(st.accent("==> ")+"source to install from [1-"+strconv.Itoa(len(cands))+"] (default 1): ", r, stdout)
	if !ok {
		return candidate{}, false
	}
	if strings.TrimSpace(input) == "" {
		return cands[0], true
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
			if pvr, ok := p.mgr.(manager.Previewer); ok {
				if pv, err := pvr.PreviewInstall(p.name); err == nil {
					if len(pv.Deps) > 0 {
						fmt.Fprintln(stdout, "      "+st.dim(fmt.Sprintf("pulls in %d: %s", len(pv.Deps), truncate(strings.Join(pv.Deps, " "), 64))))
					}
					if pv.Download > 0 || pv.Installed > 0 {
						fmt.Fprintln(stdout, "      "+st.dim(fmt.Sprintf("download %s · installed %s", manager.FormatBytes(pv.Download), manager.FormatBytes(pv.Installed))))
					}
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
