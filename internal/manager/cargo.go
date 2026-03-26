package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Cargo struct{}

func (c *Cargo) Name() model.Source { return model.SourceCargo }

func (c *Cargo) Available() bool { return commandExists("cargo") }

func (c *Cargo) Scan() ([]model.Package, error) {
	out, err := exec.Command("cargo", "install", "--list").Output()
	if err != nil {
		return nil, err
	}

	// Output format:
	// package-name v1.2.3:
	//     binary1
	//     binary2
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, " ") || line == "" {
			continue
		}
		// "package-name v1.2.3:" or "package-name v1.2.3 (path):"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		version := strings.TrimPrefix(parts[1], "v")
		version = strings.TrimSuffix(version, ":")

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceCargo,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (c *Cargo) Search(query string) ([]model.Package, error) {
	out, err := exec.Command("cargo", "search", query).Output()
	if err != nil || len(out) == 0 {
		return nil, nil
	}
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "...") {
			continue
		}
		// Format: 'name = "version"    # description'
		eqIdx := strings.Index(line, " = ")
		if eqIdx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:eqIdx])
		rest := line[eqIdx+3:]
		version := ""
		if q1 := strings.Index(rest, "\""); q1 >= 0 {
			if q2 := strings.Index(rest[q1+1:], "\""); q2 >= 0 {
				version = rest[q1+1 : q1+1+q2]
			}
		}
		desc := ""
		if hashIdx := strings.Index(rest, "# "); hashIdx >= 0 {
			desc = strings.TrimSpace(rest[hashIdx+2:])
		}
		pkgs = append(pkgs, model.Package{Name: name, Version: version, Source: model.SourceCargo, Description: desc})
	}
	return pkgs, nil
}

func (c *Cargo) InstallCmd(name string) *exec.Cmd {
	return exec.Command("cargo", "install", name)
}

func (c *Cargo) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("cargo", "install", name)
}

func (c *Cargo) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("cargo", "info", pkg.Name).Output()
		if err != nil {
			continue
		}
		// cargo info output contains a line like:
		// description: Some description here
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "description:") {
				desc := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				if desc != "" {
					descs[pkg.Name] = desc
				}
				break
			}
		}
	}
	return descs
}

func (c *Cargo) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("cargo", "uninstall", name)
}
