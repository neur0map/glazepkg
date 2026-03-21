package manager

import (
	"os/exec"

	"github.com/neur0map/glazepkg/internal/model"
)

// Manager scans a package manager and returns its installed packages.
type Manager interface {
	Name() model.Source
	Available() bool
	Scan() ([]model.Package, error)
}

// All returns every known manager.
func All() []Manager {
	return []Manager{
		&Brew{},
		&Pacman{},
		&AUR{},
		&Apt{},
		&Dnf{},
		&Snap{},
		&Pip{},
		&Pipx{},
		&Cargo{},
		&Go{},
		&Npm{},
		&Pnpm{},
		&Bun{},
		&Flatpak{},
		&MacPorts{},
		&Pkgsrc{},
		&Opam{},
		&Gem{},
		&FreeBSDPkg{},
		&Composer{},
		&Mas{},
		&Apk{},
		&Nix{},
		&Conda{},
		&Luarocks{},
		&Xbps{},
		&Portage{},
		&Guix{},
		&Winget{},
		&Chocolatey{},
		&Nuget{},
		&PowerShell{},
		&WindowsUpdates{},
		&Scoop{},
	}
}

// commandExists checks if a binary is in PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
