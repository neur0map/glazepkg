package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/neur0map/glazepkg/internal/config"
	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func init() {
	subcommands["managers"] = runManagers
}

type managerStat struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Skipped   bool   `json:"skipped,omitempty"`
	Count     int    `json:"count"`
}

func runManagers(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "skip":
			return editSkipList(args[1:], mgrs, true, stdout, stderr)
		case "unskip":
			return editSkipList(args[1:], mgrs, false, stdout, stderr)
		default:
			fmt.Fprintf(stderr, "error: unknown managers action %q (want skip or unskip)\n", args[0])
			return ExitErr
		}
	}
	return listManagers(args, mgrs, version, stdout, stderr)
}

// editSkipList adds or removes managers from `[managers] skip`. Skipping is the
// answer to a manager that fails identically on every run: one broken pnpm
// otherwise makes `gpk upgrade` exit non-zero forever.
func editSkipList(names []string, mgrs []manager.Manager, add bool, stdout, stderr io.Writer) int {
	if len(names) == 0 {
		fmt.Fprintln(stderr, "usage: gpk managers skip|unskip <name>...")
		return ExitErr
	}
	known := make(map[string]bool, len(mgrs))
	for _, m := range mgrs {
		known[string(m.Name())] = true
	}

	cfg := config.Load()
	current := make(map[string]bool, len(cfg.Managers.Skip))
	for _, n := range cfg.Managers.Skip {
		current[n] = true
	}
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if alias, ok := managerAliases[name]; ok {
			name = alias
		}
		// Unskip accepts an unknown name so a typo already in the config can be
		// removed; skip does not, or the typo would silently do nothing.
		if add && !known[name] {
			fmt.Fprintf(stderr, "error: unknown manager %q (known: %s)\n", name, knownNames(mgrs))
			return ExitErr
		}
		if add {
			current[name] = true
		} else {
			delete(current, name)
		}
	}

	cfg.Managers.Skip = make([]string, 0, len(current))
	for n := range current {
		cfg.Managers.Skip = append(cfg.Managers.Skip, n)
	}
	sort.Strings(cfg.Managers.Skip)
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(stderr, "error: saving config: %v\n", err)
		return ExitErr
	}

	st := newStyler()
	if len(cfg.Managers.Skip) == 0 {
		fmt.Fprintf(stdout, "%s %s\n", st.ok("✓"), st.dim("no managers skipped"))
		return ExitOK
	}
	fmt.Fprintf(stdout, "%s skipping %s\n", st.ok("✓"),
		st.paint(strings.Join(cfg.Managers.Skip, ", "), st.pal.White, true))
	fmt.Fprintln(stdout, "  "+st.dim("these are left out of every command unless you name one with --manager"))
	return ExitOK
}

func listManagers(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("managers", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		jsonFlag    = fs.Bool("json", false, "emit JSON envelope")
		availFlag   = fs.Bool("available", false, "only show managers detected on this system")
		noCacheFlag = fs.Bool("no-cache", false, "bypass the scan cache for package counts")
		quietFlag   = fs.Bool("quiet", false, "suppress progress on stderr")
	)
	fs.BoolVar(quietFlag, "q", false, "alias for --quiet")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}

	skip := manager.Skipped()
	counts := make(map[model.Source]int)
	if pkgs, err := collectPackages(dropSkipped(mgrs, nil), *noCacheFlag, *quietFlag, stderr, true); err == nil {
		for _, p := range pkgs {
			counts[p.Source]++
		}
	}

	stats := make([]managerStat, 0, len(mgrs))
	available := 0
	for _, m := range mgrs {
		// A skipped manager is reported as skipped rather than counted as
		// available, so a config setting can never look like a detection bug.
		avail := m.Available() && !skip[m.Name()]
		if avail {
			available++
		}
		if *availFlag && !avail {
			continue
		}
		stats = append(stats, managerStat{
			Name:      string(m.Name()),
			Available: avail,
			Skipped:   skip[m.Name()],
			Count:     counts[m.Name()],
		})
	}

	if *jsonFlag {
		if err := writeEnvelope(stdout, version, stats); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	// Available first (most packages first), then the rest by name.
	sort.SliceStable(stats, func(i, j int) bool {
		if stats[i].Available != stats[j].Available {
			return stats[i].Available
		}
		if stats[i].Available && stats[i].Count != stats[j].Count {
			return stats[i].Count > stats[j].Count
		}
		return stats[i].Name < stats[j].Name
	})

	st := newStyler()
	fmt.Fprintf(stdout, "%s  %s\n\n", st.title("Package managers"),
		st.dim(fmt.Sprintf("%d of %d available", available, len(mgrs))))

	maxCount, total, countW := 0, 0, 1
	for _, s := range stats {
		total += s.Count
		if s.Count > maxCount {
			maxCount = s.Count
		}
	}
	countW = len(strconv.Itoa(maxCount))

	const barW = 24
	for _, s := range stats {
		if s.Skipped {
			fmt.Fprintf(stdout, "  %s  %s  %s\n", st.warn("−"), st.dim(padRight(s.Name, 16)),
				st.dim("skipped · gpk managers unskip "+s.Name))
			continue
		}
		if !s.Available {
			fmt.Fprintf(stdout, "  %s  %s\n", st.dim("✗"), st.dim(padRight(s.Name, 16)))
			continue
		}
		name := st.paint(padRight(s.Name, 16), st.mgrColorOf(model.Source(s.Name)), true)
		count := st.version(fmt.Sprintf("%*d", countW, s.Count))
		bar := ""
		if st.on && maxCount > 0 {
			filled := s.Count * barW / maxCount
			if filled == 0 && s.Count > 0 {
				filled = 1
			}
			bar = "  " + st.paint(strings.Repeat("█", filled), st.mgrColorOf(model.Source(s.Name)), false) + st.dim(strings.Repeat("·", barW-filled))
		}
		fmt.Fprintf(stdout, "  %s  %s  %s%s\n", st.ok("✓"), name, count, bar)
	}
	if total > 0 {
		fmt.Fprintf(stdout, "\n  %s %s\n", st.dim("total"), st.version(strconv.Itoa(total)))
	}
	return ExitOK
}
