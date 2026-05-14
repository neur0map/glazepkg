package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func init() {
	subcommands["source-of"] = runSourceOf
}

func runSourceOf(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("source-of", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag     = fs.String("manager", "", "comma list of managers (default: all)")
		allFlag     = fs.Bool("all", false, "list every source that has the package")
		jsonFlag    = fs.Bool("json", false, "emit JSON envelope")
		noCacheFlag = fs.Bool("no-cache", false, "bypass the scan cache")
	)
	fs.StringVar(mgrFlag, "m", "", "alias for --manager")
	args = reorderFlagsFirst(args, []string{"manager", "m"})
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}

	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "error: source-of takes exactly one package name")
		return ExitErr
	}
	name := rest[0]

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	cacheOK := *mgrFlag == "" || *mgrFlag == "all"
	pkgs, err := collectPackages(filtered, *noCacheFlag, true, stderr, cacheOK)
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	var sources []string
	seen := make(map[model.Source]bool)
	// Iterate in filtered (manager.All()) order so output is stable.
	for _, m := range filtered {
		for _, p := range pkgs {
			if p.Name == name && p.Source == m.Name() && !seen[p.Source] {
				seen[p.Source] = true
				sources = append(sources, string(p.Source))
				break
			}
		}
	}

	if len(sources) == 0 {
		return ExitNegative
	}
	if !*allFlag {
		sources = sources[:1]
	}

	if *jsonFlag {
		data := struct {
			Name    string   `json:"name"`
			Sources []string `json:"sources"`
		}{Name: name, Sources: sources}
		if err := writeEnvelope(stdout, version, data); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}
	for _, s := range sources {
		fmt.Fprintln(stdout, s)
	}
	return ExitOK
}
