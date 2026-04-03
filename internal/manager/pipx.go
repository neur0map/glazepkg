package manager

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Pipx struct{}

func (p *Pipx) Name() model.Source { return model.SourcePipx }

func (p *Pipx) Available() bool { return commandExists("pipx") }

func (p *Pipx) Scan() ([]model.Package, error) {
	out, err := exec.Command("pipx", "list", "--json").Output()
	if err != nil {
		return nil, err
	}

	var result struct {
		Venvs map[string]struct {
			Metadata struct {
				MainPackage struct {
					PackageVersion string `json:"package_version"`
				} `json:"main_package"`
			} `json:"metadata"`
		} `json:"venvs"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	pkgs := make([]model.Package, 0, len(result.Venvs))
	for name, venv := range result.Venvs {
		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     venv.Metadata.MainPackage.PackageVersion,
			Source:      model.SourcePipx,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (p *Pipx) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("pipx", "upgrade", name)
}

func (p *Pipx) InstallCmd(name string) *exec.Cmd {
	return exec.Command("pipx", "install", name)
}

func (p *Pipx) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("pipx", "uninstall", name)
}

// Describe reads package summaries from the dist-info METADATA files inside
// each pipx tool's virtual environment. No network calls needed.
func (p *Pipx) Describe(pkgs []model.Package) map[string]string {
	venvsDir := pipxVenvsDir()
	if venvsDir == "" {
		return nil
	}
	descs := make(map[string]string, len(pkgs))
	for _, pkg := range pkgs {
		if desc := pipxLocalSummary(venvsDir, pkg.Name); desc != "" {
			descs[pkg.Name] = desc
		}
	}
	return descs
}

// pipxVenvsDir returns the path to the pipx venvs directory.
// It uses the PIPX_HOME env var when set, otherwise falls back to the
// default of ~/.local/share/pipx.
func pipxVenvsDir() string {
	home := os.Getenv("PIPX_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		home = filepath.Join(userHome, ".local", "share", "pipx")
	}
	return filepath.Join(home, "venvs")
}

// pipxLocalSummary reads the Summary field from the installed METADATA file
// for a pipx tool venv. The layout mirrors that of uv tools.
func pipxLocalSummary(venvsDir, name string) string {
	normalized := strings.ReplaceAll(name, "-", "_")
	pattern := filepath.Join(venvsDir, name, "lib", "python*", "site-packages", normalized+"-*.dist-info", "METADATA")
	matches, err := filepath.Glob(pattern)
	if (err != nil || len(matches) == 0) && normalized != name {
		pattern = filepath.Join(venvsDir, name, "lib", "python*", "site-packages", name+"-*.dist-info", "METADATA")
		matches, _ = filepath.Glob(pattern)
	}
	if len(matches) == 0 {
		return ""
	}
	return parseMetadataSummary(matches[0])
}
