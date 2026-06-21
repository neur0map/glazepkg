package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func init() {
	subcommands["managers"] = runManagers
}

type managerStat struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Count     int    `json:"count"`
}

func runManagers(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
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

	counts := make(map[model.Source]int)
	if pkgs, err := collectPackages(mgrs, *noCacheFlag, *quietFlag, stderr, true); err == nil {
		for _, p := range pkgs {
			counts[p.Source]++
		}
	}

	stats := make([]managerStat, 0, len(mgrs))
	available := 0
	for _, m := range mgrs {
		avail := m.Available()
		if avail {
			available++
		}
		if *availFlag && !avail {
			continue
		}
		stats = append(stats, managerStat{Name: string(m.Name()), Available: avail, Count: counts[m.Name()]})
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
	for _, s := range stats {
		mark := st.dim("✗")
		name := st.dim(padRight(s.Name, 16))
		count := ""
		if s.Available {
			mark = st.ok("✓")
			name = st.paint(padRight(s.Name, 16), st.mgrColorOf(model.Source(s.Name)), true)
			count = st.version(strconv.Itoa(s.Count))
		}
		fmt.Fprintf(stdout, "  %s  %s  %s\n", mark, name, count)
	}
	return ExitOK
}
