package cli

import "strings"

// TranslateOps maps a pacman/yay-style invocation onto a gpk subcommand argv.
//
// The first token carries the operation (`-S`, `-Syu`, `-Rns`, `-Qi`, ...);
// the leading letter is the operation and the rest are modifiers, exactly as
// pacman reads them. ok is false when the first token isn't a recognized
// operation, so the caller falls through to normal subcommand dispatch.
//
//	-S pkg      install        -Ss term   search
//	-Su/-Syu    upgrade (all)  -Si pkg    info
//	-Sc[c]      clean          -R pkg     remove
//	-Rs/-Rns    remove --with-deps        -Q         list
//	-Qi pkg     info           -Qu        outdated
//	-Qs term    list <filter>  -Qdt       autoremove --print
func TranslateOps(args []string) (out []string, ok bool) {
	if len(args) == 0 {
		return nil, false
	}
	first := args[0]
	if !strings.HasPrefix(first, "-") || first == "-" || strings.HasPrefix(first, "--") {
		return nil, false
	}
	op := first[1:]
	main, sub := op[0], op[1:]
	rest := rewritePacmanFlags(args[1:])
	has := func(c byte) bool { return strings.IndexByte(sub, c) >= 0 }

	switch main {
	case 'S':
		switch {
		case has('s'):
			return append([]string{"search"}, rest...), true
		case has('i'):
			return append([]string{"info"}, rest...), true
		case has('c'):
			head := []string{"clean"}
			if strings.Count(sub, "c") >= 2 {
				head = append(head, "--all")
			}
			return append(head, rest...), true
		case has('u'):
			return append([]string{"upgrade"}, rest...), true
		default:
			if has('y') && !hasPositional(rest) {
				return append([]string{"refresh"}, rest...), true
			}
			return append([]string{"install"}, dropVerb(rest, "install")...), true
		}
	case 'R':
		head := []string{"remove"}
		if has('s') || has('n') {
			head = append(head, "--with-deps")
		}
		return append(head, dropVerb(rest, "remove")...), true
	case 'Q':
		switch {
		case has('u'):
			return append([]string{"outdated"}, rest...), true
		case has('i'):
			return append([]string{"info"}, rest...), true
		case has('d') && has('t'):
			return append([]string{"autoremove", "--print"}, rest...), true
		default:
			return append([]string{"list"}, rest...), true
		}
	}
	return nil, false
}

// dropVerb removes a redundant leading verb so `gpk -S install foo` (the flag
// and the word) behaves like `gpk -S foo`. Only used for the package-name ops,
// where a package literally named "install"/"remove" is implausible.
func dropVerb(rest []string, verb string) []string {
	if len(rest) > 0 && rest[0] == verb {
		return rest[1:]
	}
	return rest
}

// rewritePacmanFlags maps pacman-only flags onto gpk equivalents and drops
// the ones that have no gpk meaning, so a habitual `--noconfirm`/`--needed`
// doesn't trip the subcommand flag parser.
func rewritePacmanFlags(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		switch a {
		case "--noconfirm":
			out = append(out, "--yes")
		case "--needed", "--noprogressbar", "--quiet":
			// no gpk analogue; quiet is handled per-command via -q
			if a == "--quiet" {
				out = append(out, "-q")
			}
		default:
			out = append(out, a)
		}
	}
	return out
}

// hasPositional reports whether args contains a non-flag argument, skipping the
// values of gpk's value-taking flags. Used to tell a bare `-Sy` (refresh) from
// `-Sy pkg` (refresh + install).
func hasPositional(args []string) bool {
	valueFlags := map[string]bool{"--manager": true, "-m": true, "--limit": true}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			return i+1 < len(args)
		}
		if strings.HasPrefix(a, "-") {
			if valueFlags[a] {
				i++
			}
			continue
		}
		return true
	}
	return false
}
