package manager

import (
	"fmt"
	"os/exec"
	"strings"
)

const maxCmdErrOutput = 800

func runCommand(cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		return err
	}
	if len(msg) > maxCmdErrOutput {
		msg = msg[:maxCmdErrOutput] + "..."
	}
	return fmt.Errorf("%w: %s", err, msg)
}
