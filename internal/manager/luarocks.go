package manager

import (
	"bufio"
	"os"
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

	home, _ := os.UserHomeDir()
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
		path := ""
		if len(fields) >= 4 {
			path = fields[3]
		}
		pkgs = append(pkgs, model.Package{
			Name:        fields[0],
			Version:     fields[1],
			Source:      model.SourceLuarocks,
			Scope:       luarocksScope(path, home),
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

// luarocksScope labels a rock by its install tree: "user" when under the home
// directory (the local tree), "system" otherwise. Empty when path is unknown.
func luarocksScope(path, home string) string {
	if path == "" {
		return ""
	}
	if home != "" && strings.HasPrefix(path, home) {
		return "user"
	}
	return "system"
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

func (l *Luarocks) Search(query string) ([]model.Package, error) {
	out, err := exec.Command("luarocks", "search", query, "--porcelain").Output()
	if err != nil || len(out) == 0 {
		return nil, nil
	}
	var pkgs []model.Package
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		if seen[fields[0]] {
			continue
		}
		seen[fields[0]] = true
		pkgs = append(pkgs, model.Package{Name: fields[0], Version: fields[1], Source: model.SourceLuarocks})
	}
	return pkgs, nil
}

func (l *Luarocks) InstallCmd(name string) *exec.Cmd {
	return exec.Command("luarocks", "install", "--local", name)
}

func (l *Luarocks) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("luarocks", "upgrade", "--local", name)
}

func (l *Luarocks) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("luarocks", "remove", name)
}
