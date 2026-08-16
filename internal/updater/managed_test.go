package updater

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestManagedByPath(t *testing.T) {
	cases := []struct {
		path    string
		manager string
		upgrade string
	}{
		{"/opt/homebrew/Cellar/gpk/0.6.6/bin/gpk", "homebrew", "brew upgrade gpk"},
		{"/usr/local/Cellar/gpk/0.6.6/bin/gpk", "homebrew", "brew upgrade gpk"},
		{"/home/linuxbrew/.linuxbrew/Cellar/gpk/0.6.6/bin/gpk", "homebrew", "brew upgrade gpk"},
		{"/nix/store/abc123-gpk-0.6.6/bin/gpk", "nix", "nix profile upgrade gpk"},
		{"C:/Users/x/scoop/apps/gpk/current/gpk.exe", "scoop", "scoop update gpk"},
		{"C:/ProgramData/chocolatey/lib/gpk/tools/gpk.exe", "chocolatey", "choco upgrade gpk"},
		{"/usr/local/bin/gpk", "", ""},
		{"/home/x/go/bin/gpk", "", ""},
		{"/home/x/.local/bin/gpk", "", ""},
	}
	for _, c := range cases {
		got := managedByPath(c.path)
		if got.Manager != c.manager || got.Upgrade != c.upgrade {
			t.Errorf("managedByPath(%q) = %+v, want manager %q upgrade %q", c.path, got, c.manager, c.upgrade)
		}
	}
}

// A path that merely mentions a manager's name must not count. Only the real
// install layout does, or gpk would refuse to update itself for someone who
// happens to keep a directory called Cellar.
func TestManagedByPathIgnoresLookalikes(t *testing.T) {
	for _, p := range []string{
		"/home/x/Cellar/gpk",
		"/home/x/Cellar/wine/gpk",
		"/home/x/projects/nix/store/gpk",
		"/nix/store/gpk",
		"/home/x/scoop/gpk.exe",
		"/home/x/scoop-notes/apps/gpk/gpk.exe",
		"/opt/chocolatey/gpk.exe",
	} {
		if got := managedByPath(p); got.Manager != "" {
			t.Errorf("managedByPath(%q) claimed %q", p, got.Manager)
		}
	}
}

func TestPacmanOwner(t *testing.T) {
	if got := pacmanOwner("/usr/bin/gpk is owned by gpk-bin 0.6.6\n"); got != "gpk-bin" {
		t.Errorf("got %q, want gpk-bin", got)
	}
	if got := pacmanOwner("error: No package owns /tmp/x\n"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestDpkgOwner(t *testing.T) {
	cases := map[string]string{
		"gpk: /usr/bin/gpk\n":             "gpk",
		"libc-bin, libc6: /usr/bin/gpk\n": "libc-bin",
		"no colon here\n":                 "",
	}
	for in, want := range cases {
		if got := dpkgOwner(in); got != want {
			t.Errorf("dpkgOwner(%q) = %q, want %q", in, got, want)
		}
	}
}

// The running test binary lives in a temp dir no package owns, so Managed must
// report an unowned install rather than guessing.
func TestManagedOnUnownedBinary(t *testing.T) {
	if got := Managed(); got.Manager != "" {
		t.Errorf("test binary reported as managed by %q", got.Manager)
	}
}

// Checked against the real package database: a binary the system package
// manager owns must be detected, with the owning package named.
func TestManagedByPackageDBOnRealBinary(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("system package database probe is linux-only here")
	}
	var probe string
	for _, bin := range []string{"pacman", "dpkg", "rpm"} {
		if _, err := exec.LookPath(bin); err == nil {
			probe = bin
			break
		}
	}
	if probe == "" {
		t.Skip("no system package database available")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}
	resolved, err := filepath.EvalSymlinks(sh)
	if err != nil {
		t.Skip(err)
	}

	got := managedByPackageDB(resolved)
	if got.Manager != probe {
		t.Errorf("owner of %s = %+v, want manager %q", resolved, got, probe)
	}
	if got.Package == "" {
		t.Errorf("owner of %s named no package", resolved)
	}
	if got.Upgrade != "" {
		t.Errorf("no upgrade command should be guessed for %s, got %q", probe, got.Upgrade)
	}

	unowned := filepath.Join(t.TempDir(), "definitely-unowned")
	if err := os.WriteFile(unowned, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := managedByPackageDB(unowned); got.Manager != "" {
		t.Errorf("unowned file reported as managed by %q", got.Manager)
	}
}

func TestUpgradeHint(t *testing.T) {
	cases := []struct {
		in   Install
		want string
	}{
		{Install{}, "run `gpk update`"},
		{Install{Manager: "homebrew", Package: "gpk", Upgrade: "brew upgrade gpk"}, "run `brew upgrade gpk`"},
		{Install{Manager: "pacman", Package: "gpk-bin"}, "upgrade gpk-bin with pacman"},
	}
	for _, c := range cases {
		if got := c.in.UpgradeHint(); got != c.want {
			t.Errorf("%+v.UpgradeHint() = %q, want %q", c.in, got, c.want)
		}
	}
}
