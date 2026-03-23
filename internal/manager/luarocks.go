package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Luarocks struct{}

func (l *Luarocks) Name() model.Source { return model.SourceLuarocks }

func (l *Luarocks) Available() bool { return commandExists("luarocks") }

func (l *Luarocks) Scan() ([]model.Package, error) {
	// --porcelain gives tab-separated: name\tversion\tstatus\tpath
	out, err := exec.Command("luarocks", "list", "--porcelain").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue
		}
		// Only include installed rocks
		if fields[2] != "installed" {
			continue
		}

		pkgs = append(pkgs, model.Package{
			Name:        fields[0],
			Version:     fields[1],
			Source:      model.SourceLuarocks,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (l *Luarocks) CheckUpdates(pkgs []model.Package) map[string]string {
	// --porcelain gives tab-separated: name\tinstalled\tlatest\trepo
	out, err := exec.Command("luarocks", "list", "--outdated", "--porcelain").Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 3 {
			updates[fields[0]] = fields[2]
		}
	}
	return updates
}

func (l *Luarocks) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("luarocks", "show", pkg.Name).Output()
		if err != nil {
			continue
		}
		var pkgDeps []string
		inDeps := false
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "Dependencies:" {
				inDeps = true
				continue
			}
			if inDeps {
				if !strings.HasPrefix(line, "   ") && !strings.HasPrefix(line, "\t") {
					break
				}
				fields := strings.Fields(strings.TrimSpace(line))
				if len(fields) >= 1 && fields[0] != "" {
					pkgDeps = append(pkgDeps, fields[0])
				}
			}
		}
		deps[pkg.Name] = pkgDeps
	}
	return deps
}

func (l *Luarocks) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("luarocks", "show", pkg.Name).Output()
		if err != nil {
			continue
		}
		// First line: "name version - description"
		line := strings.SplitN(string(out), "\n", 2)[0]
		if dashIdx := strings.Index(line, " - "); dashIdx >= 0 {
			desc := strings.TrimSpace(line[dashIdx+3:])
			if desc != "" {
				descs[pkg.Name] = desc
			}
		}
	}
	return descs
}

func (l *Luarocks) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("luarocks", "upgrade", name)
}
