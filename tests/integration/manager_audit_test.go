package integration

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
)

// TestManagerCapabilityMatrix prints a matrix of which managers implement
// which write-capability interfaces. Doesn't fail the test on missing caps —
// the output IS the assertion artifact.
//
// Run with: go test ./tests/integration/... -run TestManagerCapabilityMatrix -v
func TestManagerCapabilityMatrix(t *testing.T) {
	headers := []string{"manager", "avail", "Inst", "Inst+y", "Up", "Up+y", "Rm", "Rm+y", "Deep", "Deep+y"}
	t.Log(formatRow(headers, headerWidths))
	t.Log(formatRow([]string{strings.Repeat("-", 18), "-----", "----", "------", "--", "----", "--", "----", "----", "------"}, headerWidths))
	for _, m := range manager.All() {
		name := string(m.Name())
		avail := "no"
		if m.Available() {
			avail = "YES"
		}
		row := []string{
			name,
			avail,
			hasIface(m, "Installer"),
			hasIface(m, "NonInteractiveInstaller"),
			hasIface(m, "Upgrader"),
			hasIface(m, "NonInteractiveUpgrader"),
			hasIface(m, "Remover"),
			hasIface(m, "NonInteractiveRemover"),
			hasIface(m, "DeepRemover"),
			hasIface(m, "NonInteractiveDeepRemover"),
		}
		t.Log(formatRow(row, headerWidths))
	}
}

var headerWidths = []int{18, 5, 4, 6, 2, 4, 2, 4, 4, 6}

func formatRow(cells []string, widths []int) string {
	var b strings.Builder
	for i, c := range cells {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(fmt.Sprintf("%-*s", widths[i], c))
	}
	return b.String()
}

// hasIface returns "✓" if m implements the named interface, "-" if not.
// Implemented as a switch on string name because Go doesn't have reflective
// "does this implement interface named X" without explicit type assertions.
func hasIface(m manager.Manager, name string) string {
	yes, no := "✓", "-"
	switch name {
	case "Installer":
		if _, ok := m.(manager.Installer); ok {
			return yes
		}
	case "NonInteractiveInstaller":
		if _, ok := m.(manager.NonInteractiveInstaller); ok {
			return yes
		}
	case "Upgrader":
		if _, ok := m.(manager.Upgrader); ok {
			return yes
		}
	case "NonInteractiveUpgrader":
		if _, ok := m.(manager.NonInteractiveUpgrader); ok {
			return yes
		}
	case "Remover":
		if _, ok := m.(manager.Remover); ok {
			return yes
		}
	case "NonInteractiveRemover":
		if _, ok := m.(manager.NonInteractiveRemover); ok {
			return yes
		}
	case "DeepRemover":
		if _, ok := m.(manager.DeepRemover); ok {
			return yes
		}
	case "NonInteractiveDeepRemover":
		if _, ok := m.(manager.NonInteractiveDeepRemover); ok {
			return yes
		}
	}
	return no
}

// TestManagerCommandSamples prints the actual *exec.Cmd args that each
// manager would produce for a dummy package called "DUMMY" — both
// interactive and non-interactive variants. Lets the user audit that the
// commands are sensible without running them.
func TestManagerCommandSamples(t *testing.T) {
	const pkg = "DUMMY"
	for _, m := range manager.All() {
		t.Run(string(m.Name()), func(t *testing.T) {
			lines := []string{fmt.Sprintf("=== %s ===", m.Name())}
			lines = append(lines, fmt.Sprintf("  available: %v", m.Available()))

			if inst, ok := m.(manager.Installer); ok {
				lines = append(lines, fmt.Sprintf("  install:        %s", joinArgs(inst.InstallCmd(pkg))))
			} else {
				lines = append(lines, "  install:        (not supported)")
			}
			if ni, ok := m.(manager.NonInteractiveInstaller); ok {
				lines = append(lines, fmt.Sprintf("  install --yes:  %s", joinArgs(ni.InstallCmdYes(pkg))))
			} else {
				lines = append(lines, "  install --yes:  (falls back to interactive)")
			}

			if up, ok := m.(manager.Upgrader); ok {
				lines = append(lines, fmt.Sprintf("  upgrade:        %s", joinArgs(up.UpgradeCmd(pkg))))
			} else {
				lines = append(lines, "  upgrade:        (not supported)")
			}
			if ni, ok := m.(manager.NonInteractiveUpgrader); ok {
				lines = append(lines, fmt.Sprintf("  upgrade --yes:  %s", joinArgs(ni.UpgradeCmdYes(pkg))))
			} else {
				lines = append(lines, "  upgrade --yes:  (falls back to interactive)")
			}

			if rm, ok := m.(manager.Remover); ok {
				lines = append(lines, fmt.Sprintf("  remove:         %s", joinArgs(rm.RemoveCmd(pkg))))
			} else {
				lines = append(lines, "  remove:         (not supported)")
			}
			if ni, ok := m.(manager.NonInteractiveRemover); ok {
				lines = append(lines, fmt.Sprintf("  remove --yes:   %s", joinArgs(ni.RemoveCmdYes(pkg))))
			} else {
				lines = append(lines, "  remove --yes:   (falls back to interactive)")
			}

			if dr, ok := m.(manager.DeepRemover); ok {
				lines = append(lines, fmt.Sprintf("  remove deps:    %s", joinArgs(dr.RemoveCmdWithDeps(pkg))))
			}
			if ni, ok := m.(manager.NonInteractiveDeepRemover); ok {
				lines = append(lines, fmt.Sprintf("  remove deps -y: %s", joinArgs(ni.RemoveCmdWithDepsYes(pkg))))
			}

			t.Log(strings.Join(lines, "\n"))
		})
	}
}

func joinArgs(cmd *exec.Cmd) string {
	if cmd == nil {
		return "(nil)"
	}
	return cmd.String()
}
