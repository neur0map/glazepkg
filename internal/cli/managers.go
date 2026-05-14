package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/neur0map/glazepkg/internal/manager"
)

// parseManagerFilter resolves a --manager flag value against the registered
// manager set. Accepted forms:
//
//	""               → all managers (default)
//	"all"            → all managers (explicit)
//	"pacman"         → just pacman
//	"pacman,aur"     → union of named managers
//	"!brew,!cask"    → everything except the named managers
//
// Mixing positive and negative selectors in one filter is an error: it's
// almost always a mistake by the user. Unknown manager names are also an
// error; the error message includes the sorted list of known names so the
// user can see what they typed wrong.
func parseManagerFilter(value string, all []manager.Manager) ([]manager.Manager, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "all" {
		return all, nil
	}

	parts := strings.Split(value, ",")
	var positive, negative []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "!") {
			// Trim again after stripping "!" so "! pacman" and "!pacman" behave
			// the same — a space slip-up shouldn't produce a confusing error.
			negative = append(negative, strings.TrimSpace(strings.TrimPrefix(p, "!")))
		} else {
			positive = append(positive, p)
		}
	}

	if len(positive) > 0 && len(negative) > 0 {
		return nil, fmt.Errorf("--manager: cannot mix positive (%q) and negative (%q) selectors",
			positive, negative)
	}

	known := make(map[string]manager.Manager, len(all))
	for _, m := range all {
		known[string(m.Name())] = m
	}

	check := func(name string) error {
		if _, ok := known[name]; !ok {
			return fmt.Errorf("--manager: unknown manager %q (known: %s)", name, knownNames(all))
		}
		return nil
	}

	if len(positive) > 0 {
		var out []manager.Manager
		seen := make(map[string]bool)
		for _, name := range positive {
			if err := check(name); err != nil {
				return nil, err
			}
			if seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, known[name])
		}
		return out, nil
	}

	// Negation mode: validate names first, then filter.
	excluded := make(map[string]bool, len(negative))
	for _, name := range negative {
		if err := check(name); err != nil {
			return nil, err
		}
		excluded[name] = true
	}
	var out []manager.Manager
	for _, m := range all {
		if excluded[string(m.Name())] {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// knownNames returns a sorted, comma-separated string of manager names for
// error messages. Sorted because error messages get diffed in tests.
func knownNames(all []manager.Manager) string {
	names := make([]string, len(all))
	for i, m := range all {
		names[i] = string(m.Name())
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
