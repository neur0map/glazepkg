package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func init() {
	subcommands["export"] = runExport
	subcommands["import"] = runImport
}

// runExport writes the installed package set as a JSON envelope (the same shape
// as `gpk list --json`) for backup or moving to another machine. `gpk import`
// reads it back.
func runExport(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs, "out", "o")
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag     = fs.String("manager", "", "comma list of managers (default: all)")
		outFlag     = fs.String("out", "", "write to a file instead of stdout")
		noCacheFlag = fs.Bool("no-cache", false, "bypass the scan cache")
		quietFlag   = fs.Bool("quiet", false, "suppress progress on stderr")
	)
	fs.StringVar(outFlag, "o", "", "alias for --out")
	fs.StringVar(mgrFlag, "m", "", "alias for --manager")
	fs.BoolVar(quietFlag, "q", false, "alias for --quiet")
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
	pkgs, err := collectPackages(filtered, *noCacheFlag, *quietFlag, stderr, cacheWriteOKFor(*mgrFlag))
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })

	w := stdout
	if *outFlag != "" {
		f, err := os.Create(*outFlag)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitErr
		}
		defer f.Close()
		w = f
	}
	if err := writeEnvelope(w, version, toCLIPackages(pkgs)); err != nil {
		fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
		return ExitErr
	}
	if *outFlag != "" && !*quietFlag {
		fmt.Fprintf(stderr, "exported %d packages to %s\n", len(pkgs), *outFlag)
	}
	return ExitOK
}

type importPkg struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

// runImport installs everything listed in a file produced by `gpk export` (or a
// plain `source/name` list), skipping what's already present — the restore half
// of the migration story.
func runImport(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag    = fs.String("manager", "", "only import these managers")
		yesFlag    = fs.Bool("yes", false, "skip the confirmation prompt")
		dryRunFlag = fs.Bool("dry-run", false, "print the plan without installing")
		quietFlag  = fs.Bool("quiet", false, "suppress progress on stderr")
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
		fmt.Fprintln(stderr, "error: import takes exactly one file")
		return ExitErr
	}
	data, err := os.ReadFile(rest[0])
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}
	entries := parseImport(data)
	if len(entries) == 0 {
		fmt.Fprintln(stderr, "error: no packages found in file")
		return ExitErr
	}

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	installed := make(map[string]bool)
	cur, _ := collectPackages(filtered, false, true, stderr, false)
	for _, p := range cur {
		installed[string(p.Source)+":"+p.Name] = true
	}

	st := newStyler()
	var plans []installPlan
	skipped := 0
	for _, e := range entries {
		mgr := resolveImport(e, filtered)
		if mgr == nil {
			continue
		}
		if installed[string(mgr.Name())+":"+e.Name] {
			skipped++
			continue
		}
		if _, ok := mgr.(manager.Installer); !ok {
			continue
		}
		plans = append(plans, installPlan{mgr: mgr, name: e.Name})
	}

	if len(plans) == 0 {
		fmt.Fprintf(stderr, "nothing to import (%d already installed)\n", skipped)
		return ExitOK
	}
	if skipped > 0 && !*quietFlag {
		fmt.Fprintf(stderr, "%s\n", st.dim(fmt.Sprintf("skipping %d already-installed package(s)", skipped)))
	}

	r := newPromptReader(stdin)
	return executeInstalls(plans, *yesFlag, *dryRunFlag, *quietFlag, st, r, stdout, stderr)
}

// resolveImport maps an entry to an available manager: by its source when
// given, otherwise by a unique search match across the filtered set.
func resolveImport(e importPkg, filtered []manager.Manager) manager.Manager {
	if e.Source != "" {
		mgr := mgrByName(filtered, model.Source(e.Source))
		if mgr != nil && mgr.Available() {
			return mgr
		}
		return nil
	}
	cands := findInstallCandidates(e.Name, filtered)
	if len(cands) == 1 {
		return cands[0].mgr
	}
	return nil
}

// parseImport reads a gpk JSON export (enveloped or a bare array) or a plain
// text list of `source/name`, `source:name`, or bare `name` lines.
func parseImport(data []byte) []importPkg {
	var env struct {
		Data []importPkg `json:"data"`
	}
	if json.Unmarshal(data, &env) == nil && len(env.Data) > 0 {
		return env.Data
	}
	var arr []importPkg
	if json.Unmarshal(data, &arr) == nil && len(arr) > 0 {
		return arr
	}
	var out []importPkg
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexAny(line, "/:"); i > 0 {
			out = append(out, importPkg{Source: line[:i], Name: strings.TrimSpace(line[i+1:])})
		} else {
			out = append(out, importPkg{Name: line})
		}
	}
	return out
}
