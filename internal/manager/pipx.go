package manager

import (
	"encoding/json"
	"os/exec"
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

func (p *Pipx) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("pipx", "uninstall", name)
}
