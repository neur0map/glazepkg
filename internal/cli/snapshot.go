package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
	"github.com/neur0map/glazepkg/internal/snapshot"
)

func init() {
	subcommands["snapshot"] = runSnapshot
	subcommands["snap"] = runSnapshot
}

// runSnapshot exposes the snapshot store the TUI writes with `s`, so a timer
// unit or launchd agent can take periodic snapshots and diff any two of them
// later (#64).
func runSnapshot(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	action := "save"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		action, args = args[0], args[1:]
	}
	switch action {
	case "save":
		return snapshotSave(args, mgrs, version, stdout, stderr)
	case "list", "ls":
		return snapshotList(args, version, stdout, stderr)
	case "diff":
		return snapshotDiff(args, mgrs, version, stdout, stderr)
	case "prune":
		return snapshotPrune(args, version, stdout, stderr)
	}
	fmt.Fprintf(stderr, "error: unknown snapshot action %q (want save, list, diff or prune)\n", action)
	return ExitErr
}

func snapshotSave(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer) int {
	args = prepManagerArgs(args, mgrs, "keep")
	fs := flag.NewFlagSet("snapshot save", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag   = fs.String("manager", "", "comma list of managers (default: all)")
		quietFlag = fs.Bool("quiet", false, "suppress progress on stderr")
		jsonFlag  = fs.Bool("json", false, "emit JSON envelope")
		keepFlag  = fs.Int("keep", 0, "after saving, delete all but the newest N snapshots")
	)
	// Accepted and ignored: a snapshot always scans live, so --no-cache is
	// already the only behaviour.
	fs.Bool("no-cache", false, "accepted for symmetry; snapshots always scan live")
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
	// A snapshot is a record of what was installed at a moment, so it must never
	// come from a stale cache.
	pkgs, err := collectPackages(filtered, true, *quietFlag, stderr, cacheWriteOKFor(*mgrFlag))
	if err != nil {
		fmt.Fprintf(stderr, "error: scan failed: %v\n", err)
		return ExitErr
	}

	snap := snapshot.New(pkgs)
	path, err := snapshot.Save(snap)
	if err != nil {
		fmt.Fprintf(stderr, "error: saving snapshot: %v\n", err)
		return ExitErr
	}
	removed := 0
	if *keepFlag > 0 {
		if removed, err = pruneSnapshots(*keepFlag); err != nil {
			fmt.Fprintf(stderr, "error: pruning snapshots: %v\n", err)
			return ExitErr
		}
	}

	if *jsonFlag {
		data := struct {
			Path     string    `json:"path"`
			Packages int       `json:"packages"`
			Pruned   int       `json:"pruned"`
			Taken    time.Time `json:"taken"`
		}{Path: path, Packages: len(snap.Packages), Pruned: removed, Taken: snap.Timestamp}
		if err := writeEnvelope(stdout, version, data); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	st := newStyler()
	fmt.Fprintf(stdout, "%s saved %s to %s\n", st.ok("✓"),
		st.version(plural(len(snap.Packages), "package", "packages")), st.dim(path))
	if removed > 0 {
		fmt.Fprintf(stdout, "  %s\n", st.dim(fmt.Sprintf("pruned %d older snapshot(s)", removed)))
	}
	return ExitOK
}

type snapshotRow struct {
	Index    int       `json:"index"`
	Path     string    `json:"path"`
	Taken    time.Time `json:"taken"`
	Packages int       `json:"packages"`
}

func snapshotList(args []string, version string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("snapshot list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonFlag := fs.Bool("json", false, "emit JSON envelope")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}

	rows, err := listSnapshots()
	if err != nil {
		fmt.Fprintf(stderr, "error: reading snapshots: %v\n", err)
		return ExitErr
	}
	if *jsonFlag {
		if rows == nil {
			rows = []snapshotRow{}
		}
		if err := writeEnvelope(stdout, version, rows); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}

	st := newStyler()
	if len(rows) == 0 {
		fmt.Fprintln(stdout, st.dim("(no snapshots — run `gpk snapshot` to take one)"))
		return ExitOK
	}
	fmt.Fprintln(stdout, st.title(plural(len(rows), "snapshot", "snapshots")))
	for _, r := range rows {
		fmt.Fprintf(stdout, "  %s  %s  %s\n",
			st.num(fmt.Sprintf("%3d", r.Index)),
			st.paint(r.Taken.Format("2006-01-02 15:04:05"), st.pal.White, true),
			st.dim(plural(r.Packages, "package", "packages")))
	}
	if len(rows) > 1 {
		fmt.Fprintln(stdout, "\n"+st.dim("diff two of them: ")+st.accent("gpk snapshot diff 2 1"))
	}
	return ExitOK
}

// listSnapshots numbers the saved snapshots newest-first, so index 1 is the one
// `snapshot.Latest` returns and the numbers match what `snapshot list` printed.
func listSnapshots() ([]snapshotRow, error) {
	paths, err := snapshot.List()
	if err != nil {
		return nil, err
	}
	rows := make([]snapshotRow, 0, len(paths))
	for i, p := range paths {
		snap, err := snapshot.Load(p)
		if err != nil {
			continue
		}
		rows = append(rows, snapshotRow{
			Index:    i + 1,
			Path:     p,
			Taken:    snap.Timestamp,
			Packages: len(snap.Packages),
		})
	}
	return rows, nil
}

func snapshotDiff(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("snapshot diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag   = fs.String("manager", "", "comma list of managers (default: all)")
		quietFlag = fs.Bool("quiet", false, "suppress progress on stderr")
		jsonFlag  = fs.Bool("json", false, "emit JSON envelope")
	)
	fs.StringVar(mgrFlag, "m", "", "alias for --manager")
	fs.BoolVar(quietFlag, "q", false, "alias for --quiet")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}
	if fs.NArg() > 2 {
		fmt.Fprintln(stderr, "error: snapshot diff takes at most two selectors")
		return ExitErr
	}

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	// Bare `diff` reproduces the TUI's `d`: newest saved snapshot against the
	// system as it is now.
	oldSel, newSel := "1", "live"
	switch fs.NArg() {
	case 1:
		oldSel = fs.Arg(0)
	case 2:
		oldSel, newSel = fs.Arg(0), fs.Arg(1)
	}

	rows, err := listSnapshots()
	if err != nil {
		fmt.Fprintf(stderr, "error: reading snapshots: %v\n", err)
		return ExitErr
	}
	live := func() (*model.Snapshot, error) {
		pkgs, err := collectPackages(filtered, true, *quietFlag, stderr, cacheWriteOKFor(*mgrFlag))
		if err != nil {
			return nil, err
		}
		return snapshot.New(pkgs), nil
	}

	oldSnap, oldLabel, err := resolveSnapshot(oldSel, rows, live)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}
	newSnap, newLabel, err := resolveSnapshot(newSel, rows, live)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	diff := model.ComputeDiff(oldSnap, newSnap)
	if *mgrFlag != "" {
		diff = filterDiffByManagers(diff, filtered)
	}
	sortDiff(&diff)

	if *jsonFlag {
		if err := writeEnvelope(stdout, version, toDiffJSON(oldSnap, newSnap, diff)); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}
	writeDiffHuman(stdout, newStyler(), oldLabel, newLabel, diff)
	return ExitOK
}

// resolveSnapshot turns a selector into a snapshot: "live" scans the system,
// "latest" is the newest saved one, a bare number is an index from
// `snapshot list`, and anything else is a file path.
func resolveSnapshot(sel string, rows []snapshotRow, live func() (*model.Snapshot, error)) (*model.Snapshot, string, error) {
	switch sel {
	case "live", "now":
		snap, err := live()
		return snap, "now", err
	case "latest":
		sel = "1"
	}
	if i, err := strconv.Atoi(sel); err == nil {
		if i < 1 || i > len(rows) {
			if len(rows) == 0 {
				return nil, "", errors.New("no snapshots saved yet; run `gpk snapshot` first")
			}
			return nil, "", fmt.Errorf("snapshot %d out of range (1-%d); see `gpk snapshot list`", i, len(rows))
		}
		row := rows[i-1]
		snap, err := snapshot.Load(row.Path)
		return snap, row.Taken.Format("2006-01-02 15:04:05"), err
	}
	snap, err := snapshot.Load(sel)
	if err != nil {
		return nil, "", fmt.Errorf("reading snapshot %q: %w", sel, err)
	}
	return snap, filepath.Base(sel), nil
}

// filterDiffByManagers narrows a diff to the selected managers. Snapshots are
// always whole-system, so the filter has to be applied after the comparison
// rather than before it.
func filterDiffByManagers(d model.Diff, mgrs []manager.Manager) model.Diff {
	allow := make(map[model.Source]bool, len(mgrs))
	for _, m := range mgrs {
		allow[m.Name()] = true
	}
	out := model.Diff{}
	for _, p := range d.Added {
		if allow[p.Source] {
			out.Added = append(out.Added, p)
		}
	}
	for _, p := range d.Removed {
		if allow[p.Source] {
			out.Removed = append(out.Removed, p)
		}
	}
	for _, e := range d.Upgraded {
		if allow[e.New.Source] {
			out.Upgraded = append(out.Upgraded, e)
		}
	}
	return out
}

// sortDiff makes the output stable: ComputeDiff walks a map, so without this
// two runs over the same pair of snapshots print in different orders.
func sortDiff(d *model.Diff) {
	bySourceName := func(a, b model.Package) bool {
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.Name < b.Name
	}
	sort.Slice(d.Added, func(i, j int) bool { return bySourceName(d.Added[i], d.Added[j]) })
	sort.Slice(d.Removed, func(i, j int) bool { return bySourceName(d.Removed[i], d.Removed[j]) })
	sort.Slice(d.Upgraded, func(i, j int) bool { return bySourceName(d.Upgraded[i].New, d.Upgraded[j].New) })
}

type diffEntryJSON struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	Version string `json:"version,omitempty"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
}

func toDiffJSON(oldSnap, newSnap *model.Snapshot, d model.Diff) any {
	out := struct {
		From     time.Time       `json:"from"`
		To       time.Time       `json:"to"`
		Added    []diffEntryJSON `json:"added"`
		Upgraded []diffEntryJSON `json:"upgraded"`
		Removed  []diffEntryJSON `json:"removed"`
	}{
		From:     oldSnap.Timestamp,
		To:       newSnap.Timestamp,
		Added:    []diffEntryJSON{},
		Upgraded: []diffEntryJSON{},
		Removed:  []diffEntryJSON{},
	}
	for _, p := range d.Added {
		out.Added = append(out.Added, diffEntryJSON{Name: p.Name, Source: string(p.Source), Version: p.Version})
	}
	for _, e := range d.Upgraded {
		out.Upgraded = append(out.Upgraded, diffEntryJSON{
			Name: e.New.Name, Source: string(e.New.Source), From: e.Old.Version, To: e.New.Version,
		})
	}
	for _, p := range d.Removed {
		out.Removed = append(out.Removed, diffEntryJSON{Name: p.Name, Source: string(p.Source), Version: p.Version})
	}
	return out
}

func writeDiffHuman(w io.Writer, st *styler, oldLabel, newLabel string, d model.Diff) {
	fmt.Fprintf(w, "%s %s\n", st.title("Changes"),
		st.dim(oldLabel+" → "+newLabel))
	if len(d.Added)+len(d.Upgraded)+len(d.Removed) == 0 {
		fmt.Fprintln(w, st.dim("  (no changes)"))
		return
	}
	nameW := 4
	for _, p := range d.Added {
		nameW = max(nameW, len(p.Name))
	}
	for _, p := range d.Removed {
		nameW = max(nameW, len(p.Name))
	}
	for _, e := range d.Upgraded {
		nameW = max(nameW, len(e.New.Name))
	}
	for _, p := range d.Added {
		fmt.Fprintf(w, "  %s %s  %s  %s\n", st.ok("+"),
			st.paint(padRight(p.Name, nameW), st.pal.White, true),
			st.mgrName(p.Source), st.version(p.Version))
	}
	for _, e := range d.Upgraded {
		fmt.Fprintf(w, "  %s %s  %s  %s %s %s\n", st.warn("↑"),
			st.paint(padRight(e.New.Name, nameW), st.pal.White, true),
			st.mgrName(e.New.Source), st.dim(e.Old.Version), st.dim("→"), st.version(e.New.Version))
	}
	for _, p := range d.Removed {
		fmt.Fprintf(w, "  %s %s  %s  %s\n", st.bad("-"),
			st.paint(padRight(p.Name, nameW), st.pal.White, true),
			st.mgrName(p.Source), st.dim(p.Version))
	}
	fmt.Fprintf(w, "\n  %s\n", st.dim(fmt.Sprintf("+%d added  ↑%d upgraded  -%d removed",
		len(d.Added), len(d.Upgraded), len(d.Removed))))
}

func snapshotPrune(args []string, version string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("snapshot prune", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		keepFlag = fs.Int("keep", 10, "how many of the newest snapshots to keep")
		jsonFlag = fs.Bool("json", false, "emit JSON envelope")
	)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}
	if *keepFlag < 1 {
		fmt.Fprintln(stderr, "error: --keep must be at least 1")
		return ExitErr
	}

	removed, err := pruneSnapshots(*keepFlag)
	if err != nil {
		fmt.Fprintf(stderr, "error: pruning snapshots: %v\n", err)
		return ExitErr
	}
	if *jsonFlag {
		data := struct {
			Pruned int `json:"pruned"`
			Kept   int `json:"kept"`
		}{Pruned: removed, Kept: *keepFlag}
		if err := writeEnvelope(stdout, version, data); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		return ExitOK
	}
	st := newStyler()
	fmt.Fprintf(stdout, "%s pruned %s\n", st.ok("✓"), st.version(plural(removed, "snapshot", "snapshots")))
	return ExitOK
}

// pruneSnapshots deletes all but the newest keep snapshots and reports how many
// went away. Periodic snapshots grow without bound otherwise.
func pruneSnapshots(keep int) (int, error) {
	paths, err := snapshot.List()
	if err != nil {
		return 0, err
	}
	if len(paths) <= keep {
		return 0, nil
	}
	removed := 0
	for _, p := range paths[keep:] {
		if err := os.Remove(p); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
