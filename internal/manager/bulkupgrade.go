package manager

import "os/exec"

// BulkUpgrader is implemented by managers that can upgrade every package they
// own in one command. `gpk upgrade` (no arguments) and `-Syu` run this for
// each available manager. assumeYes selects the manager's non-interactive
// form where one exists.
type BulkUpgrader interface {
	UpgradeAllCmd(assumeYes bool) *exec.Cmd
}

// IgnoringBulkUpgrader is a BulkUpgrader that can leave specific packages
// untouched during a full upgrade — used to honor gpk's holds.
type IgnoringBulkUpgrader interface {
	UpgradeAllCmdIgnoring(assumeYes bool, ignore []string) *exec.Cmd
}

func (p *Pacman) UpgradeAllCmdIgnoring(yes bool, ignore []string) *exec.Cmd {
	args := []string{"-Syu"}
	if yes {
		args = append(args, "--noconfirm")
	}
	for _, n := range ignore {
		args = append(args, "--ignore", n)
	}
	return privilegedCmd("pacman", args...)
}

func (p *Pacman) UpgradeAllCmd(yes bool) *exec.Cmd {
	if yes {
		return privilegedCmd("pacman", "-Syu", "--noconfirm")
	}
	return privilegedCmd("pacman", "-Syu")
}

func (a *AUR) UpgradeAllCmd(yes bool) *exec.Cmd {
	h := aurHelper()
	if h == "" {
		return nil
	}
	if yes {
		return exec.Command(h, "-Sua", "--noconfirm")
	}
	return exec.Command(h, "-Sua")
}

func (a *Apt) UpgradeAllCmd(yes bool) *exec.Cmd {
	if yes {
		return privilegedCmd("apt-get", "upgrade", "-y")
	}
	return privilegedCmd("apt-get", "upgrade")
}

func (d *Dnf) UpgradeAllCmd(yes bool) *exec.Cmd {
	if yes {
		return privilegedCmd("dnf", "upgrade", "-y")
	}
	return privilegedCmd("dnf", "upgrade")
}

func (b *Brew) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("brew", "upgrade")
}

func (b *BrewCask) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("brew", "upgrade", "--cask")
}

func (f *Flatpak) UpgradeAllCmd(yes bool) *exec.Cmd {
	if yes {
		return exec.Command("flatpak", "update", "-y")
	}
	return exec.Command("flatpak", "update")
}

func (s *Snap) UpgradeAllCmd(_ bool) *exec.Cmd {
	return privilegedCmd("snap", "refresh")
}

func (n *Npm) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("npm", "update", "-g")
}

func (n *Pnpm) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("pnpm", "update", "-g")
}

func (g *Gem) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("gem", "update")
}

func (p *Pipx) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("pipx", "upgrade-all")
}

func (u *Uv) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("uv", "tool", "upgrade", "--all")
}

func (c *Conda) UpgradeAllCmd(yes bool) *exec.Cmd {
	if yes {
		return exec.Command(c.condaCmd(), "update", "--all", "--yes")
	}
	return exec.Command(c.condaCmd(), "update", "--all")
}

func (m *Mas) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("mas", "upgrade")
}

func (m *MacPorts) UpgradeAllCmd(_ bool) *exec.Cmd {
	return privilegedCmd("port", "upgrade", "outdated")
}

func (x *Xbps) UpgradeAllCmd(yes bool) *exec.Cmd {
	if yes {
		return privilegedCmd("xbps-install", "-Su", "-y")
	}
	return privilegedCmd("xbps-install", "-Su")
}

func (a *Apk) UpgradeAllCmd(_ bool) *exec.Cmd {
	return privilegedCmd("apk", "upgrade")
}

func (g *Guix) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("guix", "upgrade")
}

func (s *Scoop) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("scoop", "update", "*")
}

func (c *Chocolatey) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("choco", "upgrade", "all", "--yes")
}

func (w *Winget) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("winget", "upgrade", "--all")
}

func (c *Composer) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("composer", "global", "update")
}

func (m *Mise) UpgradeAllCmd(_ bool) *exec.Cmd {
	return exec.Command("mise", "upgrade")
}
