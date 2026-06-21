// Package cli implements gpk's headless subcommands.
//
// Dispatch is the single entry point from cmd/gpk/main.go. Tests call it
// directly with synthetic managers and bytes.Buffer streams; production wires
// it against manager.All() and os.Stdout/os.Stderr.
package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/neur0map/glazepkg/internal/manager"
)

// subcommandFunc is the signature every subcommand handler implements.
// args excludes the subcommand name itself (Dispatch strips it).
type subcommandFunc func(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int

// subcommands maps subcommand name → handler. Subcommand files register
// themselves here via init() so adding a new one is a one-file change.
var subcommands = map[string]subcommandFunc{}

// Dispatch routes args[0] to the matching subcommand. Returns the exit code
// the caller should propagate (cmd/gpk/main.go does os.Exit on the result).
//
// An unknown name that closely matches a real subcommand yields a "did you
// mean" hint; any other bareword is treated as a search query with an
// interactive install picker, the way `yay <pkg>` behaves.
func Dispatch(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "error: no subcommand specified")
		return ExitErr
	}
	name := args[0]
	if fn, ok := subcommands[name]; ok {
		return fn(args[1:], mgrs, version, stdout, stderr, stdin)
	}
	if strings.HasPrefix(name, "-") {
		fmt.Fprintf(stderr, "error: unknown option %q\n", name)
		return ExitErr
	}
	if best, d := closest(name, SubcommandNames()); d == 1 || (d == 2 && len(name) >= 6) {
		fmt.Fprintf(stderr, "error: unknown command %q — did you mean %q?\n", name, best)
		return ExitErr
	}
	if search, ok := subcommands["search"]; ok {
		return search(append([]string{"--install"}, args...), mgrs, version, stdout, stderr, stdin)
	}
	fmt.Fprintf(stderr, "error: unknown subcommand %q\n", name)
	return ExitErr
}

// SubcommandNames returns the registered subcommand names, sorted, for use
// in help text. Stable order so help output doesn't churn between builds.
func SubcommandNames() []string {
	names := make([]string, 0, len(subcommands))
	for n := range subcommands {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
