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
