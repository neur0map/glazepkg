package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
)

// Privilege modes for commands that need root. manager.privilegedCmd marks
// them as `sudo -S <argv>`; the cli rewrites that marker to the chosen mode
// just before running (or showing) the command.
const (
	privSudo   = "sudo"   // sudo on the terminal (the default)
	privPkexec = "pkexec" // polkit; works without a terminal in a desktop session
	privNone   = "none"   // run the privileged argv as is: the caller is root already
	privAuto   = "auto"   // sudo with a terminal, pkexec without one when installed
)

// privilegeEnv names the environment variable that sets the default mode;
// the --privilege flag overrides it.
const privilegeEnv = "GPK_PRIVILEGE"

// privilege is the resolved mode for the current Dispatch. Dispatch resets it
// from the flag and environment on every call.
var privilege = privSudo

// pkexecLookPath finds pkexec; tests replace it.
var pkexecLookPath = exec.LookPath

// liftPrivilege removes `--privilege <mode>` / `--privilege=<mode>` from args
// (wherever it sits before a `--`) and resolves the mode against the
// environment and stdin. It is a global flag so every write command honours it.
func liftPrivilege(args []string, stdin any) ([]string, string, error) {
	value := os.Getenv(privilegeEnv)
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			out = append(out, args[i:]...)
			break
		}
		switch {
		case a == "--privilege":
			if i+1 >= len(args) {
				return nil, "", fmt.Errorf("--privilege needs a value (sudo, pkexec, none or auto)")
			}
			value = args[i+1]
			i++
			continue
		case strings.HasPrefix(a, "--privilege="):
			value = strings.TrimPrefix(a, "--privilege=")
			continue
		}
		out = append(out, a)
	}
	mode, err := resolvePrivilege(value, stdinIsTerminal(stdin))
	return out, mode, err
}

// resolvePrivilege validates a mode name and turns auto into a concrete mode.
// An empty value keeps gpk's historic behaviour: sudo.
func resolvePrivilege(value string, tty bool) (string, error) {
	switch v := strings.ToLower(strings.TrimSpace(value)); v {
	case "", privSudo:
		return privSudo, nil
	case privPkexec, privNone:
		return v, nil
	case privAuto:
		if tty {
			return privSudo, nil
		}
		if _, err := pkexecLookPath("pkexec"); err == nil {
			return privPkexec, nil
		}
		return privSudo, nil
	default:
		return "", fmt.Errorf("unknown privilege mode %q (want sudo, pkexec, none or auto)", value)
	}
}

func stdinIsTerminal(stdin any) bool {
	f, ok := stdin.(*os.File)
	return ok && isatty.IsTerminal(f.Fd())
}

// errAURNeedsTerminal is returned for an AUR build gpk would run with makepkg
// (no yay or paru) when the privilege mode can't prompt: makepkg calls sudo on
// its own, which needs a terminal.
type errAURNeedsTerminal struct{ name string }

func (e errAURNeedsTerminal) Error() string {
	return fmt.Sprintf("building %s from the AUR with makepkg needs a terminal, because makepkg asks for the sudo password itself; run `gpk install %s -m aur` in a terminal, or install yay or paru", e.name, e.name)
}

// privilegedArgv rewrites one command line for mode:
//   - the `sudo -S <argv>` marker becomes `sudo <argv>`, `pkexec <argv>` or
//     plain `<argv>`;
//   - yay and paru get `--sudo pkexec` in pkexec mode, since they escalate
//     their own pacman calls;
//   - a makepkg build fails in pkexec mode (see errAURNeedsTerminal).
//
// Anything else is returned unchanged.
func privilegedArgv(args []string, mode string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}
	if len(args) >= 2 && args[0] == "sudo" && args[1] == "-S" {
		rest := args[2:]
		switch mode {
		case privPkexec:
			return append([]string{"pkexec"}, rest...), nil
		case privNone:
			return append([]string(nil), rest...), nil
		default:
			return append([]string{"sudo"}, rest...), nil
		}
	}
	if mode != privPkexec {
		return args, nil
	}
	switch filepath.Base(args[0]) {
	case "yay", "paru":
		for _, a := range args[1:] {
			if a == "--sudo" || strings.HasPrefix(a, "--sudo=") {
				return args, nil
			}
		}
		out := append([]string{args[0], "--sudo", "pkexec"}, args[1:]...)
		return out, nil
	case "sh":
		if len(args) >= 3 && args[1] == "-c" && strings.Contains(args[2], "makepkg") {
			return nil, errAURNeedsTerminal{name: args[len(args)-1]}
		}
	}
	return args, nil
}

// withPrivilege returns cmd rewritten for the current privilege mode. The
// original is never modified; a rewritten command keeps its Dir and Env.
func withPrivilege(cmd *exec.Cmd) (*exec.Cmd, error) {
	if cmd == nil {
		return nil, fmt.Errorf("nil command")
	}
	args, err := privilegedArgv(cmd.Args, privilege)
	if err != nil {
		return cmd, err
	}
	if sameArgs(args, cmd.Args) {
		return cmd, nil
	}
	out := exec.Command(args[0], args[1:]...)
	out.Dir = cmd.Dir
	out.Env = cmd.Env
	return out, nil
}

// displayArgs is the argv shown in plans: the command as it will run, or the
// sudo-stripped form when the mode rejects it (the run reports the error).
func displayArgs(cmd *exec.Cmd) []string {
	if c, err := withPrivilege(cmd); err == nil {
		return c.Args
	}
	return stripSudoStdinFlag(cmd).Args
}

func sameArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
