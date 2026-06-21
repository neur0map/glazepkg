package cli

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

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
	cands, _ := findInstallCandidates("git", []manager.Manager{pacman, brew})
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
	cands, _ := findInstallCandidates("ripgrep", []manager.Manager{pacman, brew})
	if len(cands) != 2 {
		t.Fatalf("got %d candidates, want 2", len(cands))
	}
}

func TestFindCandidates_NotFound(t *testing.T) {
	pacman := &fakeManager{
		name: model.SourcePacman, available: true,
		searchFn: func(q string) ([]model.Package, error) { return nil, nil },
	}
	if cands, _ := findInstallCandidates("nope", []manager.Manager{pacman}); len(cands) != 0 {
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
	cands, _ := findInstallCandidates("yay", []manager.Manager{pacman, aur})
	if len(cands) != 1 || cands[0].mgr.Name() != model.SourceAUR {
		t.Fatalf("got %d candidates, want 1 (aur)", len(cands))
	}
}

// A searcher that errors (e.g. AUR RPC unreachable) with no results surfaces
// the error so callers can say "couldn't reach X" instead of "not found".
func TestFindCandidates_SearchErrorSurfaces(t *testing.T) {
	down := &fakeManager{
		name: model.SourceAUR, available: true,
		searchFn: func(q string) ([]model.Package, error) {
			return nil, errors.New("dial tcp: timeout")
		},
	}
	cands, err := findInstallCandidates("anything", []manager.Manager{down})
	if len(cands) != 0 {
		t.Fatalf("got %d candidates, want 0", len(cands))
	}
	if err == nil {
		t.Error("search error should surface, got nil")
	}
}

func TestExitAmbiguousConstant(t *testing.T) {
	if ExitAmbiguous != 3 {
		t.Errorf("ExitAmbiguous = %d, want 3", ExitAmbiguous)
	}
}

func TestUseStableLocale(t *testing.T) {
	orig, had := os.LookupEnv("LC_ALL")
	defer func() {
		if had {
			os.Setenv("LC_ALL", orig)
		} else {
			os.Unsetenv("LC_ALL")
		}
	}()
	UseStableLocale()
	if os.Getenv("LC_ALL") != "C" {
		t.Errorf("LC_ALL = %q, want C", os.Getenv("LC_ALL"))
	}
}

func TestHeadlessExecSetsUserEnv(t *testing.T) {
	cmd := exec.Command("/bin/true")
	if err := headlessExec(cmd); err != nil {
		t.Fatalf("run: %v", err)
	}
	if cmd.Env == nil {
		t.Error("interactive command should run with the captured user env")
	}
}
