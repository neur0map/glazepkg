package manager

import "os/exec"

// CacheCleaner is implemented by managers that can clear downloaded package
// caches. `gpk clean` / `-Sc` runs it for every available manager. all selects
// the more aggressive "remove everything" form where the manager has one.
type CacheCleaner interface {
	CleanCacheCmd(all, assumeYes bool) *exec.Cmd
}

func (p *Pacman) CleanCacheCmd(all, yes bool) *exec.Cmd {
	args := []string{"-Sc"}
	if all {
		args = []string{"-Scc"}
	}
	if yes {
		args = append(args, "--noconfirm")
	}
	return privilegedCmd("pacman", args...)
}

func (a *Apt) CleanCacheCmd(all, _ bool) *exec.Cmd {
	if all {
		return privilegedCmd("apt-get", "clean")
	}
	return privilegedCmd("apt-get", "autoclean")
}

func (d *Dnf) CleanCacheCmd(all, _ bool) *exec.Cmd {
	if all {
		return privilegedCmd("dnf", "clean", "all")
	}
	return privilegedCmd("dnf", "clean", "packages")
}

func (b *Brew) CleanCacheCmd(all, _ bool) *exec.Cmd {
	if all {
		return exec.Command("brew", "cleanup", "--prune=all")
	}
	return exec.Command("brew", "cleanup")
}

func (n *Npm) CleanCacheCmd(_, _ bool) *exec.Cmd {
	return exec.Command("npm", "cache", "clean", "--force")
}

func (n *Pnpm) CleanCacheCmd(_, _ bool) *exec.Cmd {
	return exec.Command("pnpm", "store", "prune")
}

func (b *Bun) CleanCacheCmd(_, _ bool) *exec.Cmd {
	return exec.Command("bun", "pm", "cache", "rm")
}

func (p *Pip) CleanCacheCmd(_, _ bool) *exec.Cmd {
	return exec.Command("pip", "cache", "purge")
}

func (u *Uv) CleanCacheCmd(_, _ bool) *exec.Cmd {
	return exec.Command("uv", "cache", "clean")
}

func (c *Conda) CleanCacheCmd(all, _ bool) *exec.Cmd {
	if all {
		return exec.Command(c.condaCmd(), "clean", "--all", "--yes")
	}
	return exec.Command(c.condaCmd(), "clean", "--packages", "--tarballs", "--yes")
}

func (g *Gem) CleanCacheCmd(_, _ bool) *exec.Cmd {
	return exec.Command("gem", "cleanup")
}

func (c *Composer) CleanCacheCmd(_, _ bool) *exec.Cmd {
	return exec.Command("composer", "clear-cache")
}

func (s *Scoop) CleanCacheCmd(_, _ bool) *exec.Cmd {
	return exec.Command("scoop", "cache", "rm", "*")
}

func (a *Apk) CleanCacheCmd(_, _ bool) *exec.Cmd {
	return privilegedCmd("apk", "cache", "clean")
}

func (x *Xbps) CleanCacheCmd(all, yes bool) *exec.Cmd {
	args := []string{"-O"}
	if all {
		args = []string{"-Oo"}
	}
	if yes {
		args = append(args, "-y")
	}
	return privilegedCmd("xbps-remove", args...)
}
