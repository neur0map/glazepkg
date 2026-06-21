package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func init() {
	subcommands["info"] = runInfo
}

func runInfo(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag     = fs.String("manager", "", "comma list of managers (default: all)")
		jsonFlag    = fs.Bool("json", false, "emit JSON envelope")
		noCacheFlag = fs.Bool("no-cache", false, "bypass the scan cache")
	)
	fs.StringVar(mgrFlag, "m", *mgrFlag, "alias for --manager")
	args = prepManagerArgs(args, mgrs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}

	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "error: info takes exactly one package name")
		return ExitErr
	}
	name := rest[0]

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	cacheOK := cacheWriteOKFor(*mgrFlag)
	pkgs, err := collectPackages(filtered, *noCacheFlag, true, stderr, cacheOK)
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	var found *model.Package
	for _, m := range filtered {
		for i := range pkgs {
			if pkgs[i].Name == name && pkgs[i].Source == m.Name() {
				found = &pkgs[i]
				break
			}
		}
		if found != nil {
			break
		}
	}

	st := newStyler()

	if found != nil {
		if *jsonFlag {
			if err := writeEnvelope(stdout, version, toCLIPackage(*found)); err != nil {
				fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
				return ExitErr
			}
			return ExitOK
		}
		writeInfoHuman(stdout, *found, true, st)
		return ExitOK
	}

	// Not installed: fall back to what's available across managers, so
	// `gpk info <pkg>` / `-Si <pkg>` answers "where can I get it, which
	// version, what is it" the way pacman -Si does.
	cands := findInstallCandidates(name, filtered)
	if len(cands) == 0 {
		if sug := suggestPackages(name, filtered); len(sug) > 0 {
			fmt.Fprintf(stderr, "%q not found. did you mean: %s\n", name, strings.Join(sug, ", "))
		}
		return ExitNegative
	}

	if *jsonFlag {
		if err := writeEnvelope(stdout, version, toCLIPackage(cands[0].pkg)); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}
	for _, c := range cands {
		writeInfoHuman(stdout, c.pkg, false, st)
	}
	return ExitOK
}

func writeInfoHuman(w io.Writer, p model.Package, installed bool, st *styler) {
	var lines []string
	field := func(k, v string) {
		if v == "" {
			return
		}
		lines = append(lines, st.dim(padRight(k, 13))+v)
	}
	field("Version", st.version(p.Version))
	field("Source", st.mgrName(p.Source))
	if installed {
		field("Status", st.ok("installed"))
	} else {
		field("Status", st.dim("available"))
	}
	field("Description", p.Description)
	field("Size", p.Size)
	field("Repository", p.Repository)
	if p.LatestVersion != "" && p.LatestVersion != p.Version {
		field("Latest", st.warn(p.LatestVersion))
	}
	if len(p.DependsOn) > 0 {
		field("Depends on", strings.Join(p.DependsOn, ", "))
	}
	if len(p.RequiredBy) > 0 {
		field("Required by", strings.Join(p.RequiredBy, ", "))
	}
	fmt.Fprintln(w, st.box(p.Name, lines))
}
