package cli

import (
	"bytes"
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

func TestFindCandidates_SingleMatch(t *testing.T) {
	pacman := &fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "git", Source: model.SourcePacman}}, nil
		},
	}
	brew := &fakeManager{
		name: model.SourceBrew, available: true,
		searchFn: func(q string) ([]model.Package, error) { return nil, nil },
	}
	cands := findInstallCandidates("git", []manager.Manager{pacman, brew})
	if len(cands) != 1 || cands[0].mgr.Name() != model.SourcePacman {
		t.Fatalf("got %d candidates, want 1 (pacman)", len(cands))
	}
}

func TestFindCandidates_Ambiguous(t *testing.T) {
	pacman := &fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "ripgrep"}}, nil
		},
	}
	brew := &fakeManager{
		name: model.SourceBrew, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "ripgrep"}}, nil
		},
	}
	cands := findInstallCandidates("ripgrep", []manager.Manager{pacman, brew})
	if len(cands) != 2 {
		t.Fatalf("got %d candidates, want 2", len(cands))
	}
}

func TestFindCandidates_NotFound(t *testing.T) {
	pacman := &fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) { return nil, nil },
	}
	if cands := findInstallCandidates("nope", []manager.Manager{pacman}); len(cands) != 0 {
		t.Fatalf("got %d candidates, want 0", len(cands))
	}
}

func TestFindCandidates_DedupesToCanonicalManager(t *testing.T) {
	// pacman's search surfaces an AUR-sourced result; the AUR manager returns
	// the same package. The candidate must map to the AUR manager and not
	// appear twice.
	pacman := &fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "yay", Source: model.SourceAUR}}, nil
		},
	}
	aur := &fakeManager{
		name: model.SourceAUR, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return []model.Package{{Name: "yay", Source: model.SourceAUR}}, nil
		},
	}
	cands := findInstallCandidates("yay", []manager.Manager{pacman, aur})
	if len(cands) != 1 || cands[0].mgr.Name() != model.SourceAUR {
		t.Fatalf("got %d candidates, want 1 (aur)", len(cands))
	}
}

func TestExitAmbiguousConstant(t *testing.T) {
	if ExitAmbiguous != 3 {
		t.Errorf("ExitAmbiguous = %d, want 3", ExitAmbiguous)
	}
}
