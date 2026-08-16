package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func init() {
	subcommands["list"] = runList
}

func runList(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag     = fs.String("manager", "", "comma list of managers (e.g. pacman,aur or !brew); 'all' for all")
		jsonFlag    = fs.Bool("json", false, "emit JSON envelope")
		noCacheFlag = fs.Bool("no-cache", false, "bypass the scan cache; do a fresh live scan")
		quietFlag   = fs.Bool("quiet", false, "suppress progress messages on stderr")
		updatesOnly = fs.Bool("updates-only", false, "only packages whose latest_version differs from version")
	)
	fs.BoolVar(quietFlag, "q", *quietFlag, "alias for --quiet")
	fs.StringVar(mgrFlag, "m", *mgrFlag, "alias for --manager")
	args = prepManagerArgs(args, mgrs)
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

	cacheOK := cacheWriteOKFor(*mgrFlag)
	pkgs, err := collectPackages(filtered, *noCacheFlag, *quietFlag, stderr, cacheOK)
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	if *updatesOnly {
		pkgs = withUpdates(filtered, pkgs, *quietFlag, stderr)
	}

	if terms := fs.Args(); len(terms) > 0 {
		pkgs = filterByTerms(pkgs, terms)
	}

	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })

	if *jsonFlag {
		if err := writeEnvelope(stdout, version, toCLIPackages(pkgs)); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	writeListHuman(stdout, pkgs, newStyler())
	return ExitOK
}

// collectPackages either loads from scan cache or runs a fresh live scan
// across the filtered manager set, writing back to cache afterward. A live scan
// captures its subprocess output, so the progress line is gpk's to animate.
func collectPackages(mgrs []manager.Manager, noCache, quiet bool, stderr io.Writer, cacheOK bool) ([]model.Package, error) {
	if !noCache {
		if cached := manager.LoadScanCache(); cached != nil {
			// Apply the same manager filter to the cached data so --manager
			// is honored even on cache hits.
			return filterByManagers(cached, mgrs), nil
		}
	}

	var sp *spinner
	if !quiet {
		sp = startSpinner(stderr, newStyler(), "")
		defer sp.stop()
	}

	var out []model.Package
	for _, m := range mgrs {
		if !m.Available() {
			continue
		}
		sp.update("scanning " + string(m.Name()))
		pkgs, err := m.Scan()
		if err != nil {
			sp.note(fmt.Sprintf("warning: %s scan failed: %v", m.Name(), err))
			continue
		}
		out = append(out, pkgs...)
	}
	if cacheOK && (!noCache || len(out) > 0) {
		manager.SaveScanCache(out)
	}
	return out, nil
}

func filterByManagers(pkgs []model.Package, mgrs []manager.Manager) []model.Package {
	allow := make(map[model.Source]bool, len(mgrs))
	for _, m := range mgrs {
		allow[m.Name()] = true
	}
	out := pkgs[:0:0]
	for _, p := range pkgs {
		if allow[p.Source] {
			out = append(out, p)
		}
	}
	return out
}

// filterByTerms keeps packages whose name or description contains any of the
// given substrings (case-insensitive). Powers `gpk list <term>` and `-Qs`.
func filterByTerms(pkgs []model.Package, terms []string) []model.Package {
	lowered := make([]string, len(terms))
	for i, t := range terms {
		lowered[i] = strings.ToLower(t)
	}
	out := pkgs[:0:0]
	for _, p := range pkgs {
		name, desc := strings.ToLower(p.Name), strings.ToLower(p.Description)
		for _, t := range lowered {
			if strings.Contains(name, t) || strings.Contains(desc, t) {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// fetchUpdates asks each manager for the latest available versions, showing a
// progress line while it works. Update checks reach the network (repo
// databases, the AUR RPC, PyPI) so this is the second place a gpk command can
// sit for a long time with nothing of its own to print.
func fetchUpdates(mgrs []manager.Manager, pkgs []model.Package, cache *manager.UpdateCache, quiet bool, stderr io.Writer) map[string]string {
	if !quiet {
		sp := startSpinner(stderr, newStyler(), "checking for updates")
		defer sp.stop()
	}
	return manager.FetchUpdates(mgrs, pkgs, cache)
}

// withUpdates returns only packages whose LatestVersion is set and differs
// from Version. Lazily fetches updates for managers missing from the cache.
func withUpdates(mgrs []manager.Manager, pkgs []model.Package, quiet bool, stderr io.Writer) []model.Package {
	cache := manager.NewUpdateCache()
	updates := fetchUpdates(mgrs, pkgs, cache, quiet, stderr)
	out := make([]model.Package, 0, len(pkgs))
	for _, p := range pkgs {
		if latest, ok := updates[p.Key()]; ok && latest != "" && latest != p.Version {
			p.LatestVersion = latest
			out = append(out, p)
		}
	}
	return out
}

// writeListHuman prints a plain text table: NAME VERSION SOURCE.
// No ANSI codes; the cli emits plain text whenever stdout isn't a TTY,
// and tests always run with a bytes.Buffer writer (non-TTY).
func writeListHuman(w io.Writer, pkgs []model.Package, st *styler) {
	if len(pkgs) == 0 {
		fmt.Fprintln(w, st.dim("(no packages)"))
		return
	}
	nameW, verW, srcW := 4, 7, 6 // header widths
	for _, p := range pkgs {
		if len(p.Name) > nameW {
			nameW = len(p.Name)
		}
		if len(p.Version) > verW {
			verW = len(p.Version)
		}
		if l := len(string(p.Source)); l > srcW {
			srcW = l
		}
	}
	fmt.Fprintln(w, st.dim(fmt.Sprintf("%-*s  %-*s  %-*s", nameW, "NAME", verW, "VERSION", srcW, "SOURCE")))
	fmt.Fprintln(w, st.dim(strings.Repeat("-", nameW+verW+srcW+4)))
	for _, p := range pkgs {
		name := st.paint(padRight(p.Name, nameW), st.pal.White, false)
		ver := st.version(padRight(p.Version, verW))
		src := st.paint(padRight(string(p.Source), srcW), st.mgrColorOf(p.Source), true)
		fmt.Fprintf(w, "%s  %s  %s\n", name, ver, src)
	}
}
