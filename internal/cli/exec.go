package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

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

// invalidateAfterWrite updates cached state for a manager after install,
// remove, or upgrade. Always called after a successful subprocess run.
//
// Currently updates: scan cache (added packages upserted, removed packages
// dropped); update cache entries for those packages are invalidated.
//
// Safe to call concurrently; cache files are rewritten atomically.
func invalidateAfterWrite(mgr manager.Manager, added, removed []model.Package) {
	// Scan cache: surgically upsert installed packages and drop removed
	// ones so the next gpk list keeps its warm cache. Call sites whose
	// effect on installed packages is ambiguous (autoremove, history undo)
	// pass neither; those still delete the whole file so the next list
	// does a fresh scan. If the rewrite fails, delete too: freshness
	// matters more than performance on a write operation.
	if len(added) == 0 && len(removed) == 0 {
		manager.DeleteScanCache()
	} else {
		removeKeys := make([]string, 0, len(removed))
		for _, p := range removed {
			removeKeys = append(removeKeys, p.Key())
		}
		if err := manager.UpdateScanCache(added, removeKeys); err != nil {
			manager.DeleteScanCache()
		}
	}

	// Update cache: invalidate keys for this manager's packages.
	cache := manager.NewUpdateCache()
	var keys []string
	for _, p := range added {
		keys = append(keys, p.Key())
	}
	for _, p := range removed {
		keys = append(keys, p.Key())
	}
	cache.Invalidate(keys)
}
