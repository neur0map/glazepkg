package cli

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func TestConfirmAction_YesVariants(t *testing.T) {
	for _, input := range []string{"y\n", "yes\n", "Y\n", "YES\n", " y \n"} {
		t.Run(input, func(t *testing.T) {
			var out bytes.Buffer
			got := confirmAction("Proceed? [y/N] ", strings.NewReader(input), &out)
			if !got {
				t.Errorf("input %q expected true, got false", input)
			}
		})
	}
}

func TestConfirmAction_NoVariants(t *testing.T) {
	for _, input := range []string{"n\n", "no\n", "\n", "anything else\n", ""} {
		t.Run(input, func(t *testing.T) {
			var out bytes.Buffer
			got := confirmAction("Proceed? [y/N] ", strings.NewReader(input), &out)
			if got {
				t.Errorf("input %q expected false, got true", input)
			}
		})
	}
}

func TestConfirmAction_NilStdinReturnsFalse(t *testing.T) {
	var out bytes.Buffer
	if confirmAction("Proceed? [y/N] ", nil, &out) {
		t.Error("expected false on nil stdin")
	}
}

func TestStripSudoStdinFlag_RemovesS(t *testing.T) {
	cmd := exec.Command("sudo", "-S", "pacman", "-S", "git")
	stripped := stripSudoStdinFlag(cmd)
	if stripped.Args[0] != "sudo" {
		t.Errorf("args[0] = %q, want sudo", stripped.Args[0])
	}
	if len(stripped.Args) < 2 || stripped.Args[1] != "pacman" {
		t.Errorf("args = %v, want [sudo pacman -S git]", stripped.Args)
	}
	// Importantly the SECOND "-S" (pacman's install flag) must survive.
	joined := strings.Join(stripped.Args, " ")
	if !strings.Contains(joined, "pacman -S git") {
		t.Errorf("lost pacman flags: %q", joined)
	}
}

func TestStripSudoStdinFlag_LeavesNonSudoAlone(t *testing.T) {
	cmd := exec.Command("brew", "install", "git")
	stripped := stripSudoStdinFlag(cmd)
	if stripped != cmd {
		t.Error("non-sudo command should be returned unchanged")
	}
}

func TestStripSudoStdinFlag_SudoWithoutSStaysIntact(t *testing.T) {
	cmd := exec.Command("sudo", "pacman", "-S", "git")
	stripped := stripSudoStdinFlag(cmd)
	if len(stripped.Args) != 4 || stripped.Args[1] != "pacman" {
		t.Errorf("args = %v, want [sudo pacman -S git]", stripped.Args)
	}
}

func TestResolveInstallManager_SingleMatch(t *testing.T) {
	pacman := &fakeManager{
		name:      model.SourcePacman,
		available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "git", Source: model.SourcePacman}}, nil
		},
	}
	brew := &fakeManager{
		name:      model.SourceBrew,
		available: true,
		searchFn: func(q string) ([]model.Package, error) { return nil, nil },
	}
	got, err := resolveInstallManager("git", []manager.Manager{pacman, brew})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != model.SourcePacman {
		t.Errorf("got %s, want pacman", got.Name())
	}
}

func TestResolveInstallManager_AmbiguousReturnsErr(t *testing.T) {
	pacman := &fakeManager{
		name:      model.SourcePacman,
		available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "ripgrep"}}, nil
		},
	}
	brew := &fakeManager{
		name:      model.SourceBrew,
		available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "ripgrep"}}, nil
		},
	}
	_, err := resolveInstallManager("ripgrep", []manager.Manager{pacman, brew})
	if !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("got %v, want ErrAmbiguous", err)
	}
	if !strings.Contains(err.Error(), "pacman") || !strings.Contains(err.Error(), "brew") {
		t.Errorf("error message should list both managers: %v", err)
	}
}

func TestResolveInstallManager_NotFoundReturnsErr(t *testing.T) {
	pacman := &fakeManager{
		name:      model.SourcePacman,
		available: true,
		searchFn: func(q string) ([]model.Package, error) { return nil, nil },
	}
	_, err := resolveInstallManager("nope", []manager.Manager{pacman})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestResolveInstallManager_SkipsNonSearcher(t *testing.T) {
	// A fakeManager whose searchFn is nil acts as a non-Searcher via the
	// nil-guard in Search(), but the type assertion still succeeds.
	// We need a separate type that doesn't implement Search at all.
	pacman := &fakeManager{
		name:      model.SourcePacman,
		available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "git"}}, nil
		},
	}
	got, err := resolveInstallManager("git", []manager.Manager{pacman})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != model.SourcePacman {
		t.Errorf("got %s, want pacman", got.Name())
	}
}

func TestExitAmbiguousConstant(t *testing.T) {
	if ExitAmbiguous != 3 {
		t.Errorf("ExitAmbiguous = %d, want 3", ExitAmbiguous)
	}
}
