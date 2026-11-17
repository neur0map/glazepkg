package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestPrivilegedArgv(t *testing.T) {
	sudoMarked := []string{"sudo", "-S", "pacman", "-S", "--noconfirm", "ripgrep"}
	makepkg := []string{"sh", "-c", "set -e; makepkg -si --noconfirm", "sh", "foo-git"}
	cases := []struct {
		name string
		args []string
		mode string
		want string
		err  bool
	}{
		{"sudo strips -S", sudoMarked, privSudo, "sudo pacman -S --noconfirm ripgrep", false},
		{"pkexec replaces sudo", sudoMarked, privPkexec, "pkexec pacman -S --noconfirm ripgrep", false},
		{"none runs bare", sudoMarked, privNone, "pacman -S --noconfirm ripgrep", false},
		{"unprivileged untouched", []string{"flatpak", "install", "-y", "org.gimp.GIMP"}, privPkexec, "flatpak install -y org.gimp.GIMP", false},
		{"yay gets --sudo pkexec", []string{"yay", "-S", "--noconfirm", "foo"}, privPkexec, "yay --sudo pkexec -S --noconfirm foo", false},
		{"paru gets --sudo pkexec", []string{"/usr/bin/paru", "-S", "foo"}, privPkexec, "/usr/bin/paru --sudo pkexec -S foo", false},
		{"yay keeps an explicit --sudo", []string{"yay", "--sudo", "doas", "-S", "foo"}, privPkexec, "yay --sudo doas -S foo", false},
		{"yay untouched under sudo", []string{"yay", "-S", "foo"}, privSudo, "yay -S foo", false},
		{"yay untouched under none", []string{"yay", "-S", "foo"}, privNone, "yay -S foo", false},
		{"makepkg refused under pkexec", makepkg, privPkexec, "", true},
		{"makepkg fine under sudo", makepkg, privSudo, strings.Join(makepkg, " "), false},
	}
	for _, c := range cases {
		got, err := privilegedArgv(c.args, c.mode)
		if c.err {
			var aur errAURNeedsTerminal
			if !errors.As(err, &aur) || !strings.Contains(err.Error(), "foo-git") || !strings.Contains(err.Error(), "terminal") {
				t.Errorf("%s: err = %v, want a terminal error naming foo-git", c.name, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
			continue
		}
		if s := strings.Join(got, " "); s != c.want {
			t.Errorf("%s: got %q, want %q", c.name, s, c.want)
		}
	}
}

func TestResolvePrivilege(t *testing.T) {
	defer func(f func(string) (string, error)) { pkexecLookPath = f }(pkexecLookPath)
	found := func(string) (string, error) { return "/usr/bin/pkexec", nil }
	missing := func(string) (string, error) { return "", exec.ErrNotFound }

	cases := []struct {
		value string
		tty   bool
		look  func(string) (string, error)
		want  string
		err   bool
	}{
		{"", false, found, privSudo, false},
		{"sudo", false, found, privSudo, false},
		{"PKEXEC", true, missing, privPkexec, false},
		{"none", false, found, privNone, false},
		{"auto", true, found, privSudo, false},
		{"auto", false, found, privPkexec, false},
		{"auto", false, missing, privSudo, false},
		{"doas", false, found, "", true},
	}
	for _, c := range cases {
		pkexecLookPath = c.look
		got, err := resolvePrivilege(c.value, c.tty)
		if (err != nil) != c.err || got != c.want {
			t.Errorf("resolvePrivilege(%q, tty=%v) = %q, %v; want %q (err %v)", c.value, c.tty, got, err, c.want, c.err)
		}
	}
}

func TestLiftPrivilegeFlagBeatsEnv(t *testing.T) {
	t.Setenv("GPK_PRIVILEGE", "pkexec")
	args, mode, err := liftPrivilege([]string{"install", "--privilege", "none", "foo", "--", "--privilege=sudo"}, nil)
	if err != nil || mode != privNone {
		t.Fatalf("mode = %q, err = %v", mode, err)
	}
	if strings.Join(args, " ") != "install foo -- --privilege=sudo" {
		t.Errorf("args = %v", args)
	}

	args, mode, err = liftPrivilege([]string{"remove", "foo"}, nil)
	if err != nil || mode != privPkexec || strings.Join(args, " ") != "remove foo" {
		t.Errorf("env mode: args=%v mode=%q err=%v", args, mode, err)
	}

	if _, _, err := liftPrivilege([]string{"install", "--privilege"}, nil); err == nil {
		t.Error("missing value should be an error")
	}
}

// privFakeMgrs returns a manager whose install is marked privileged the way
// manager.privilegedCmd marks it for a non-root user.
func privFakeMgrs() []manager.Manager {
	return []manager.Manager{&fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: q, Version: "1-1", Source: model.SourcePacman}}, nil
		},
		installCmdYesFn: func(name string) *exec.Cmd {
			return exec.Command("sudo", "-S", "pacman", "-S", "--noconfirm", name)
		},
		installCmdFn: func(name string) *exec.Cmd {
			return exec.Command("sudo", "-S", "pacman", "-S", name)
		},
	}, &fakeManager{
		name: model.SourceAUR, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: q, Version: "1-1", Source: model.SourceAUR}}, nil
		},
		installCmdYesFn: func(name string) *exec.Cmd {
			return exec.Command("sh", "-c", "set -e; makepkg -si --noconfirm", "sh", name)
		},
		installCmdFn: func(name string) *exec.Cmd {
			return exec.Command("sh", "-c", "set -e; makepkg -si", "sh", name)
		},
	}}
}

func planCommand(t *testing.T, out string) []string {
	t.Helper()
	var env struct {
		Data struct {
			Steps []struct {
				Command []string `json:"command"`
			} `json:"steps"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil || len(env.Data.Steps) == 0 {
		t.Fatalf("bad plan JSON %q: %v", out, err)
	}
	return env.Data.Steps[0].Command
}

func TestInstallPlanFollowsPrivilege(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cases := []struct {
		env  string
		args []string
		want string
	}{
		{"", nil, "sudo pacman -S --noconfirm rg"},
		{"", []string{"--privilege", "pkexec"}, "pkexec pacman -S --noconfirm rg"},
		{"pkexec", nil, "pkexec pacman -S --noconfirm rg"},
		{"pkexec", []string{"--privilege=none"}, "pacman -S --noconfirm rg"},
	}
	for _, c := range cases {
		t.Setenv("GPK_PRIVILEGE", c.env)
		var out, errOut bytes.Buffer
		args := append([]string{"install", "rg", "-m", "pacman", "--yes", "--json"}, c.args...)
		if code := Dispatch(args, privFakeMgrs(), "test", &out, &errOut, nil); code != ExitOK {
			t.Fatalf("%v: exit %d (stderr=%q)", args, code, errOut.String())
		}
		if got := strings.Join(planCommand(t, out.String()), " "); got != c.want {
			t.Errorf("env=%q %v: command %q, want %q", c.env, c.args, got, c.want)
		}
	}
}

func TestAURMakepkgUnderPkexecNeedsTerminal(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GPK_PRIVILEGE", "")
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "foo-git", "-m", "aur", "--yes", "--privilege", "pkexec"}, privFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Fatalf("exit %d, want %d", code, ExitErr)
	}
	if !strings.Contains(errOut.String(), "foo-git") || !strings.Contains(errOut.String(), "needs a terminal") {
		t.Errorf("stderr = %q", errOut.String())
	}
	if strings.Contains(out.String(), "Install plan") {
		t.Errorf("nothing should be planned: %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	code = Dispatch([]string{"install", "foo-git", "-m", "aur", "--yes", "--events", "--privilege", "pkexec"}, privFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Fatalf("events: exit %d, want %d", code, ExitErr)
	}
	evs := decodeEvents(t, out.String())
	if len(evs) != 1 || evs[0]["event"] != "done" || !strings.Contains(evs[0]["error"].(string), "needs a terminal") {
		t.Errorf("events = %v", evs)
	}
}

func TestUnknownPrivilegeModeFails(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Dispatch([]string{"install", "x", "--privilege", "doas"}, privFakeMgrs(), "test", &out, &errOut, nil); code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
	if !strings.Contains(errOut.String(), "unknown privilege mode") {
		t.Errorf("stderr = %q", errOut.String())
	}
}
