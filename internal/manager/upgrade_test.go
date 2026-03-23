package manager

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestBySource(t *testing.T) {
	mgr := BySource(model.SourceBrew)
	if mgr == nil {
		t.Fatal("BySource(brew) returned nil")
	}
	if mgr.Name() != model.SourceBrew {
		t.Fatalf("expected brew, got %s", mgr.Name())
	}
}

func TestBySourceUnknown(t *testing.T) {
	mgr := BySource("nonexistent-manager")
	if mgr != nil {
		t.Fatalf("expected nil for unknown source, got %v", mgr)
	}
}

func TestBySourceAllRegistered(t *testing.T) {
	for _, mgr := range All() {
		got := BySource(mgr.Name())
		if got == nil {
			t.Errorf("BySource(%s) returned nil for a registered manager", mgr.Name())
		}
	}
}

func TestUpgraderInterface(t *testing.T) {
	expected := map[model.Source]bool{
		model.SourceBrew:      true,
		model.SourcePacman:    true,
		model.SourceApt:       true,
		model.SourceDnf:       true,
		model.SourceSnap:      true,
		model.SourcePip:       true,
		model.SourcePipx:      true,
		model.SourceCargo:     true,
		model.SourceNpm:       true,
		model.SourceFlatpak:   true,
		model.SourceGem:       true,
		model.SourceOpam:      true,
		model.SourceApk:       true,
		model.SourceXbps:      true,
		model.SourceConda:     true,
		model.SourceLuarocks:  true,
		model.SourceChocolatey: true,
		model.SourceScoop:     true,
		model.SourceWinget:    true,
	}

	for _, mgr := range All() {
		_, isUpgrader := mgr.(Upgrader)
		if expected[mgr.Name()] && !isUpgrader {
			t.Errorf("%s should implement Upgrader but doesn't", mgr.Name())
		}
	}
}

func TestUpgradeCmdNotNil(t *testing.T) {
	for _, mgr := range All() {
		upgrader, ok := mgr.(Upgrader)
		if !ok {
			continue
		}
		cmd := upgrader.UpgradeCmd("test-package")
		if cmd == nil {
			t.Errorf("%s.UpgradeCmd returned nil", mgr.Name())
		}
		if len(cmd.Args) == 0 {
			t.Errorf("%s.UpgradeCmd returned cmd with no args", mgr.Name())
		}
	}
}

func TestRemoverInterface(t *testing.T) {
	expected := map[model.Source]bool{
		model.SourceBrew:       true,
		model.SourcePacman:     true,
		model.SourceApt:        true,
		model.SourceDnf:        true,
		model.SourceSnap:       true,
		model.SourcePip:        true,
		model.SourcePipx:       true,
		model.SourceCargo:      true,
		model.SourceNpm:        true,
		model.SourceFlatpak:    true,
		model.SourceGem:        true,
		model.SourceOpam:       true,
		model.SourceApk:        true,
		model.SourceXbps:       true,
		model.SourceConda:      true,
		model.SourceLuarocks:   true,
		model.SourceChocolatey: true,
		model.SourceScoop:      true,
		model.SourceWinget:     true,
	}

	for _, mgr := range All() {
		_, isRemover := mgr.(Remover)
		if expected[mgr.Name()] && !isRemover {
			t.Errorf("%s should implement Remover but doesn't", mgr.Name())
		}
	}
}

func TestDeepRemoverInterface(t *testing.T) {
	expected := map[model.Source]bool{
		model.SourceApt:    true,
		model.SourcePacman: true,
		model.SourceDnf:    true,
		model.SourceXbps:   true,
	}

	for _, mgr := range All() {
		_, isDeep := mgr.(DeepRemover)
		if expected[mgr.Name()] && !isDeep {
			t.Errorf("%s should implement DeepRemover but doesn't", mgr.Name())
		}
	}
}

func TestRemoveCmdNotNil(t *testing.T) {
	for _, mgr := range All() {
		remover, ok := mgr.(Remover)
		if !ok {
			continue
		}
		cmd := remover.RemoveCmd("test-package")
		if cmd == nil {
			t.Errorf("%s.RemoveCmd returned nil", mgr.Name())
		}
		if len(cmd.Args) == 0 {
			t.Errorf("%s.RemoveCmd returned cmd with no args", mgr.Name())
		}
	}
}

func TestDeepRemoveCmdNotNil(t *testing.T) {
	for _, mgr := range All() {
		deep, ok := mgr.(DeepRemover)
		if !ok {
			continue
		}
		cmd := deep.RemoveCmdWithDeps("test-package")
		if cmd == nil {
			t.Errorf("%s.RemoveCmdWithDeps returned nil", mgr.Name())
		}
		if len(cmd.Args) == 0 {
			t.Errorf("%s.RemoveCmdWithDeps returned cmd with no args", mgr.Name())
		}
	}
}
