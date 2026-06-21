package cli

import (
	"os/exec"
	"testing"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// fakeManager is a configurable Manager for tests. Each capability interface
// method delegates to a function pointer; unset pointers behave as "the
// manager doesn't implement that capability" via the wrapping interface
// assertions in the cli package. Tests use only the fields they care about.
type fakeManager struct {
	name      model.Source
	available bool

	scanFn                 func() ([]model.Package, error)
	checkUpdatesFn         func(pkgs []model.Package) map[string]string
	describeFn             func(pkgs []model.Package) map[string]string
	depsFn                 func(pkgs []model.Package) map[string][]string
	upgradeCmdFn           func(name string) *exec.Cmd
	upgradeCmdYesFn        func(name string) *exec.Cmd
	removeCmdFn            func(name string) *exec.Cmd
	removeCmdYesFn         func(name string) *exec.Cmd
	removeCmdWithDepsFn    func(name string) *exec.Cmd
	removeCmdWithDepsYesFn func(name string) *exec.Cmd
	searchFn               func(query string) ([]model.Package, error)
	installCmdFn           func(name string) *exec.Cmd
	installCmdYesFn        func(name string) *exec.Cmd
	upgradeAllCmdFn        func(yes bool) *exec.Cmd
	versionsFn             func(name string) ([]string, error)
	installVersionFn       func(name, version string) *exec.Cmd
	cleanCacheFn           func(all, yes bool) *exec.Cmd
	orphansFn              func() ([]string, error)
	removeOrphansFn        func(orphans []string, yes bool) *exec.Cmd
}

func (f *fakeManager) Name() model.Source { return f.name }
func (f *fakeManager) Available() bool    { return f.available }

func (f *fakeManager) Scan() ([]model.Package, error) {
	if f.scanFn == nil {
		return nil, nil
	}
	return f.scanFn()
}

func (f *fakeManager) CheckUpdates(pkgs []model.Package) map[string]string {
	if f.checkUpdatesFn == nil {
		return nil
	}
	return f.checkUpdatesFn(pkgs)
}

func (f *fakeManager) Describe(pkgs []model.Package) map[string]string {
	if f.describeFn == nil {
		return nil
	}
	return f.describeFn(pkgs)
}

func (f *fakeManager) ListDependencies(pkgs []model.Package) map[string][]string {
	if f.depsFn == nil {
		return nil
	}
	return f.depsFn(pkgs)
}

func (f *fakeManager) UpgradeCmd(name string) *exec.Cmd {
	if f.upgradeCmdFn == nil {
		return nil
	}
	return f.upgradeCmdFn(name)
}

func (f *fakeManager) RemoveCmd(name string) *exec.Cmd {
	if f.removeCmdFn == nil {
		return nil
	}
	return f.removeCmdFn(name)
}

func (f *fakeManager) Search(query string) ([]model.Package, error) {
	if f.searchFn == nil {
		return nil, nil
	}
	return f.searchFn(query)
}

func (f *fakeManager) InstallCmd(name string) *exec.Cmd {
	if f.installCmdFn == nil {
		return nil
	}
	return f.installCmdFn(name)
}

func (f *fakeManager) InstallCmdYes(name string) *exec.Cmd {
	if f.installCmdYesFn == nil {
		return nil
	}
	return f.installCmdYesFn(name)
}

func (f *fakeManager) UpgradeAllCmd(yes bool) *exec.Cmd {
	if f.upgradeAllCmdFn == nil {
		return nil
	}
	return f.upgradeAllCmdFn(yes)
}

func (f *fakeManager) Versions(name string) ([]string, error) {
	if f.versionsFn == nil {
		return nil, nil
	}
	return f.versionsFn(name)
}

func (f *fakeManager) InstallVersionCmd(name, version string) *exec.Cmd {
	if f.installVersionFn == nil {
		return nil
	}
	return f.installVersionFn(name, version)
}

func (f *fakeManager) CleanCacheCmd(all, yes bool) *exec.Cmd {
	if f.cleanCacheFn == nil {
		return nil
	}
	return f.cleanCacheFn(all, yes)
}

func (f *fakeManager) Orphans() ([]string, error) {
	if f.orphansFn == nil {
		return nil, nil
	}
	return f.orphansFn()
}

func (f *fakeManager) RemoveOrphansCmd(orphans []string, yes bool) *exec.Cmd {
	if f.removeOrphansFn == nil {
		return nil
	}
	return f.removeOrphansFn(orphans, yes)
}

func (f *fakeManager) UpgradeCmdYes(name string) *exec.Cmd {
	if f.upgradeCmdYesFn == nil {
		return nil
	}
	return f.upgradeCmdYesFn(name)
}

func (f *fakeManager) RemoveCmdYes(name string) *exec.Cmd {
	if f.removeCmdYesFn == nil {
		return nil
	}
	return f.removeCmdYesFn(name)
}

func (f *fakeManager) RemoveCmdWithDeps(name string) *exec.Cmd {
	if f.removeCmdWithDepsFn == nil {
		return nil
	}
	return f.removeCmdWithDepsFn(name)
}

func (f *fakeManager) RemoveCmdWithDepsYes(name string) *exec.Cmd {
	if f.removeCmdWithDepsYesFn == nil {
		return nil
	}
	return f.removeCmdWithDepsYesFn(name)
}

// fakePackage is a one-liner constructor for tests that only care about the
// Name/Version/Source triple.
func fakePackage(name, version string, source model.Source) model.Package {
	return model.Package{
		Name:        name,
		Version:     version,
		Source:      source,
		InstalledAt: time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC),
	}
}

// TestFakeManagerSmoke ensures the fake compiles and behaves as expected
// before subcommand tests start depending on it.
func TestFakeManagerSmoke(t *testing.T) {
	pkg := fakePackage("foo", "1.0", model.SourcePacman)
	f := &fakeManager{
		name:      model.SourcePacman,
		available: true,
		scanFn:    func() ([]model.Package, error) { return []model.Package{pkg}, nil },
	}
	if f.Name() != model.SourcePacman {
		t.Errorf("Name() = %s", f.Name())
	}
	if !f.Available() {
		t.Errorf("Available() = false")
	}
	got, err := f.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got) != 1 || got[0].Name != "foo" {
		t.Errorf("Scan returned %+v", got)
	}
}
