package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func installFakeMgrs() []manager.Manager {
	pacman := &fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			if q == "ripgrep" || q == "git" {
				return []model.Package{{Name: q, Source: model.SourcePacman}}, nil
			}
			return nil, nil
		},
		installCmdFn: func(name string) *exec.Cmd {
			// Use /bin/true so the test "install" succeeds without side effects.
			return exec.Command("/bin/true", "install", name)
		},
	}
	brew := &fakeManager{
		name: model.SourceBrew, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			if q == "ripgrep" {
				return []model.Package{{Name: q, Source: model.SourceBrew}}, nil
			}
			return nil, nil
		},
		installCmdFn: func(name string) *exec.Cmd {
			return exec.Command("/bin/true", "install", name)
		},
	}
	return []manager.Manager{pacman, brew}
}

func TestInstallNoArgs(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Errorf("exit %d, want %d", code, ExitErr)
	}
}

func TestInstallAmbiguous(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "ripgrep"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitAmbiguous {
		t.Errorf("exit %d, want %d (stderr=%q)", code, ExitAmbiguous, errOut.String())
	}
	if !strings.Contains(errOut.String(), "pacman") || !strings.Contains(errOut.String(), "brew") {
		t.Errorf("stderr should list both managers: %q", errOut.String())
	}
}

func TestInstallNotFound(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "nope-does-not-exist"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitNegative {
		t.Errorf("exit %d, want %d", code, ExitNegative)
	}
}

func TestInstallExplicitManagerYes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "ripgrep", "--manager", "brew", "--yes", "--quiet"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, want %d (stderr=%q)", code, ExitOK, errOut.String())
	}
}

func TestInstallDryRun(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "git", "--dry-run"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, want %d", code, ExitOK)
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Errorf("expected dry-run notice in stdout: %q", out.String())
	}
}

func TestInstallPromptYes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("y\n")
	code := Dispatch([]string{"install", "git", "--quiet"}, installFakeMgrs(), "test", &out, &errOut, stdin)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
}

func TestInstallPromptNo(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("n\n")
	code := Dispatch([]string{"install", "git"}, installFakeMgrs(), "test", &out, &errOut, stdin)
	if code != ExitOK {
		t.Fatalf("exit %d (cancellation should still be 0): stderr=%q", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "cancelled") {
		t.Errorf("stderr should mention cancelled: %q", errOut.String())
	}
}

func TestInstallHelpExitsZero(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "--help"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Errorf("exit %d, want %d", code, ExitOK)
	}
}

func TestInstallPickVersionNonInteractive(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	fake := &fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "foo", Source: model.SourcePacman}}, nil
		},
		versionsFn:       func(string) ([]string, error) { return []string{"1.0", "2.0"}, nil },
		installVersionFn: func(n, v string) *exec.Cmd { return exec.Command("/bin/true", n, v) },
	}
	var out, errOut bytes.Buffer
	// No TTY in tests, so --pick-version can't prompt and asks for an explicit one.
	code := Dispatch([]string{"install", "foo", "--manager", "pacman", "--pick-version"}, []manager.Manager{fake}, "test", &out, &errOut, nil)
	if code != ExitErr {
		t.Fatalf("exit %d, want %d", code, ExitErr)
	}
	if !strings.Contains(errOut.String(), "specify a version") {
		t.Errorf("stderr = %q, want it to ask for an explicit version", errOut.String())
	}
	if !strings.Contains(errOut.String(), "2.0") {
		t.Errorf("stderr should list available versions: %q", errOut.String())
	}
}

func TestInstallPreferResolvesAmbiguity(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	if err := os.MkdirAll(filepath.Join(cfgDir, "glazepkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "glazepkg", "config.toml"), []byte("[install]\nprefer = [\"brew\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	// ripgrep is in both pacman and brew; the preference picks brew without asking.
	code := Dispatch([]string{"install", "ripgrep", "--dry-run"}, installFakeMgrs(), "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "brew") {
		t.Errorf("plan should use the preferred manager (brew):\n%s", out.String())
	}
	if strings.Contains(out.String(), "pacman") {
		t.Errorf("plan should not fall back to pacman when brew is preferred:\n%s", out.String())
	}
}

// installerOnly is a manager that can install but not search — the case that
// used to fail "not found" when named explicitly.
type installerOnly struct{ src model.Source }

func (i installerOnly) Name() model.Source             { return i.src }
func (i installerOnly) Available() bool                { return true }
func (i installerOnly) Scan() ([]model.Package, error) { return nil, nil }
func (i installerOnly) InstallCmd(name string) *exec.Cmd {
	return exec.Command("/bin/true", "install", name)
}

func TestInstallExplicitNonSearchableManager(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	m := installerOnly{src: model.SourceGo}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "golang.org/x/tools/cmd/stringer", "--manager", "go", "--dry-run"}, []manager.Manager{m}, "test", &out, &errOut, nil)
	if code != ExitOK {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "stringer") {
		t.Errorf("plan should install via the named manager:\n%s", out.String())
	}
}

func TestInstallAmbiguousOrdersSystemFirst(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// Registration order is brew, then pacman; the message must still lead with
	// pacman because system managers outrank language ecosystems by default.
	rg := func(src model.Source) *fakeManager {
		return &fakeManager{
			name: src, available: true,
			searchFn: func(q string) ([]model.Package, error) {
				return []model.Package{{Name: "ripgrep", Source: src}}, nil
			},
		}
	}
	mgrs := []manager.Manager{rg(model.SourceCargo), rg(model.SourcePacman)}
	var out, errOut bytes.Buffer
	code := Dispatch([]string{"install", "ripgrep"}, mgrs, "test", &out, &errOut, nil)
	if code != ExitAmbiguous {
		t.Fatalf("exit %d, want %d", code, ExitAmbiguous)
	}
	msg := errOut.String()
	if strings.Index(msg, "pacman") > strings.Index(msg, "cargo") {
		t.Errorf("pacman should be listed before cargo: %q", msg)
	}
}
