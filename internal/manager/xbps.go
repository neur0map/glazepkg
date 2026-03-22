package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Xbps struct{}

func (x *Xbps) Name() model.Source { return model.SourceXbps }

func (x *Xbps) Available() bool { return commandExists("xbps-query") }

func (x *Xbps) Scan() ([]model.Package, error) {
	out, err := exec.Command("xbps-query", "-l").Output()
	if err != nil {
		return nil, err
	}

	// Output: "ii pkgname-version_rev   short description"
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		// fields[0] = state (ii, uu, etc), fields[1] = name-version_rev
		name, version := SplitXbpsNameVersion(fields[1])
		if name == "" {
			continue
		}
		desc := ""
		if len(fields) > 2 {
			desc = strings.Join(fields[2:], " ")
		}

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Description: desc,
			Source:      model.SourceXbps,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (x *Xbps) CheckUpdates(pkgs []model.Package) map[string]string {
	// -S sync, -u update, -n dry-run
	out, err := exec.Command("xbps-install", "-Sun").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	// Output: "pkgver action arch repo installedsize downloadsize"
	// Filter for "update" action
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		if fields[1] != "update" {
			continue
		}
		name, version := SplitXbpsNameVersion(fields[0])
		if name != "" {
			updates[name] = version
		}
	}
	return updates
}

func (x *Xbps) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("xbps-query", "-p", "short_desc", pkg.Name).Output()
		if err != nil {
			continue
		}
		desc := strings.TrimSpace(string(out))
		if desc != "" {
			descs[pkg.Name] = desc
		}
	}
	return descs
}

func (x *Xbps) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("xbps-query", "-x", pkg.Name).Output()
		if err != nil {
			continue
		}
		var pkgDeps []string
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			// Lines are "dep>=version" or "dep-version_rev"
			// Strip version constraints
			for i, c := range line {
				if c == '>' || c == '<' || c == '=' {
					line = line[:i]
					break
				}
			}
			name, _ := SplitXbpsNameVersion(line)
			if name == "" {
				name = line
			}
			if name != "" {
				pkgDeps = append(pkgDeps, name)
			}
		}
		deps[pkg.Name] = pkgDeps
	}
	return deps
}

func (x *Xbps) UpgradePackage(name string) error {
	cmd := exec.Command("xbps-install", "-S", "--yes", name)
	return runPrivilegedCommand(cmd)
}

// SplitXbpsNameVersion splits "name-version_revision" by the last hyphen
// before a digit.
func SplitXbpsNameVersion(s string) (string, string) {
	for i := len(s) - 1; i > 0; i-- {
		if s[i] == '-' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
