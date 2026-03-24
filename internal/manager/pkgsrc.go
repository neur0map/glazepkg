package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Pkgsrc struct{}

func (p *Pkgsrc) Name() model.Source { return model.SourcePkgsrc }

func (p *Pkgsrc) Available() bool { return commandExists("pkg_info") }

func (p *Pkgsrc) Scan() ([]model.Package, error) {
	out, err := exec.Command("pkg_info").Output()
	if err != nil {
		return nil, err
	}

	// Output format: "name-version    description"
	// The LAST hyphen separates name from version.
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		nameVer := fields[0]
		name, version := SplitPkgsrcNameVersion(nameVer)
		if name == "" {
			continue
		}

		desc := ""
		if len(fields) > 1 {
			desc = strings.Join(fields[1:], " ")
		}

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Description: desc,
			Source:      model.SourcePkgsrc,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (p *Pkgsrc) RemoveCmd(name string) *exec.Cmd {
	return privilegedCmd("pkg_delete", name)
}

func (p *Pkgsrc) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		// -qc gives the one-line comment without headers
		out, err := exec.Command("pkg_info", "-qc", pkg.Name).Output()
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

func (p *Pkgsrc) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		nameVer := pkg.Name
		if pkg.Version != "" {
			nameVer = pkg.Name + "-" + pkg.Version
		}
		out, err := exec.Command("pkg_info", "-qN", nameVer).Output()
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
			name, _ := SplitPkgsrcNameVersion(line)
			if name != "" {
				pkgDeps = append(pkgDeps, name)
			}
		}
		deps[pkg.Name] = pkgDeps
	}
	return deps
}

// SplitPkgsrcNameVersion splits "name-version" by the last hyphen.
func SplitPkgsrcNameVersion(s string) (string, string) {
	idx := strings.LastIndex(s, "-")
	if idx <= 0 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
}
