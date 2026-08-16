package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

// realMgrs returns the production manager set for tests that need real names.
// Tests asserting filter behavior against fakes use the fakes directly.
func realMgrs() []manager.Manager { return manager.All() }

func sourceSet(mgrs []manager.Manager) []string {
	out := make([]string, len(mgrs))
	for i, m := range mgrs {
		out[i] = string(m.Name())
	}
	sort.Strings(out)
	return out
}

func TestParseManagerFilter_Empty(t *testing.T) {
	got, err := parseManagerFilter("", realMgrs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(realMgrs()) {
		t.Errorf("empty filter returned %d mgrs, want %d (all)", len(got), len(realMgrs()))
	}
}

func TestParseManagerFilter_All(t *testing.T) {
	got, err := parseManagerFilter("all", realMgrs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(realMgrs()) {
		t.Errorf("'all' filter returned %d mgrs, want %d", len(got), len(realMgrs()))
	}
}

func TestParseManagerFilter_Single(t *testing.T) {
	got, err := parseManagerFilter("pacman", realMgrs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"pacman"}
	if diff := sourceSet(got); !reflect.DeepEqual(diff, want) {
		t.Errorf("got %v, want %v", diff, want)
	}
}

func TestParseManagerFilter_CommaList(t *testing.T) {
	got, err := parseManagerFilter("pacman,aur", realMgrs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"aur", "pacman"}
	if diff := sourceSet(got); !reflect.DeepEqual(diff, want) {
		t.Errorf("got %v, want %v", diff, want)
	}
}

func TestParseManagerFilter_Negation(t *testing.T) {
	// '!brew,!brew-cask' means everything except those two.
	got, err := parseManagerFilter("!brew,!brew-cask", realMgrs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, m := range got {
		if m.Name() == model.SourceBrew || m.Name() == model.SourceBrewCask {
			t.Errorf("expected %s to be excluded", m.Name())
		}
	}
	// Should have len(all) - 2 entries.
	if len(got) != len(realMgrs())-2 {
		t.Errorf("got %d mgrs, want %d", len(got), len(realMgrs())-2)
	}
}

func TestParseManagerFilter_Unknown(t *testing.T) {
	_, err := parseManagerFilter("yum", realMgrs())
	if err == nil {
		t.Fatal("expected error for unknown manager, got nil")
	}
}

func TestParseManagerFilter_MixedPositiveAndNegative(t *testing.T) {
	// Mixing positive and negative selectors is an error — too easy to confuse.
	_, err := parseManagerFilter("pacman,!brew", realMgrs())
	if err == nil {
		t.Fatal("expected error mixing positive+negative selectors, got nil")
	}
}

func TestParseManagerFilter_NegationWithSpace(t *testing.T) {
	// "! pacman" (space slipped after the bang) should behave like "!pacman",
	// not produce a confusing "unknown manager \" pacman\"" error.
	got, err := parseManagerFilter("! pacman", realMgrs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, m := range got {
		if string(m.Name()) == "pacman" {
			t.Errorf("pacman should have been excluded")
		}
	}
}

// writeSkipConfig points the config at a temp dir listing names to skip.
func writeSkipConfig(t *testing.T, names ...string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if len(names) == 0 {
		return
	}
	if err := os.MkdirAll(filepath.Join(dir, "glazepkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "[managers]\nskip = [\"" + strings.Join(names, "\", \"") + "\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "glazepkg", "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A manager that fails identically on every run can be taken out of the default
// set, which is the whole point: one broken pnpm otherwise makes `gpk upgrade`
// exit non-zero forever.
func TestParseManagerFilter_SkipsConfiguredManagers(t *testing.T) {
	writeSkipConfig(t, "pnpm", "npm")
	for _, filter := range []string{"", "all"} {
		got, err := parseManagerFilter(filter, realMgrs())
		if err != nil {
			t.Fatalf("filter %q: %v", filter, err)
		}
		for _, m := range got {
			if m.Name() == model.SourcePnpm || m.Name() == model.SourceNpm {
				t.Errorf("filter %q kept skipped manager %s", filter, m.Name())
			}
		}
		if len(got) != len(realMgrs())-2 {
			t.Errorf("filter %q returned %d mgrs, want %d", filter, len(got), len(realMgrs())-2)
		}
	}
}

// Naming a manager is an explicit request, so it beats the config. That keeps
// the "retry" hint in the failure summary working after a skip.
func TestParseManagerFilter_ExplicitNameBeatsSkip(t *testing.T) {
	writeSkipConfig(t, "pnpm")
	got, err := parseManagerFilter("pnpm", realMgrs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name() != model.SourcePnpm {
		t.Errorf("explicit --manager pnpm returned %v, want just pnpm", sourceSet(got))
	}
}

// A negative filter still honors the skip list; both are exclusions.
func TestParseManagerFilter_SkipCombinesWithNegation(t *testing.T) {
	writeSkipConfig(t, "pnpm")
	got, err := parseManagerFilter("!brew", realMgrs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, m := range got {
		if m.Name() == model.SourcePnpm || m.Name() == model.SourceBrew {
			t.Errorf("filter kept excluded manager %s", m.Name())
		}
	}
}

func TestParseManagerFilter_NoSkipConfigChangesNothing(t *testing.T) {
	writeSkipConfig(t)
	got, err := parseManagerFilter("", realMgrs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(realMgrs()) {
		t.Errorf("returned %d mgrs, want all %d", len(got), len(realMgrs()))
	}
}
