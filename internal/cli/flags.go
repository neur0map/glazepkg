package cli

import (
	"strings"

	"github.com/neur0map/glazepkg/internal/manager"
)

// managerAliases maps friendly shorthands to canonical source names so
// `--cask` and `--choco` resolve like the real thing.
var managerAliases = map[string]string{
	"cask":           "brew-cask",
	"choco":          "chocolatey",
	"windows-update": "windows-updates",
	"winupdate":      "windows-updates",
}

// prepManagerArgs normalizes a subcommand's args: it lifts inline manager
// selectors (`-aur`, `--brew`, `--cask`) and any `--manager`/`-m` values into
// a single leading `--manager a,b` flag, then reorders remaining flags ahead
// of positionals. This lets users write `gpk -S ffmpeg --aur` the way they
// would with yay.
func prepManagerArgs(args []string, mgrs []manager.Manager, valueFlags ...string) []string {
	known := make(map[string]bool, len(mgrs))
	for _, m := range mgrs {
		known[string(m.Name())] = true
	}

	var rest, selected []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			rest = append(rest, args[i:]...)
			break
		}
		switch {
		case a == "--manager" || a == "-m":
			if i+1 < len(args) {
				selected = append(selected, splitSelectors(args[i+1])...)
				i++
			}
			continue
		case strings.HasPrefix(a, "--manager="):
			selected = append(selected, splitSelectors(strings.TrimPrefix(a, "--manager="))...)
			continue
		case strings.HasPrefix(a, "-m="):
			selected = append(selected, splitSelectors(strings.TrimPrefix(a, "-m="))...)
			continue
		}
		if name, ok := inlineManager(a, known); ok {
			selected = append(selected, name)
			continue
		}
		rest = append(rest, a)
	}

	if len(selected) > 0 {
		rest = append([]string{"--manager", strings.Join(dedupStrings(selected), ",")}, rest...)
	}
	return reorderFlagsFirst(rest, append([]string{"manager", "m"}, valueFlags...))
}

// inlineManager reports whether tok is a bare `-name`/`--name` manager
// selector and returns the canonical name. Tokens with `=` or that aren't
// known managers are left for normal flag parsing.
func inlineManager(tok string, known map[string]bool) (string, bool) {
	if !strings.HasPrefix(tok, "-") || tok == "-" || strings.Contains(tok, "=") {
		return "", false
	}
	name := strings.ToLower(strings.TrimLeft(tok, "-"))
	if alias, ok := managerAliases[name]; ok {
		name = alias
	}
	if known[name] {
		return name, true
	}
	return "", false
}

func splitSelectors(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func dedupStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// reorderFlagsFirst moves flag-looking tokens (and their values for known
// string-valued flag names) ahead of positional arguments so users can write
// `gpk installed git --json` without flags being misparsed as positional.
//
// stringFlags is the list of flag names (without dashes) that consume the
// following arg as their value. For Phase 1, these are "manager" and "m".
// All other flags are treated as boolean (no following value).
//
// The `--` separator (POSIX "end of flags") stops scanning; everything after
// it stays in positional order. This lets users force a package name that
// happens to start with a dash.
func reorderFlagsFirst(args []string, stringFlags []string) []string {
	takesValue := make(map[string]bool, len(stringFlags))
	for _, f := range stringFlags {
		takesValue[f] = true
	}

	var flags, positional []string
	i := 0
	for ; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			// Everything after `--` is positional, including dash-prefixed args.
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			positional = append(positional, a)
			continue
		}
		flags = append(flags, a)
		// "--name=value" form: value is inline, no extra arg to consume.
		if strings.Contains(a, "=") {
			continue
		}
		// Strip dashes to get the flag name.
		name := strings.TrimLeft(a, "-")
		if takesValue[name] && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positional...)
}
