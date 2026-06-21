package manager

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// VersionLister is implemented by managers that can enumerate the versions of
// a package available to install, newest first. gpk uses it for the
// `gpk downgrade <name>` version picker.
type VersionLister interface {
	Versions(name string) ([]string, error)
}

// Downgrader is implemented by managers whose downgrade path differs from a
// normal versioned install (notably pacman, which reinstalls from its local
// package cache). Managers without it fall back to VersionedInstaller.
type Downgrader interface {
	DowngradeCmd(name, version string) *exec.Cmd
}

const pacmanCacheDir = "/var/cache/pacman/pkg"

func (p *Pacman) Versions(name string) ([]string, error) {
	matches, _ := filepath.Glob(filepath.Join(pacmanCacheDir, name+"-*.pkg.tar.*"))
	seen := make(map[string]bool)
	var versions []string
	for _, m := range matches {
		base := filepath.Base(m)
		if strings.HasSuffix(base, ".sig") {
			continue
		}
		if v := pacmanCacheVersion(base, name); v != "" && !seen[v] {
			seen[v] = true
			versions = append(versions, v)
		}
	}
	return versions, nil
}

func (p *Pacman) DowngradeCmd(name, version string) *exec.Cmd {
	file := pacmanCacheFile(name, version)
	if file == "" {
		return nil
	}
	return privilegedCmd("pacman", "-U", file)
}

// pacmanCacheVersion extracts "pkgver-pkgrel" from a cache filename of the form
// "<name>-<ver>-<rel>-<arch>.pkg.tar.*". It rejects filenames where the version
// field doesn't start with a digit, which filters out longer-named packages
// that share the prefix (e.g. foo-doc vs foo).
func pacmanCacheVersion(base, name string) string {
	rest, ok := strings.CutPrefix(base, name+"-")
	if !ok {
		return ""
	}
	if i := strings.Index(rest, ".pkg.tar"); i >= 0 {
		rest = rest[:i]
	}
	parts := strings.Split(rest, "-")
	if len(parts) < 3 {
		return ""
	}
	ver := strings.Join(parts[:len(parts)-2], "-")
	rel := parts[len(parts)-2]
	if ver == "" || ver[0] < '0' || ver[0] > '9' {
		return ""
	}
	return ver + "-" + rel
}

func pacmanCacheFile(name, version string) string {
	matches, _ := filepath.Glob(filepath.Join(pacmanCacheDir, name+"-"+version+"-*.pkg.tar.*"))
	for _, m := range matches {
		if strings.HasSuffix(m, ".sig") {
			continue
		}
		if _, err := os.Stat(m); err == nil {
			return m
		}
	}
	return ""
}

func (p *Pip) Versions(name string) ([]string, error) {
	out, err := exec.Command("pip", "index", "versions", name).Output()
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "Available versions:"); ok {
			var versions []string
			for _, v := range strings.Split(rest, ",") {
				if v = strings.TrimSpace(v); v != "" {
					versions = append(versions, v)
				}
			}
			return versions, nil
		}
	}
	return nil, nil
}

func (n *Npm) Versions(name string) ([]string, error) {
	out, err := exec.Command("npm", "view", name, "versions", "--json").Output()
	if err != nil {
		return nil, err
	}
	var versions []string
	if err := json.Unmarshal(out, &versions); err != nil {
		return nil, err
	}
	// npm lists oldest first; present newest first.
	for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
		versions[i], versions[j] = versions[j], versions[i]
	}
	return versions, nil
}

func (a *Apt) Versions(name string) ([]string, error) {
	out, err := exec.Command("apt-cache", "madison", name).Output()
	if err != nil {
		return nil, err
	}
	var versions []string
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Split(line, "|")
		if len(fields) < 2 {
			continue
		}
		if v := strings.TrimSpace(fields[1]); v != "" && !seen[v] {
			seen[v] = true
			versions = append(versions, v)
		}
	}
	return versions, nil
}

func (g *Gem) Versions(name string) ([]string, error) {
	out, err := exec.Command("gem", "list", "-ra", "-e", name).Output()
	if err != nil {
		return nil, err
	}
	open := strings.Index(string(out), "(")
	closeIdx := strings.LastIndex(string(out), ")")
	if open < 0 || closeIdx <= open {
		return nil, nil
	}
	var versions []string
	for _, v := range strings.Split(string(out)[open+1:closeIdx], ",") {
		if v = strings.TrimSpace(v); v != "" {
			versions = append(versions, v)
		}
	}
	return versions, nil
}
