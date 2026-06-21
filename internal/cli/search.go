package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func init() {
	subcommands["search"] = runSearch
}

type searchRow struct {
	pkg       model.Package
	mgr       manager.Manager
	installed bool
}

func runSearch(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs, "limit")
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag     = fs.String("manager", "", "comma list of managers (default: all)")
		jsonFlag    = fs.Bool("json", false, "emit JSON envelope")
		installFlag = fs.Bool("install", false, "pick results to install interactively")
		yesFlag     = fs.Bool("yes", false, "skip the install confirmation prompt")
		quietFlag   = fs.Bool("quiet", false, "suppress progress messages on stderr")
		limitFlag   = fs.Int("limit", 60, "maximum results to display")
	)
	fs.BoolVar(installFlag, "i", false, "alias for --install")
	fs.BoolVar(yesFlag, "y", false, "alias for --yes")
	fs.BoolVar(quietFlag, "q", false, "alias for --quiet")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}

	terms := fs.Args()
	if len(terms) == 0 {
		fmt.Fprintln(stderr, "error: search requires a query")
		return ExitErr
	}
	query := strings.Join(terms, " ")

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	rows := searchManagers(filtered, query)
	rankSearchRows(rows, query)
	markInstalledRows(rows)
	if *limitFlag > 0 && len(rows) > *limitFlag {
		rows = rows[:*limitFlag]
	}

	if len(rows) == 0 {
		fmt.Fprintf(stderr, "no packages found for %q\n", query)
		if sug := suggestPackages(query, filtered); len(sug) > 0 {
			fmt.Fprintf(stderr, "did you mean: %s\n", strings.Join(sug, ", "))
		}
		return ExitNegative
	}

	if *jsonFlag {
		out := make([]cliPackage, len(rows))
		for i, r := range rows {
			out[i] = toCLIPackage(r.pkg)
		}
		if err := writeEnvelope(stdout, version, out); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	st := newStyler()
	writeSearchHuman(stdout, query, rows, st)

	if !*installFlag {
		return ExitOK
	}

	r := newPromptReader(stdin)
	if !canPrompt(r) {
		return ExitOK
	}
	input, ok := readSelection("\n"+st.accent("==> ")+"packages to install (eg: 1 2 3, 1-3): ", r, stdout)
	if !ok || input == "" {
		return ExitOK
	}
	idxs, err := parseSelection(input, len(rows))
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}
	var plans []installPlan
	for _, i := range idxs {
		plan, code, ok := buildPlan(candidate{mgr: rows[i].mgr, pkg: rows[i].pkg}, "", stderr)
		if !ok {
			return code
		}
		plans = append(plans, plan)
	}
	return executeInstalls(plans, *yesFlag, false, *quietFlag, st, r, stdout, stderr)
}

// searchManagers queries every available Searcher in parallel and returns one
// row per source/name match, mapped to the canonical installing manager.
func searchManagers(filtered []manager.Manager, query string) []searchRow {
	type res struct {
		pkgs []model.Package
		from model.Source
	}
	var wg sync.WaitGroup
	ch := make(chan res, len(filtered))
	for _, m := range filtered {
		s, ok := m.(manager.Searcher)
		if !ok || !m.Available() {
			continue
		}
		wg.Add(1)
		go func(s manager.Searcher, from model.Source) {
			defer wg.Done()
			pkgs, err := s.Search(query)
			if err != nil {
				pkgs = nil
			}
			ch <- res{pkgs: pkgs, from: from}
		}(s, m.Name())
	}
	go func() { wg.Wait(); close(ch) }()

	seen := make(map[string]bool)
	var rows []searchRow
	for r := range ch {
		for _, p := range r.pkgs {
			src := p.Source
			if src == "" {
				src = r.from
			}
			key := string(src) + ":" + p.Name
			if seen[key] {
				continue
			}
			canonical := mgrByName(filtered, src)
			if canonical == nil {
				continue
			}
			seen[key] = true
			p.Source = src
			rows = append(rows, searchRow{pkg: p, mgr: canonical})
		}
	}
	return rows
}

// rankSearchRows orders rows by relevance to query: exact name, then prefix,
// then substring, then alphabetical; ties broken by source for stability.
func rankSearchRows(rows []searchRow, query string) {
	q := strings.ToLower(query)
	score := func(name string) int {
		n := strings.ToLower(name)
		switch {
		case n == q:
			return 0
		case strings.HasPrefix(n, q):
			return 1
		case strings.Contains(n, q):
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		si, sj := score(rows[i].pkg.Name), score(rows[j].pkg.Name)
		if si != sj {
			return si < sj
		}
		if rows[i].pkg.Name != rows[j].pkg.Name {
			return rows[i].pkg.Name < rows[j].pkg.Name
		}
		return rows[i].pkg.Source < rows[j].pkg.Source
	})
}

// markInstalledRows flags rows already present in the scan cache. It never
// triggers a live scan, so search stays fast even on a cold cache.
func markInstalledRows(rows []searchRow) {
	cached := manager.LoadScanCache()
	if len(cached) == 0 {
		return
	}
	installed := make(map[string]bool, len(cached))
	for _, p := range cached {
		installed[string(p.Source)+":"+p.Name] = true
		installed["*:"+p.Name] = true
	}
	for i := range rows {
		if installed[string(rows[i].pkg.Source)+":"+rows[i].pkg.Name] || installed["*:"+rows[i].pkg.Name] {
			rows[i].installed = true
		}
	}
}

func writeSearchHuman(w io.Writer, query string, rows []searchRow, st *styler) {
	fmt.Fprintf(w, "%s %s  %s\n\n", st.title("results for"), st.accent(strconv.Quote(query)), st.dim("("+strconv.Itoa(len(rows))+")"))

	nameW, srcW := 0, 0
	for _, r := range rows {
		if l := len(r.pkg.Name); l > nameW {
			nameW = l
		}
		if l := len(string(r.pkg.Source)); l > srcW {
			srcW = l
		}
	}
	nameW = clamp(nameW, 1, 32)
	srcW = clamp(srcW, 1, 14)
	numW := len(strconv.Itoa(len(rows)))

	for i, r := range rows {
		idx := st.num(fmt.Sprintf("%*d", numW, i+1))
		name := st.paint(padRight(r.pkg.Name, nameW), st.pal.White, true)
		src := st.paint(padRight(string(r.pkg.Source), srcW), st.mgrColorOf(r.pkg.Source), true)
		ver := st.version(r.pkg.Version)
		line := fmt.Sprintf("  %s  %s  %s  %s", idx, name, src, ver)
		if r.installed {
			line += "  " + st.ok("(installed)")
		}
		if r.pkg.Description != "" {
			line += "\n" + strings.Repeat(" ", numW+4) + st.dim(truncate(r.pkg.Description, 72))
		}
		fmt.Fprintln(w, line)
	}
}

func (s *styler) mgrColorOf(src model.Source) string {
	if hex, ok := s.mgr[src]; ok {
		return hex
	}
	return s.pal.Subtext
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
