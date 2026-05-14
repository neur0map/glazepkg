package cli

import "strings"

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
