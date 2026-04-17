package manager

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

// Describe reads package descriptions from the package.json that bun installs
// alongside each global package. Offline — no npm CLI or network required.
func (b *Bun) Describe(pkgs []model.Package) map[string]string {
	dir := bunGlobalNodeModulesDir()
	if dir == "" {
		return nil
	}
	descs := make(map[string]string, len(pkgs))
	for _, pkg := range pkgs {
		if desc := bunLocalDescription(dir, pkg.Name); desc != "" {
			descs[pkg.Name] = desc
		}
	}
	return descs
}

// bunGlobalNodeModulesDir returns the path to bun's global node_modules.
// It honors $BUN_INSTALL when set, otherwise falls back to ~/.bun.
func bunGlobalNodeModulesDir() string {
	root := os.Getenv("BUN_INSTALL")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		root = filepath.Join(home, ".bun")
	}
	return filepath.Join(root, "install", "global", "node_modules")
}

// bunLocalDescription reads the "description" field from the installed
// package.json under the global node_modules directory.
func bunLocalDescription(nodeModulesDir, name string) string {
	data, err := os.ReadFile(filepath.Join(nodeModulesDir, name, "package.json"))
	if err != nil {
		return ""
	}
	var meta struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return ""
	}
	return strings.TrimSpace(meta.Description)
}

func (b *Bun) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("bun", "update", "-g", name)
}

func (b *Bun) InstallCmd(name string) *exec.Cmd {
	return exec.Command("bun", "add", "-g", name)
}
