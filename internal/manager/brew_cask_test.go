package manager

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestBrewCaskPackagesFromInstalledJSON(t *testing.T) {
	installed := "150.0.1"
	info := &brewInfo{
		Casks: []brewCask{{
			Token:         "firefox",
			Desc:          "Web browser",
			Version:       "150.0.1",
			Installed:     &installed,
			InstalledTime: 1764630334,
		}},
	}

	pkgs := brewCaskPackages(info, "/opt/homebrew/Caskroom", time.Unix(100, 0))
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 cask, got %d", len(pkgs))
	}
	pkg := pkgs[0]
	if pkg.Name != "firefox" {
		t.Fatalf("Name = %q, want firefox", pkg.Name)
	}
	if pkg.Version != "150.0.1" {
		t.Fatalf("Version = %q, want 150.0.1", pkg.Version)
	}
	if pkg.Description != "Web browser" {
		t.Fatalf("Description = %q, want Web browser", pkg.Description)
	}
	if pkg.Source != model.SourceBrewCask {
		t.Fatalf("Source = %q, want %q", pkg.Source, model.SourceBrewCask)
	}
	if pkg.Location != "/opt/homebrew/Caskroom/firefox/150.0.1" {
		t.Fatalf("Location = %q", pkg.Location)
	}
	if pkg.InstalledAt.Unix() != 1764630334 {
		t.Fatalf("InstalledAt = %v", pkg.InstalledAt)
	}
}

func TestBrewCaskPackagesIgnoresUninstalledCasks(t *testing.T) {
	info := &brewInfo{
		Casks: []brewCask{{
			Token:   "firefox",
			Desc:    "Web browser",
			Version: "150.0.1",
		}},
	}

	pkgs := brewCaskPackages(info, "/opt/homebrew/Caskroom", time.Unix(100, 0))
	if len(pkgs) != 0 {
		t.Fatalf("expected no packages for uninstalled cask, got %#v", pkgs)
	}
}

func TestBrewCaskPackagesNilInfoReturnsNil(t *testing.T) {
	pkgs := brewCaskPackages(nil, "/opt/homebrew/Caskroom", time.Unix(100, 0))
	if pkgs != nil {
		t.Fatalf("expected nil packages, got %#v", pkgs)
	}
}

func TestBrewCaskPackagesIgnoresEmptyToken(t *testing.T) {
	installed := "150.0.1"
	info := &brewInfo{
		Casks: []brewCask{{
			Token:     "",
			Desc:      "Web browser",
			Version:   "150.0.1",
			Installed: &installed,
		}},
	}

	pkgs := brewCaskPackages(info, "/opt/homebrew/Caskroom", time.Unix(100, 0))
	if len(pkgs) != 0 {
		t.Fatalf("expected no packages for empty token, got %#v", pkgs)
	}
}

func TestBrewCaskPackagesFallsBackToVersionForBlankInstalled(t *testing.T) {
	installed := "   "
	info := &brewInfo{
		Casks: []brewCask{{
			Token:     "firefox",
			Desc:      "Web browser",
			Version:   "150.0.1",
			Installed: &installed,
		}},
	}

	pkgs := brewCaskPackages(info, "/opt/homebrew/Caskroom", time.Unix(100, 0))
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 cask, got %d", len(pkgs))
	}
	if pkgs[0].Version != "150.0.1" {
		t.Fatalf("Version = %q, want 150.0.1", pkgs[0].Version)
	}
	if pkgs[0].Location != "/opt/homebrew/Caskroom/firefox/150.0.1" {
		t.Fatalf("Location = %q", pkgs[0].Location)
	}
}

func TestBrewCaskCommandsUseExplicitCaskFlag(t *testing.T) {
	mgr := &BrewCask{}

	tests := map[string][]string{
		"install": mgr.InstallCmd("firefox").Args,
		"upgrade": mgr.UpgradeCmd("firefox").Args,
		"remove":  mgr.RemoveCmd("firefox").Args,
	}
	want := map[string][]string{
		"install": []string{"brew", "install", "--cask", "firefox"},
		"upgrade": []string{"brew", "upgrade", "--cask", "firefox"},
		"remove":  []string{"brew", "uninstall", "--cask", "firefox"},
	}

	for name, got := range tests {
		if !reflect.DeepEqual(got, want[name]) {
			t.Fatalf("%s args = %#v, want %#v", name, got, want[name])
		}
	}
}

func TestParseBrewCaskUpdates(t *testing.T) {
	data := []byte(`{
        "formulae": [],
        "casks": [
            {
                "name": "firefox",
                "installed_versions": ["149.0"],
                "current_version": "150.0.1"
            }
        ]
    }`)

	got := parseBrewCaskUpdates(data)
	if got["firefox"] != "150.0.1" {
		t.Fatalf("firefox update = %q, want 150.0.1", got["firefox"])
	}
}

func TestParseBrewCaskUpdatesSkipsMalformedEntries(t *testing.T) {
	data := []byte(`{
        "casks": [
            {"name": "", "current_version": "150.0.1"},
            {"name": "firefox", "current_version": ""},
            {"name": "chromium", "current_version": "151.0"}
        ]
    }`)

	got := parseBrewCaskUpdates(data)
	if len(got) != 1 {
		t.Fatalf("updates = %#v, want one valid entry", got)
	}
	if got["chromium"] != "151.0" {
		t.Fatalf("chromium update = %q, want 151.0", got["chromium"])
	}
}

func TestParseBrewCaskSearch(t *testing.T) {
	got := parseBrewCaskSearch(`

==> Casks
firefox

chromium
`)

	want := []model.Package{
		{Name: "firefox", Source: model.SourceBrewCask},
		{Name: "chromium", Source: model.SourceBrewCask},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("packages = %#v, want %#v", got, want)
	}
}

func TestBrewInfoJSONIncludesCasks(t *testing.T) {
	data := []byte(`{
        "formulae": [],
        "casks": [
            {"token": "firefox", "desc": "Web browser", "version": "150.0.1", "installed": "150.0.1"}
        ]
    }`)

	var info brewInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(info.Casks) != 1 || info.Casks[0].Token != "firefox" {
		t.Fatalf("unexpected casks: %#v", info.Casks)
	}
}
