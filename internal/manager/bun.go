package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Bun struct{}

func (b *Bun) Name() model.Source { return model.SourceBun }

func (b *Bun) Available() bool { return commandExists("bun") }

func (b *Bun) Scan() ([]model.Package, error) {
	out, err := exec.Command("bun", "pm", "ls", "-g").Output()
	if err != nil {
		return nil, err
	}

	// Output format varies, typically:
	// /path/to/global node_modules
	// ├── package@version
	// └── package@version
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Strip tree characters
		line = strings.TrimLeft(line, "├└─│ ")
		if line == "" || !strings.Contains(line, "@") {
			continue
		}
		// Split "package@version"
		idx := strings.LastIndex(line, "@")
		if idx <= 0 {
			continue
		}
		name := line[:idx]
		version := line[idx+1:]

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceBun,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (b *Bun) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("bun", "remove", "-g", name)
}

func (b *Bun) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("npm", "info", pkg.Name, "description").Output()
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

func (b *Bun) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("bun", "update", "-g", name)
}

func (b *Bun) InstallCmd(name string) *exec.Cmd {
	return exec.Command("bun", "add", "-g", name)
}
