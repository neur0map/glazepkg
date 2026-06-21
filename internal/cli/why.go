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
	subcommands["why"] = runWhy
}

// depName strips a version constraint from a dependency token so a dep recorded
// as "glibc>=2.40" or "black==24.1" matches the bare package name.
func depName(s string) string {
	if i := strings.IndexAny(s, "<>=~!"); i >= 0 {
		return s[:i]
	}
	return s
}

// runWhy reports which installed packages depend on the named one — the reverse
// of the dependency list `gpk info` shows, answering "what needs this, is it
// safe to remove" the way `brew uses` and `pacman -Qi` (Required By) do.
// Reverse deps are derived by inverting every installed package's dependency
// list, so it works for any manager that reports dependencies.
func runWhy(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("why", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag     = fs.String("manager", "", "comma list of managers (default: all)")
		jsonFlag    = fs.Bool("json", false, "emit a JSON envelope")
		noCacheFlag = fs.Bool("no-cache", false, "bypass the scan cache")
		quietFlag   = fs.Bool("quiet", false, "suppress progress on stderr")
	)
	fs.StringVar(mgrFlag, "m", "", "alias for --manager")
	fs.BoolVar(quietFlag, "q", false, "alias for --quiet")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "error: why takes exactly one package name")
		return ExitErr
	}
	target := rest[0]

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}
	pkgs, err := collectPackages(filtered, *noCacheFlag, *quietFlag, stderr, cacheWriteOKFor(*mgrFlag))
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	explicit := false
	for _, p := range pkgs {
		if p.Name == target {
			explicit = true
			break
		}
	}

	deps := manager.FetchDependencies(filtered, pkgs, manager.NewDepsCache())
	byKey := make(map[string]model.Package, len(pkgs))
	for _, p := range pkgs {
		byKey[p.Key()] = p
	}
	var dependents []model.Package
	for key, ds := range deps {
		for _, d := range ds {
			if depName(d) == target {
				if p, ok := byKey[key]; ok {
					dependents = append(dependents, p)
				}
				break
			}
		}
	}
	sort.Slice(dependents, func(i, j int) bool { return dependents[i].Name < dependents[j].Name })
	// A package is present if it's explicitly installed or something depends on
	// it (a satisfied dependency is, by definition, installed).
	present := explicit || len(dependents) > 0

	if *jsonFlag {
		names := make([]string, len(dependents))
		for i, p := range dependents {
			names[i] = p.Name
		}
		data := struct {
			Name       string   `json:"name"`
			Installed  bool     `json:"installed"`
			RequiredBy []string `json:"required_by"`
		}{target, present, names}
		if err := writeEnvelope(stdout, version, data); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		if !present {
			return ExitNegative
		}
		return ExitOK
	}

	st := newStyler()
	if !present {
		fmt.Fprintf(stderr, "error: %q is not installed\n", target)
		return ExitNegative
	}
	if len(dependents) == 0 {
		fmt.Fprintln(stdout, st.dim("nothing requires ")+st.accent(target)+st.dim(" — safe to remove"))
		return ExitOK
	}
	fmt.Fprintln(stdout, st.title("Required by ")+st.accent(target))
	for _, p := range dependents {
		fmt.Fprintf(stdout, "  %s %s %s\n", st.mgrName(p.Source), st.paint(p.Name, st.pal.White, false), st.dim(p.Version))
	}
	return ExitOK
}
