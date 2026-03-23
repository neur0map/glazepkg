package manager

import "os/exec"

// Upgrader is implemented by managers that can upgrade a single package.
type Upgrader interface {
	// UpgradeCmd returns the command to upgrade a single package.
	// The caller is responsible for executing it (typically via tea.ExecProcess
	// so the user sees output and can interact with prompts like sudo).
	UpgradeCmd(name string) *exec.Cmd
}

// PreUpgrader is an optional interface implemented by managers that need to
// run cleanup or preparation steps synchronously before the upgrade command
// executes.  The UI layer calls PrepareUpgrade inside the same goroutine that
// will subsequently run the command, so the preparation is guaranteed to
// complete before the command starts.
//
// Implementations must treat all errors as non-fatal: if cleanup is not
// possible (e.g. permission denied on the artifact being cleaned), return nil
// and allow the upgrade command to surface the real error to the user.
type PreUpgrader interface {
	PrepareUpgrade(name string) error
}
