package ui

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestPackageURL(t *testing.T) {
	cases := []struct {
		pkg  model.Package
		want string
	}{
		{model.Package{Name: "btop", Source: model.SourceBrew}, "https://formulae.brew.sh/formula/btop"},
		{model.Package{Name: "btop", Source: model.SourceBrewCask}, "https://formulae.brew.sh/cask/btop"},
		{model.Package{Name: "typescript", Source: model.SourceNpm}, "https://www.npmjs.com/package/typescript"},
		{model.Package{Name: "ripgrep", Source: model.SourceCargo}, "https://crates.io/crates/ripgrep"},
		{model.Package{Name: "black", Source: model.SourcePip}, "https://pypi.org/project/black"},
		{model.Package{Name: "yay", Source: model.SourceAUR}, "https://aur.archlinux.org/packages/yay"},
		{model.Package{Name: "x", Source: model.SourceWindowsUpdates}, ""},
	}
	for _, c := range cases {
		if got := packageURL(c.pkg); got != c.want {
			t.Errorf("packageURL(%s/%s) = %q, want %q", c.pkg.Source, c.pkg.Name, got, c.want)
		}
	}
}

func TestOpenURLForOS(t *testing.T) {
	cases := map[string]string{
		"darwin":  "open",
		"windows": "rundll32",
		"linux":   "xdg-open",
	}
	for goos, want := range cases {
		cmd := openURLForOS(goos, "https://example.com")
		if cmd.Args[0] != want {
			t.Errorf("openURLForOS(%s) -> %q, want %q", goos, cmd.Args[0], want)
		}
	}
}

func TestHandleDetailKeyOpenURL(t *testing.T) {
	m := &Model{detailPkg: model.Package{Name: "btop", Source: model.SourceBrew}}
	if _, cmd := m.handleDetailKey("o"); cmd == nil {
		t.Fatal("expected a command to open the URL")
	}

	m2 := &Model{detailPkg: model.Package{Name: "x", Source: model.SourceWindowsUpdates}}
	_, cmd := m2.handleDetailKey("o")
	if cmd != nil {
		t.Fatal("expected no command when URL is unavailable")
	}
	if m2.statusMsg == "" {
		t.Error("expected a status message when URL is unavailable")
	}
}
