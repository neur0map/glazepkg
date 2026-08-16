package parsing

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

// Real output captured from `aqua list --installed --all` (aqua v2.62.3).
// Fields are tab-separated: "<package name>\t<version>\t<registry name>".
func TestAquaListInstalledParsing(t *testing.T) {
	output := "cli/cli\tv2.97.0\tstandard\n" +
		"junegunn/fzf\tv0.74.2\tstandard\n" +
		"BurntSushi/ripgrep\t15.2.0\tstandard\n"

	pkgs := manager.ParseAquaList([]byte(output))
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}

	// ParseAquaList sorts by name: BurntSushi/ripgrep, cli/cli, junegunn/fzf.
	want := []struct {
		name, version, registry string
	}{
		{"BurntSushi/ripgrep", "15.2.0", "standard"},
		{"cli/cli", "v2.97.0", "standard"},
		{"junegunn/fzf", "v0.74.2", "standard"},
	}
	for i, w := range want {
		if pkgs[i].Name != w.name || pkgs[i].Version != w.version {
			t.Errorf("pkg %d: got %q %q, want %q %q", i, pkgs[i].Name, pkgs[i].Version, w.name, w.version)
		}
		if pkgs[i].Repository != w.registry {
			t.Errorf("pkg %d registry: got %q, want %q", i, pkgs[i].Repository, w.registry)
		}
		if pkgs[i].Source != model.SourceAqua {
			t.Errorf("pkg %d source: got %q, want %q", i, pkgs[i].Source, model.SourceAqua)
		}
	}
}

// Real output captured from `aqua list` (registry packages), format
// "<registry name>,<package name>". Search filters by package-name substring.
func TestAquaRegistrySearchParsing(t *testing.T) {
	output := "standard,tsenart/vegeta\n" +
		"standard,bonnefoa/kubectl-fzf\n" +
		"standard,junegunn/fzf/fzf-tmux\n" +
		"standard,junegunn/fzf\n"

	matches := manager.ParseAquaRegistryList([]byte(output), "fzf")
	if len(matches) != 3 {
		t.Fatalf("expected 3 fzf matches, got %d: %+v", len(matches), matches)
	}
	for _, m := range matches {
		if m.Source != model.SourceAqua {
			t.Errorf("match %q source: got %q, want %q", m.Name, m.Source, model.SourceAqua)
		}
		if m.Repository != "standard" {
			t.Errorf("match %q registry: got %q, want standard", m.Name, m.Repository)
		}
	}
	if matches[0].Name != "bonnefoa/kubectl-fzf" {
		t.Errorf("first match: got %q, want bonnefoa/kubectl-fzf", matches[0].Name)
	}

	// vegeta must be filtered out (no "fzf" substring).
	all := manager.ParseAquaRegistryList([]byte(output), "")
	if len(all) != 4 {
		t.Fatalf("empty query should return all 4 entries, got %d", len(all))
	}
}
