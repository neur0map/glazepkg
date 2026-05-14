package manager

import "os/exec"

// NonInteractiveInstaller is implemented by managers that have a way to
// suppress their interactive y/N prompt during install (e.g., pacman's
// --noconfirm, apt's -y). When the gpk caller passes --yes, gpk uses this
// variant if available; otherwise it falls back to InstallCmd and the user
// must answer the manager's own prompt.
type NonInteractiveInstaller interface {
	InstallCmdYes(name string) *exec.Cmd
}

// NonInteractiveUpgrader: same as NonInteractiveInstaller but for upgrade.
type NonInteractiveUpgrader interface {
	UpgradeCmdYes(name string) *exec.Cmd
}

// NonInteractiveRemover: same shape, for remove.
type NonInteractiveRemover interface {
	RemoveCmdYes(name string) *exec.Cmd
}

// NonInteractiveDeepRemover: same shape, for remove --with-deps.
type NonInteractiveDeepRemover interface {
	RemoveCmdWithDepsYes(name string) *exec.Cmd
}
