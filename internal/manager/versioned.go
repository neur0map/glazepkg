package manager

import "os/exec"

// VersionedInstaller is implemented by managers that can install a specific
// version of a package. gpk uses it for `gpk install name@version` and for
// downgrades. Managers that don't implement it reject pinned installs.
type VersionedInstaller interface {
	InstallVersionCmd(name, version string) *exec.Cmd
}

func (p *Pip) InstallVersionCmd(name, version string) *exec.Cmd {
	return exec.Command("pip", "install", name+"=="+version)
}

func (p *Pipx) InstallVersionCmd(name, version string) *exec.Cmd {
	return exec.Command("pipx", "install", name+"=="+version)
}

func (u *Uv) InstallVersionCmd(name, version string) *exec.Cmd {
	return exec.Command("uv", "tool", "install", name+"=="+version)
}

func (n *Npm) InstallVersionCmd(name, version string) *exec.Cmd {
	return exec.Command("npm", "install", "-g", name+"@"+version)
}

func (n *Pnpm) InstallVersionCmd(name, version string) *exec.Cmd {
	return exec.Command("pnpm", "add", "-g", name+"@"+version)
}

func (b *Bun) InstallVersionCmd(name, version string) *exec.Cmd {
	return exec.Command("bun", "add", "-g", name+"@"+version)
}

func (g *Go) InstallVersionCmd(name, version string) *exec.Cmd {
	return exec.Command("go", "install", name+"@"+version)
}

func (g *Gem) InstallVersionCmd(name, version string) *exec.Cmd {
	return exec.Command("gem", "install", name, "-v", version)
}

func (c *Composer) InstallVersionCmd(name, version string) *exec.Cmd {
	return exec.Command("composer", "global", "require", name+":"+version)
}

func (c *Conda) InstallVersionCmd(name, version string) *exec.Cmd {
	return exec.Command(c.condaCmd(), "install", "--yes", name+"="+version)
}

func (a *Apt) InstallVersionCmd(name, version string) *exec.Cmd {
	return privilegedCmd("apt-get", "install", "-y", name+"="+version)
}

func (c *Cargo) InstallVersionCmd(name, version string) *exec.Cmd {
	return exec.Command("cargo", "install", name, "--version", version)
}
