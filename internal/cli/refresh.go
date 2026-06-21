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
	subcommands["refresh"] = runRefresh
}

// runRefresh rebuilds the scan cache from a fresh live scan — gpk's analogue of
// `pacman -Sy`, which syncs the databases the other commands read from.
func runRefresh(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("refresh", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag   = fs.String("manager", "", "comma list of managers (default: all)")
		jsonFlag  = fs.Bool("json", false, "emit JSON envelope")
		quietFlag = fs.Bool("quiet", false, "suppress progress on stderr")
	)
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

	pkgs, err := collectPackages(filtered, true, *quietFlag, stderr, cacheWriteOKFor(*mgrFlag))
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	seen := make(map[model.Source]bool)
	for _, p := range pkgs {
		seen[p.Source] = true
	}

	if *jsonFlag {
		data := struct {
			Packages int `json:"packages"`
			Managers int `json:"managers"`
		}{Packages: len(pkgs), Managers: len(seen)}
		if err := writeEnvelope(stdout, version, data); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	st := newStyler()
	fmt.Fprintf(stdout, "%s refreshed %s across %s\n",
		st.ok("✓"),
		st.version(fmt.Sprintf("%d packages", len(pkgs))),
		st.accent(fmt.Sprintf("%d managers", len(seen))))
	return ExitOK
}
