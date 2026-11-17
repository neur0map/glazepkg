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
	subcommands["apps"] = runApps
}

// loadApps is the app inventory source; tests replace it with fixtures.
var loadApps = manager.LoadApps

// runApps lists launchable desktop apps (the entries an app launcher shows)
// joined to the package that installed each, with its installed size.
func runApps(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("apps", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag     = fs.String("manager", "", "only apps owned by these managers (e.g. pacman,aur,flatpak,local)")
		jsonFlag    = fs.Bool("json", false, "emit JSON envelope")
		noCacheFlag = fs.Bool("no-cache", false, "bypass the app cache; do a fresh scan")
		quietFlag   = fs.Bool("quiet", false, "accepted for symmetry with list; apps prints no progress")
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
	allow := make(map[model.Source]bool, len(filtered))
	for _, m := range filtered {
		allow[m.Name()] = true
	}

	all := loadApps(*noCacheFlag)
	apps := make([]manager.DesktopApp, 0, len(all))
	for _, a := range all {
		if allow[a.Source] {
			apps = append(apps, a)
		}
	}
	if terms := fs.Args(); len(terms) > 0 {
		apps = filterAppsByTerms(apps, terms)
	}

	if *jsonFlag {
		if err := writeEnvelope(stdout, version, apps); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}
	writeAppsHuman(stdout, apps, newStyler())
	return ExitOK
}

func filterAppsByTerms(apps []manager.DesktopApp, terms []string) []manager.DesktopApp {
	out := apps[:0:0]
	for _, a := range apps {
		hay := strings.ToLower(a.Name + " " + a.DesktopID + " " + a.Package + " " + a.GenericName)
		for _, t := range terms {
			if strings.Contains(hay, strings.ToLower(t)) {
				out = append(out, a)
				break
			}
		}
	}
	return out
}

// writeAppsHuman prints NAME PACKAGE SOURCE SIZE, one row per entry.
func writeAppsHuman(w io.Writer, apps []manager.DesktopApp, st *styler) {
	if len(apps) == 0 {
		fmt.Fprintln(w, st.dim("(no apps)"))
		return
	}
	nameW, pkgW, srcW := 4, 7, 6
	for _, a := range apps {
		nameW = max(nameW, len(a.Name))
		pkgW = max(pkgW, len(a.Package))
		srcW = max(srcW, len(string(a.Source)))
	}
	nameW, pkgW = min(nameW, 40), min(pkgW, 40)
	fmt.Fprintln(w, st.dim(fmt.Sprintf("%-*s  %-*s  %-*s  %s", nameW, "NAME", pkgW, "PACKAGE", srcW, "SOURCE", "SIZE")))
	fmt.Fprintln(w, st.dim(strings.Repeat("-", nameW+pkgW+srcW+16)))
	for _, a := range apps {
		pkg := a.Package
		if pkg == "" {
			pkg = "-"
		}
		size := a.Size
		if size == "" {
			size = "-"
		}
		fmt.Fprintf(w, "%s  %s  %s  %s\n",
			st.paint(padRight(truncate(a.Name, nameW), nameW), st.pal.White, false),
			padRight(truncate(pkg, pkgW), pkgW),
			st.paint(padRight(string(a.Source), srcW), st.mgrColorOf(a.Source), true),
			st.dim(size))
	}
}
