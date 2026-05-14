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

	"github.com/neur0map/glazepkg/internal/manager"
)

// subcommandFunc is the signature every subcommand handler implements.
// args excludes the subcommand name itself (Dispatch strips it).
type subcommandFunc func(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer) int

// subcommands maps subcommand name → handler. Subcommand files register
// themselves here via init() so adding a new one is a one-file change.
var subcommands = map[string]subcommandFunc{}

// Dispatch routes args[0] to the matching subcommand. Returns the exit code
// the caller should propagate (cmd/gpk/main.go does os.Exit on the result).
func Dispatch(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "error: no subcommand specified")
		return ExitErr
	}
	name := args[0]
	fn, ok := subcommands[name]
	if !ok {
		fmt.Fprintf(stderr, "error: unknown subcommand %q\n", name)
		return ExitErr
	}
	return fn(args[1:], mgrs, version, stdout, stderr)
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

