package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

// confirmAction prints prompt to out, reads one line from in, and returns
// true iff the user typed y/yes (case-insensitive). A line that's just a
// newline, EOF, or anything else returns false (default-no).
//
// If in is nil, returns false — defensive against handlers that forget to
// thread stdin through. Callers should always check the --yes flag first
// and skip confirmAction entirely when set.
func confirmAction(prompt string, in io.Reader, out io.Writer) bool {
	if in == nil {
		return false
	}
	fmt.Fprint(out, prompt)
	r := bufio.NewReader(in)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

// headlessExec runs cmd with the parent process's stdin/stdout/stderr so
// interactive prompts (sudo password, pacman confirmations) reach the
// user's terminal. If cmd is a sudo wrapper using "-S" (read password from
// stdin — a TUI-era convention), the "-S" is stripped so sudo uses its
// normal tty prompt instead.
//
// This is the only place exec.Cmd is run in the cli package. All write
// subcommands call this, never exec.Cmd.Run directly.
func headlessExec(cmd *exec.Cmd) error {
	if cmd == nil {
		return fmt.Errorf("nil command")
	}
	cmd = stripSudoStdinFlag(cmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// stripSudoStdinFlag returns a copy of cmd with the "-S" argument removed
// when cmd is invoking sudo. Leaves non-sudo commands untouched. The "-S"
// flag was added by manager.privilegedCmd to support the TUI's password
// modal; in headless mode we want sudo to prompt on the user's tty.
func stripSudoStdinFlag(cmd *exec.Cmd) *exec.Cmd {
	if len(cmd.Args) < 2 || cmd.Args[0] != "sudo" || cmd.Args[1] != "-S" {
		return cmd
	}
	newArgs := append([]string{"sudo"}, cmd.Args[2:]...)
	newCmd := exec.Command(newArgs[0], newArgs[1:]...)
	newCmd.Dir = cmd.Dir
	newCmd.Env = cmd.Env
	return newCmd
}

// resolveInstallManager picks the manager to use for `gpk install <name>`.
//
// If the user passed --manager (via the filtered set being narrower than
// the full set), and exactly one of those managers reports the package
// available via Search, that manager wins.
//
// If the user did NOT pass --manager (filtered == manager.All()), and the
// package is available via Search in exactly one manager, that one wins.
//
// If the package is available in multiple managers, returns an error
// suitable for exit-3: "available in N managers (a, b, c), pick with
// --manager". Callers detect this via errors.Is(err, ErrAmbiguous).
//
// If no manager has the package available, returns ErrNotFound.
//
// The "available" check uses each manager's Search method (managers that
// don't implement Searcher are skipped, so they can't be auto-picked).
func resolveInstallManager(name string, filtered []manager.Manager) (manager.Manager, error) {
	var candidates []manager.Manager
	for _, m := range filtered {
		if !m.Available() {
			continue
		}
		s, ok := m.(manager.Searcher)
		if !ok {
			continue
		}
		results, err := s.Search(name)
		if err != nil {
			continue // swallow per-manager errors; we just won't auto-pick this one
		}
		for _, p := range results {
			if p.Name == name {
				candidates = append(candidates, m)
				break
			}
		}
	}

	switch len(candidates) {
	case 0:
		return nil, fmt.Errorf("%w: %q not available in any searched manager", ErrNotFound, name)
	case 1:
		return candidates[0], nil
	default:
		var names []string
		for _, m := range candidates {
			names = append(names, string(m.Name()))
		}
		return nil, fmt.Errorf("%w: %q available in %d managers (%s)",
			ErrAmbiguous, name, len(candidates), strings.Join(names, ", "))
	}
}

// Sentinel errors returned by resolveInstallManager. Tests use errors.Is.
var (
	ErrAmbiguous = fmt.Errorf("ambiguous package")
	ErrNotFound  = fmt.Errorf("package not found")
)

// invalidateAfterWrite clears cached state for a manager after install,
// remove, or upgrade. Always called after a successful subprocess run.
//
// Currently invalidates: scan cache (rewritten on next gpk list); update
// cache entries for that manager's packages.
//
// Safe to call concurrently; cache files are rewritten atomically.
func invalidateAfterWrite(mgr manager.Manager, pkgs []model.Package) {
	// Scan cache: nuke it entirely so the next gpk list does a fresh scan.
	// We can't surgically remove just one manager's packages without first
	// reading and rewriting the file, and this is a write operation —
	// freshness matters more than performance.
	_ = os.Remove(scanCachePath())

	// Update cache: invalidate keys for this manager's packages.
	cache := manager.NewUpdateCache()
	var keys []string
	for _, p := range pkgs {
		keys = append(keys, p.Key())
	}
	cache.Invalidate(keys)
}

// scanCachePath duplicates the logic in internal/manager/scancache.go so
// we don't expose the path constant publicly. If manager ever exports a
// "DeleteScanCache" helper, switch to that.
func scanCachePath() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "glazepkg", "cache", "scan.json")
}
