package cli

import (
	"io"
	"os/exec"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

// planStep is one resolved action in a write command's plan, shaped for
// `--json` so a GUI or script can preview exactly what gpk would run.
type planStep struct {
	Manager    string   `json:"manager"`
	Name       string   `json:"name"`
	Version    string   `json:"version,omitempty"`
	RequiredBy []string `json:"required_by,omitempty"`
	Command    []string `json:"command"`
}

type planEnvelope struct {
	Action string     `json:"action"`
	Steps  []planStep `json:"steps"`
}

func emitPlanJSON(w io.Writer, version, action string, steps []planStep) error {
	return writeEnvelope(w, version, planEnvelope{Action: action, Steps: steps})
}

// lookupInstalled asks every manager in mgrs that can search its whole
// installed database (manager.InstalledLooker) for name. Remove and installed
// fall back to it when a manager is named and the scan, which lists only
// explicitly installed packages, has no such name.
func lookupInstalled(mgrs []manager.Manager, name string) []model.Package {
	var out []model.Package
	for _, m := range mgrs {
		l, ok := m.(manager.InstalledLooker)
		if !ok || !m.Available() {
			continue
		}
		if p, ok := l.LookupInstalled([]string{name})[name]; ok {
			out = append(out, p)
		}
	}
	return out
}

// withRequiredBy fills pkg.RequiredBy from the manager's installed database
// when the scan didn't carry it.
func withRequiredBy(mgr manager.Manager, pkg model.Package) model.Package {
	if len(pkg.RequiredBy) > 0 {
		return pkg
	}
	if l, ok := mgr.(manager.InstalledLooker); ok {
		if p, ok := l.LookupInstalled([]string{pkg.Name})[pkg.Name]; ok {
			pkg.RequiredBy = p.RequiredBy
		}
	}
	return pkg
}

// cmdArgs returns the argv gpk would execute, rewritten for the --privilege
// mode (sudo without the TUI-era "-S", pkexec, or bare) so the JSON matches
// what actually runs headless.
func cmdArgs(cmd *exec.Cmd) []string {
	if cmd == nil {
		return nil
	}
	return displayArgs(cmd)
}
