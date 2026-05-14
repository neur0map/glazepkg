package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func init() {
	subcommands["installed"] = runInstalled
}

func runInstalled(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("installed", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag     = fs.String("manager", "", "comma list of managers (default: all)")
		jsonFlag    = fs.Bool("json", false, "emit JSON envelope")
		noCacheFlag = fs.Bool("no-cache", false, "bypass the scan cache")
		quietFlag   = fs.Bool("quiet", false, "suppress missing-name report on stderr")
	)
	fs.BoolVar(quietFlag, "q", *quietFlag, "alias for --quiet")
	fs.StringVar(mgrFlag, "m", *mgrFlag, "alias for --manager")
	args = reorderFlagsFirst(args, []string{"manager", "m"})
	if err := fs.Parse(args); err != nil {
		return ExitErr
	}

	names := fs.Args()
	if len(names) == 0 {
		fmt.Fprintln(stderr, "error: installed requires at least one package name")
		return ExitErr
	}

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	pkgs, err := collectPackages(filtered, *noCacheFlag, true, stderr) // installed always quiet about scans
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	byName := make(map[string][]model.Package)
	for _, p := range pkgs {
		byName[p.Name] = append(byName[p.Name], p)
	}

	type result struct {
		Name      string       `json:"name"`
		Installed bool         `json:"installed"`
		Matches   []cliPackage `json:"matches"`
	}
	results := make([]result, 0, len(names))
	var missing []string
	for _, n := range names {
		matches := byName[n]
		r := result{Name: n, Installed: len(matches) > 0, Matches: toCLIPackages(matches)}
		results = append(results, r)
		if !r.Installed {
			missing = append(missing, n)
		}
	}

	if *jsonFlag {
		if err := writeEnvelope(stdout, version, results); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
	}

	if len(missing) > 0 {
		if !*quietFlag {
			fmt.Fprintf(stderr, "not installed: %s\n", strings.Join(missing, ", "))
		}
		return ExitNegative
	}
	return ExitOK
}
