package manager

import (
	"errors"
	"os/exec"

	"github.com/neur0map/glazepkg/internal/model"
)

// ErrUpgradeNotSupported is returned when a manager cannot upgrade a single package.
var ErrUpgradeNotSupported = errors.New("This package manager does not support upgrading a single package.")

// Manager scans a package manager and returns its installed packages.
type Manager interface {
	Name() model.Source
	Available() bool
	Scan() ([]model.Package, error)
	UpgradePackage(name string) error
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

// BySource returns the manager responsible for the provided source, if any.
func BySource(source model.Source) Manager {
	for _, mgr := range All() {
		if mgr.Name() == source {
			return mgr
		}
	}
	return nil
}

// commandExists checks if a binary is in PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
