package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func init() {
	subcommands["info"] = runInfo
}

func runInfo(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag     = fs.String("manager", "", "comma list of managers (default: all)")
		jsonFlag    = fs.Bool("json", false, "emit JSON envelope")
		noCacheFlag = fs.Bool("no-cache", false, "bypass the scan cache")
	)
	fs.StringVar(mgrFlag, "m", *mgrFlag, "alias for --manager")
	args = reorderFlagsFirst(args, []string{"manager", "m"})
	if err := fs.Parse(args); err != nil {
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

	pkgs, err := collectPackages(filtered, *noCacheFlag, true, stderr)
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

	if found == nil {
		return ExitNegative
	}

	if *jsonFlag {
		if err := writeEnvelope(stdout, version, toCLIPackage(*found)); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	writeInfoHuman(stdout, *found)
	return ExitOK
}

func writeInfoHuman(w io.Writer, p model.Package) {
	field := func(k, v string) {
		if v == "" {
			return
		}
		fmt.Fprintf(w, "%-14s %s\n", k+":", v)
	}
	field("Name", p.Name)
	field("Version", p.Version)
	field("Source", string(p.Source))
	field("Description", p.Description)
	field("Size", p.Size)
	field("Repository", p.Repository)
	if p.LatestVersion != "" && p.LatestVersion != p.Version {
		field("Latest", p.LatestVersion)
	}
	if len(p.DependsOn) > 0 {
		fmt.Fprintln(w, "Depends on:")
		for _, d := range p.DependsOn {
			fmt.Fprintf(w, "  %s\n", d)
		}
	}
	if len(p.RequiredBy) > 0 {
		fmt.Fprintln(w, "Required by:")
		for _, d := range p.RequiredBy {
			fmt.Fprintf(w, "  %s\n", d)
		}
	}
}
