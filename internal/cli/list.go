package cli

import (
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

func runList(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag      = fs.String("manager", "", "comma list of managers (e.g. pacman,aur or !brew); 'all' for all")
		jsonFlag     = fs.Bool("json", false, "emit JSON envelope")
		noCacheFlag  = fs.Bool("no-cache", false, "bypass the scan cache; do a fresh live scan")
		quietFlag    = fs.Bool("quiet", false, "suppress progress messages on stderr")
		updatesOnly  = fs.Bool("updates-only", false, "only packages whose latest_version differs from version")
	)
	fs.BoolVar(quietFlag, "q", *quietFlag, "alias for --quiet")
	fs.StringVar(mgrFlag, "m", *mgrFlag, "alias for --manager")
	if err := fs.Parse(args); err != nil {
		return ExitErr
	}

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	pkgs, err := collectPackages(filtered, *noCacheFlag, *quietFlag, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	if *updatesOnly {
		pkgs = withUpdates(filtered, pkgs)
	}

	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })

	if *jsonFlag {
		if err := writeEnvelope(stdout, version, toCLIPackages(pkgs)); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	writeListHuman(stdout, pkgs)
	return ExitOK
}

// collectPackages either loads from scan cache or runs a fresh live scan
// across the filtered manager set, writing back to cache afterward.
func collectPackages(mgrs []manager.Manager, noCache, quiet bool, stderr io.Writer) ([]model.Package, error) {
	if !noCache {
		if cached := manager.LoadScanCache(); cached != nil {
			// Apply the same manager filter to the cached data so --manager
			// is honored even on cache hits.
			return filterByManagers(cached, mgrs), nil
		}
	}

	var out []model.Package
	for _, m := range mgrs {
		if !m.Available() {
			continue
		}
		if !quiet {
			fmt.Fprintf(stderr, "scanning %s...\n", m.Name())
		}
		pkgs, err := m.Scan()
		if err != nil {
			if !quiet {
				fmt.Fprintf(stderr, "warning: %s scan failed: %v\n", m.Name(), err)
			}
			continue
		}
		out = append(out, pkgs...)
	}
	if !noCache || len(out) > 0 {
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

// withUpdates returns only packages whose LatestVersion is set and differs
// from Version. Lazily fetches updates for managers missing from the cache.
func withUpdates(mgrs []manager.Manager, pkgs []model.Package) []model.Package {
	cache := manager.NewUpdateCache()
	updates := manager.FetchUpdates(mgrs, pkgs, cache)
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
func writeListHuman(w io.Writer, pkgs []model.Package) {
	if len(pkgs) == 0 {
		fmt.Fprintln(w, "(no packages)")
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
	fmt.Fprintf(w, "%-*s  %-*s  %-*s\n", nameW, "NAME", verW, "VERSION", srcW, "SOURCE")
	fmt.Fprintln(w, strings.Repeat("-", nameW+verW+srcW+4))
	for _, p := range pkgs {
		fmt.Fprintf(w, "%-*s  %-*s  %-*s\n", nameW, p.Name, verW, p.Version, srcW, string(p.Source))
	}
}
