package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
	"github.com/neur0map/glazepkg/internal/snapshot"
)

func init() {
	subcommands["hold"] = runHold
	subcommands["unhold"] = runUnhold
	subcommands["holds"] = runHoldsList
}

func runHold(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("hold", flag.ContinueOnError)
	fs.SetOutput(stderr)
	mgrFlag := fs.String("manager", "", "limit to a manager")
	noCacheFlag := fs.Bool("no-cache", false, "bypass the scan cache")
	fs.StringVar(mgrFlag, "m", "", "alias for --manager")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}

	st := newStyler()
	names := fs.Args()
	if len(names) == 0 {
		printHolds(stdout, snapshot.LoadHolds(), st)
		return ExitOK
	}

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}
	pkgs, err := collectPackages(filtered, *noCacheFlag, true, stderr, cacheWriteOKFor(*mgrFlag))
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	holds := snapshot.LoadHolds()
	added := 0
	for _, name := range names {
		var srcs []model.Source
		for _, p := range pkgs {
			if p.Name == name {
				srcs = append(srcs, p.Source)
			}
		}
		if len(srcs) == 0 {
			fmt.Fprintf(stderr, "%q is not installed; not held\n", name)
			continue
		}
		for _, s := range srcs {
			if !snapshot.IsHeld(holds, s, name) {
				holds = append(holds, snapshot.Hold{Source: s, Name: name})
				added++
				fmt.Fprintf(stdout, "%s held %s %s\n", st.ok("✓"), name, st.mgrName(s))
			}
		}
	}
	if added > 0 {
		_ = snapshot.SaveHolds(holds)
	}
	return ExitOK
}

func runUnhold(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("unhold", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}
	names := fs.Args()
	if len(names) == 0 {
		fmt.Fprintln(stderr, "error: unhold requires at least one package name")
		return ExitErr
	}
	wanted := make(map[string]bool, len(names))
	for _, n := range names {
		wanted[n] = true
	}

	st := newStyler()
	holds := snapshot.LoadHolds()
	var kept []snapshot.Hold
	removed := 0
	for _, h := range holds {
		if wanted[h.Name] {
			removed++
			fmt.Fprintf(stdout, "%s released %s %s\n", st.ok("✓"), h.Name, st.mgrName(h.Source))
			continue
		}
		kept = append(kept, h)
	}
	if removed == 0 {
		fmt.Fprintln(stderr, "no matching holds")
		return ExitNegative
	}
	_ = snapshot.SaveHolds(kept)
	return ExitOK
}

func runHoldsList(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("holds", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonFlag := fs.Bool("json", false, "emit JSON envelope")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}
	holds := snapshot.LoadHolds()
	if *jsonFlag {
		if holds == nil {
			holds = []snapshot.Hold{}
		}
		if err := writeEnvelope(stdout, version, holds); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}
	printHolds(stdout, holds, newStyler())
	return ExitOK
}

func printHolds(w io.Writer, holds []snapshot.Hold, st *styler) {
	if len(holds) == 0 {
		fmt.Fprintln(w, st.dim("no held packages"))
		return
	}
	fmt.Fprintln(w, st.title("Held packages"))
	for _, h := range holds {
		fmt.Fprintf(w, "  %s  %s\n", st.mgrName(h.Source), h.Name)
	}
}
