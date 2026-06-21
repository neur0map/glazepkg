package cli

import (
	"io"
	"os/exec"
)

// planStep is one resolved action in a write command's plan, shaped for
// `--json` so a GUI or script can preview exactly what gpk would run.
type planStep struct {
	Manager string   `json:"manager"`
	Name    string   `json:"name"`
	Version string   `json:"version,omitempty"`
	Command []string `json:"command"`
}

type planEnvelope struct {
	Action string     `json:"action"`
	Steps  []planStep `json:"steps"`
}

func emitPlanJSON(w io.Writer, version, action string, steps []planStep) error {
	return writeEnvelope(w, version, planEnvelope{Action: action, Steps: steps})
}

// cmdArgs returns the argv gpk would execute, with the TUI-era sudo "-S" flag
// stripped so the JSON matches what actually runs headless.
func cmdArgs(cmd *exec.Cmd) []string {
	if cmd == nil {
		return nil
	}
	return stripSudoStdinFlag(cmd).Args
}
