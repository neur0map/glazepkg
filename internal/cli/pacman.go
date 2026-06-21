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
			return append([]string{"install"}, rest...), true
		}
	case 'R':
		head := []string{"remove"}
		if has('s') || has('n') {
			head = append(head, "--with-deps")
		}
		return append(head, rest...), true
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
