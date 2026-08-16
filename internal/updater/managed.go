package updater

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Install describes who owns the running binary.
type Install struct {
	Manager string // empty when nothing owns it
	Package string
	Upgrade string // command to run instead of `gpk update`; empty when ambiguous
}

// Managed reports the package manager that installed the running binary.
// A zero Install means gpk owns its own file (release download, go install,
// a local build) and may replace it.
//
// Replacing a file a package manager owns desyncs that manager's database: brew
// reverts it on the next upgrade, pacman flags it as modified, and the Nix store
// is read-only. So gpk asks who owns it before touching anything.
func Managed() Install {
	path, err := os.Executable()
	if err != nil {
		return Install{}
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	if in := managedByPath(path); in.Manager != "" {
		return in
	}
	return managedByPackageDB(path)
}

// managedByPath covers the managers whose install layout is unmistakable, with
// no subprocess needed. Each marker has to be a real path segment with the
// manager's own subtree below it, so a user directory that happens to be named
// Cellar or scoop is not mistaken for an install root.
func managedByPath(path string) Install {
	slashed := filepath.ToSlash(path)
	segs := strings.Split(slashed, "/")
	switch {
	case strings.HasPrefix(slashed, "/nix/store/") && len(segs) > 4:
		return Install{Manager: "nix", Package: "gpk", Upgrade: "nix profile upgrade gpk"}
	// Cellar/<formula>/<version>/bin/gpk under any prefix: /opt/homebrew,
	// /usr/local or Linuxbrew.
	case subtreeAt(segs, "Cellar", "", 4):
		return Install{Manager: "homebrew", Package: "gpk", Upgrade: "brew upgrade gpk"}
	case subtreeAt(segs, "scoop", "apps", 3):
		return Install{Manager: "scoop", Package: "gpk", Upgrade: "scoop update gpk"}
	case subtreeAt(segs, "chocolatey", "lib", 3):
		return Install{Manager: "chocolatey", Package: "gpk", Upgrade: "choco upgrade gpk"}
	}
	return Install{}
}

// subtreeAt reports whether segs contains marker as a segment, optionally
// followed by sub, with at least depth segments after marker. Matching folds
// case because the Windows layouts are case-insensitive.
func subtreeAt(segs []string, marker, sub string, depth int) bool {
	for i, s := range segs {
		if !strings.EqualFold(s, marker) {
			continue
		}
		if len(segs)-i <= depth {
			continue
		}
		if sub != "" && !strings.EqualFold(segs[i+1], sub) {
			continue
		}
		return true
	}
	return false
}

// managedByPackageDB asks the system package database who owns path. No upgrade
// command is offered: the database knows the package but not whether it came
// from a distro repo or a helper like yay, and a wrong command here is worse
// than none.
func managedByPackageDB(path string) Install {
	probes := []struct {
		bin, manager string
		args         []string
		owner        func(string) string
	}{
		{"pacman", "pacman", []string{"-Qo"}, pacmanOwner},
		{"dpkg", "dpkg", []string{"-S"}, dpkgOwner},
		{"rpm", "rpm", []string{"-qf", "--queryformat", "%{NAME}"}, strings.TrimSpace},
	}
	for _, p := range probes {
		if _, err := exec.LookPath(p.bin); err != nil {
			continue
		}
		out, err := exec.Command(p.bin, append(p.args, path)...).Output()
		if err != nil {
			return Install{}
		}
		if name := p.owner(string(out)); name != "" {
			return Install{Manager: p.manager, Package: name}
		}
		return Install{}
	}
	return Install{}
}

// pacmanOwner reads "/usr/bin/gpk is owned by gpk-bin 0.6.6".
func pacmanOwner(out string) string {
	_, rest, ok := strings.Cut(strings.TrimSpace(out), " is owned by ")
	if !ok {
		return ""
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// dpkgOwner reads "gpk: /usr/bin/gpk", where the package field may list several.
func dpkgOwner(out string) string {
	name, _, ok := strings.Cut(strings.TrimSpace(out), ": ")
	if !ok {
		return ""
	}
	if first, _, multi := strings.Cut(name, ","); multi {
		name = first
	}
	return strings.TrimSpace(name)
}

// UpgradeHint returns the sentence to show a user who wants a newer gpk.
func (i Install) UpgradeHint() string {
	switch {
	case i.Upgrade != "":
		return "run `" + i.Upgrade + "`"
	case i.Manager != "":
		return "upgrade " + i.Package + " with " + i.Manager
	default:
		return "run `gpk update`"
	}
}
