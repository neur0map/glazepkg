package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
	"github.com/neur0map/glazepkg/internal/snapshot"
)

func init() {
	subcommands["outdated"] = runOutdated
}

type outdatedEntry struct {
	Name    string `json:"name"`
	Current string `json:"current"`
	Latest  string `json:"latest"`
	Source  string `json:"source"`
}

func runOutdated(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("outdated", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag      = fs.String("manager", "", "comma list of managers (default: all)")
		countFlag    = fs.Bool("count", false, "emit only the integer count on stdout")
		exitCodeFlag = fs.Bool("exit-code", false, "exit 2 if any updates available")
		jsonFlag     = fs.Bool("json", false, "emit JSON envelope")
		noCacheFlag  = fs.Bool("no-cache", false, "force fresh CheckUpdates")
		quietFlag    = fs.Bool("quiet", false, "suppress progress on stderr")
	)
	fs.BoolVar(quietFlag, "q", false, "alias for --quiet")
	fs.StringVar(mgrFlag, "m", "", "alias for --manager")
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

	cacheOK := cacheWriteOKFor(*mgrFlag)
	pkgs, err := collectPackages(filtered, *noCacheFlag, *quietFlag, stderr, cacheOK)
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	cache := manager.NewUpdateCache()
	if *noCacheFlag {
		// Invalidate cache for the filtered managers so FetchUpdates always
		// goes to the live source.
		var keys []string
		for _, p := range pkgs {
			keys = append(keys, p.Key())
		}
		cache.Invalidate(keys)
	}
	updates := manager.FetchUpdates(filtered, pkgs, cache)

	// Build entries. Held packages are pinned, so they're left out.
	holds := snapshot.LoadHolds()
	entries := make([]outdatedEntry, 0, len(updates))
	for _, p := range pkgs {
		latest, ok := updates[p.Key()]
		if !ok || latest == "" || latest == p.Version {
			continue
		}
		if snapshot.IsHeld(holds, p.Source, p.Name) {
			continue
		}
		entries = append(entries, outdatedEntry{
			Name:    p.Name,
			Current: p.Version,
			Latest:  latest,
			Source:  string(p.Source),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Source != entries[j].Source {
			return entries[i].Source < entries[j].Source
		}
		return entries[i].Name < entries[j].Name
	})

	switch {
	case *countFlag:
		fmt.Fprintf(stdout, "%d\n", len(entries))
	case *jsonFlag:
		// Force [] not null for empty list. encoding/json emits [] for a
		// non-nil empty slice.
		if entries == nil {
			entries = []outdatedEntry{}
		}
		if err := writeEnvelope(stdout, version, entries); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
	default:
		writeOutdatedHuman(stdout, entries, newStyler())
	}

	if *exitCodeFlag && len(entries) > 0 {
		return ExitNegative
	}
	return ExitOK
}

func writeOutdatedHuman(w io.Writer, entries []outdatedEntry, st *styler) {
	if len(entries) == 0 {
		fmt.Fprintln(w, st.dim("(no updates)"))
		return
	}
	nameW, srcW := 4, 6
	for _, e := range entries {
		if len(e.Name) > nameW {
			nameW = len(e.Name)
		}
		if len(e.Source) > srcW {
			srcW = len(e.Source)
		}
	}
	plural := "s"
	if len(entries) == 1 {
		plural = ""
	}
	fmt.Fprintln(w, st.title(fmt.Sprintf("%d update%s available", len(entries), plural)))
	for _, e := range entries {
		name := st.paint(padRight(e.Name, nameW), st.pal.White, true)
		src := st.paint(padRight(e.Source, srcW), st.mgrColorOf(model.Source(e.Source)), true)
		fmt.Fprintf(w, "  %s  %s  %s %s %s\n", name, src, st.dim(e.Current), st.dim("→"), st.warn(e.Latest))
	}
	fmt.Fprintln(w, "\n"+st.dim("run ")+st.accent("gpk upgrade")+st.dim(" to apply (or gpk upgrade <name>)"))
}
