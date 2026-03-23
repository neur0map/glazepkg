package manager

import "os/exec"

// Upgrader is implemented by managers that can upgrade a single package.
type Upgrader interface {
	// UpgradeCmd returns the command to upgrade a single package.
	// The caller is responsible for executing it (typically via tea.ExecProcess
	// so the user sees output and can interact with prompts like sudo).
	UpgradeCmd(name string) *exec.Cmd
}
