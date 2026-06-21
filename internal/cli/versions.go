package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
	vercmp "github.com/neur0map/glazepkg/internal/version"
)

func init() {
	subcommands["versions"] = runVersions
}

type versionSet struct {
	Manager  string   `json:"manager"`
	Versions []string `json:"versions"`
}

// runVersions lists the installable versions of a package per manager,
// newest-first via the cross-format comparator. It exposes non-interactively
// (and as --json) what `install --pick-version` shows interactively, so a
// backend or script can build its own version picker.
func runVersions(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	args = prepManagerArgs(args, mgrs)
	fs := flag.NewFlagSet("versions", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mgrFlag  = fs.String("manager", "", "comma list of managers (default: all)")
		jsonFlag = fs.Bool("json", false, "emit a JSON envelope")
	)
	fs.StringVar(mgrFlag, "m", "", "alias for --manager")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "error: versions takes exactly one package name")
		return ExitErr
	}
	name := rest[0]

	filtered, err := parseManagerFilter(*mgrFlag, mgrs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitErr
	}

	type res struct {
		src  model.Source
		vers []string
	}
	var (
		mu       sync.Mutex
		out      []res
		firstErr error
		wg       sync.WaitGroup
	)
	for _, m := range filtered {
		lister, ok := m.(manager.VersionLister)
		if !ok || !m.Available() {
			continue
		}
		wg.Add(1)
		go func(m manager.Manager, l manager.VersionLister) {
			defer wg.Done()
			vs, err := l.Versions(name)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("%s: %w", m.Name(), err)
				}
				mu.Unlock()
				return
			}
			if len(vs) == 0 {
				return
			}
			sort.Slice(vs, func(i, j int) bool { return vercmp.Compare(vs[i], vs[j]) > 0 })
			mu.Lock()
			out = append(out, res{m.Name(), vs})
			mu.Unlock()
		}(m, lister)
	}
	wg.Wait()
	// System managers first (like the install picker / search), then by name.
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := preferRank(out[i].src, defaultPreference), preferRank(out[j].src, defaultPreference)
		if pi != pj {
			return pi < pj
		}
		return out[i].src < out[j].src
	})
	if len(out) == 0 && firstErr != nil {
		fmt.Fprintf(stderr, "error: couldn't reach %s (check your connection)\n", firstErr)
		return ExitErr
	}

	if *jsonFlag {
		sets := make([]versionSet, len(out))
		for i, r := range out {
			sets[i] = versionSet{Manager: string(r.src), Versions: r.vers}
		}
		data := struct {
			Name    string       `json:"name"`
			Sources []versionSet `json:"sources"`
		}{name, sets}
		if err := writeEnvelope(stdout, version, data); err != nil {
			fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
			return ExitErr
		}
		if len(out) == 0 {
			return ExitNegative
		}
		return ExitOK
	}

	st := newStyler()
	if len(out) == 0 {
		fmt.Fprintf(stderr, "error: no installable versions found for %q\n", name)
		return ExitNegative
	}
	fmt.Fprintln(stdout, st.title(name+" versions"))
	const limit = 15
	for _, r := range out {
		shown, extra := r.vers, 0
		if len(shown) > limit {
			extra = len(shown) - limit
			shown = shown[:limit]
		}
		line := "  " + st.mgrName(r.src) + " "
		for _, v := range shown {
			line += " " + st.version(v)
		}
		if extra > 0 {
			line += st.dim(fmt.Sprintf("  (+%d more)", extra))
		}
		fmt.Fprintln(stdout, line)
	}
	return ExitOK
}
