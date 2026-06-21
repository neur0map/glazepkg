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

// userEnv is the environment as gpk started, captured before UseStableLocale
// forces a C locale for parsing. Interactive commands run with it so prompts
// reach the user in their own language.
var userEnv = os.Environ()

// UseStableLocale forces a C locale on the process so the tools gpk parses emit
// stable, English field names regardless of the system language. It's applied
// only on the CLI path; interactive commands restore userEnv via headlessExec.
func UseStableLocale() {
	os.Setenv("LC_ALL", "C")
	os.Unsetenv("LANGUAGE")
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
	if cmd.Env == nil {
		cmd.Env = userEnv
	}
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
