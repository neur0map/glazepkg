package manager

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Portage struct{}

func (p *Portage) Name() model.Source { return model.SourcePortage }

func (p *Portage) Available() bool {
	// qlist (from portage-utils) is fast and always available on Gentoo
	return commandExists("qlist")
}

func (p *Portage) Scan() ([]model.Package, error) {
	// Build set of explicitly installed packages from /var/lib/portage/world
	// World file contains "category/name" per line (no version).
	worldPkgs := make(map[string]bool)
	if data, err := os.ReadFile("/var/lib/portage/world"); err == nil {
		sc := bufio.NewScanner(strings.NewReader(string(data)))
		for sc.Scan() {
			if atom := strings.TrimSpace(sc.Text()); atom != "" {
				worldPkgs[atom] = true
			}
		}
	}

	// qlist -Iv outputs "category/name-version" per line
	out, err := exec.Command("qlist", "-Iv").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		name, version := splitPortageCPV(line)
		if name == "" {
			continue
		}
		// Skip auto-installed dependencies when we have the world set
		if len(worldPkgs) > 0 && !worldPkgs[name] {
			continue
		}

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourcePortage,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (p *Portage) RemoveCmd(name string) *exec.Cmd {
	return privilegedCmd("emerge", "--unmerge", name)
}

func (p *Portage) CheckUpdates(pkgs []model.Package) map[string]string {
	// emerge -puDN @world shows packages that would be updated
	out, err := exec.Command("emerge", "-puDN", "@world").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	// Lines like: "[ebuild     U  ] cat/pkg-1.2.3 [1.2.2] ..."
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "[ebuild") || !strings.Contains(line, "U") {
			continue
		}
		// Find the closing ] of [ebuild ... ]
		closeIdx := strings.Index(line, "] ")
		if closeIdx < 0 {
			continue
		}
		rest := strings.TrimSpace(line[closeIdx+2:])
		// rest = "cat/pkg-newver [oldver] USE=..."
		fields := strings.Fields(rest)
		if len(fields) < 1 {
			continue
		}
		// Strip ::repo suffix
		cpv := fields[0]
		if colonIdx := strings.Index(cpv, "::"); colonIdx >= 0 {
			cpv = cpv[:colonIdx]
		}
		name, version := splitPortageCPV(cpv)
		if name != "" && version != "" {
			updates[name] = version
		}
	}
	return updates
}

func (p *Portage) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		// emerge -s searches by name, parse Description: line
		out, err := exec.Command("emerge", "-s", pkg.Name).Output()
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "Description:") {
				desc := strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
				if desc != "" {
					descs[pkg.Name] = desc
				}
				break
			}
		}
	}
	return descs
}

func (p *Portage) ListDependencies(pkgs []model.Package) map[string][]string {
	if !commandExists("equery") {
		return nil
	}
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("equery", "-q", "depgraph", "-MUl", pkg.Name).Output()
		if err != nil {
			continue
		}
		var pkgDeps []string
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "[") {
				continue
			}
			// Lines are "cat/dep-version" — extract just the name
			name, _ := splitPortageCPV(line)
			if name != "" && name != pkg.Name {
				pkgDeps = append(pkgDeps, name)
			}
		}
		deps[pkg.Name] = pkgDeps
	}
	return deps
}

func (p *Portage) UpgradeCmd(name string) *exec.Cmd {
	return privilegedCmd("emerge", "--update", name)
}

func (p *Portage) Search(query string) ([]model.Package, error) {
	out, err := exec.Command("emerge", "-s", query).Output()
	if err != nil && len(out) == 0 {
		return nil, nil
	}
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var name, desc string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "* ") {
			name = strings.TrimPrefix(line, "* ")
		}
		if strings.HasPrefix(strings.TrimSpace(line), "Description:") {
			desc = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "Description:"))
			if name != "" {
				pkgs = append(pkgs, model.Package{Name: name, Source: model.SourcePortage, Description: desc})
				name = ""
				desc = ""
			}
		}
	}
	return pkgs, nil
}

func (p *Portage) InstallCmd(name string) *exec.Cmd {
	return privilegedCmd("emerge", name)
}

// splitPortageCPV splits "category/name-version" into ("category/name", "version").
// The version starts at the last hyphen before a digit.
func splitPortageCPV(cpv string) (string, string) {
	for i := len(cpv) - 1; i > 0; i-- {
		if cpv[i] == '-' && i+1 < len(cpv) && cpv[i+1] >= '0' && cpv[i+1] <= '9' {
			return cpv[:i], cpv[i+1:]
		}
	}
	return cpv, ""
}
