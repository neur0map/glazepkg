package manager

import (
	"strings"

	"github.com/neur0map/glazepkg/internal/config"
	"github.com/neur0map/glazepkg/internal/model"
)

// Skipped returns the managers the user has told gpk to leave alone via
// `[managers] skip` in the config. Unknown names are kept in the set rather
// than rejected: a typo should not make gpk refuse to run, and `gpk managers`
// shows what is actually being skipped.
func Skipped() map[model.Source]bool {
	names := config.Load().Managers.Skip
	skip := make(map[model.Source]bool, len(names))
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			skip[model.Source(n)] = true
		}
	}
	return skip
}

// Enabled returns every manager except the skipped ones. Surfaces that operate
// on "all managers" without letting the user name one (the TUI) use this;
// All stays the full registry so an explicitly named manager is still
// resolvable.
func Enabled() []Manager {
	skip := Skipped()
	if len(skip) == 0 {
		return All()
	}
	all := All()
	out := make([]Manager, 0, len(all))
	for _, m := range all {
		if !skip[m.Name()] {
			out = append(out, m)
		}
	}
	return out
}
