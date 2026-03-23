//go:build windows

package manager

import (
	"os"
	"os/exec"

	"golang.org/x/sys/windows"
)

// isElevated reports whether the current process holds Windows administrator
// privileges by inspecting the current process token via the Windows API.
//
// This is the correct, API-level check — unlike heuristics such as running
// "net session", it works without spawning a child process and is not
// affected by UAC virtualisation or session type.
func isElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// privilegedCmd constructs an exec.Cmd that will run with the elevated
// privileges required to write to protected directories such as
// C:\ProgramData\chocolatey\lib\<pkg>\.chocolateyPending.
//
// Resolution order:
//  1. Process is already elevated (running as Administrator) → run directly;
//     no wrapper is needed and choco will succeed without the access-denied
//     error caused by the .chocolateyPending lock file.
//  2. gsudo (https://github.com/gerardog/gsudo) is on PATH → wrap the
//     command with "gsudo --wait" so it runs elevated in the same console
//     window and its stdout/stderr are still captured by glazepkg normally.
//     gsudo can itself be installed with: choco install gsudo
//  3. Neither condition holds → return the command tagged with the sentinel
//     environment variable GLAZEPKG_NEEDS_ELEVATION=1 so that
//     runUpgradeRequest can detect this early and surface a clear, actionable
//     error message instead of letting choco fail deep inside its own
//     file-system operations with a cryptic "Access is denied" error.
func privilegedCmd(name string, args ...string) *exec.Cmd {
	if isElevated() {
		return exec.Command(name, args...)
	}

	// gsudo transparently re-executes the target command in an elevated
	// context while keeping stdout/stderr connected to the parent process,
	// which lets glazepkg stream and capture the output as usual.
	if _, err := exec.LookPath("gsudo"); err == nil {
		// --wait: block until the elevated child exits (required so that
		// CombinedOutput() receives the full output before returning).
		return exec.Command("gsudo", append([]string{"--wait", name}, args...)...)
	}

	// Tag the command with a sentinel so the UI can fail fast with a helpful
	// message rather than propagating the raw Windows "Access is denied" error
	// which gives users no guidance on how to fix it.
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "GLAZEPKG_NEEDS_ELEVATION=1")
	return cmd
}
