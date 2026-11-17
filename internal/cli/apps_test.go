package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func fixtureApps(noCache bool) []manager.DesktopApp {
	return []manager.DesktopApp{
		{DesktopID: "firefox", Name: "Firefox", Categories: []string{"Network"}, DesktopFile: "/usr/share/applications/firefox.desktop",
			Package: "firefox", Source: model.SourcePacman, Version: "1-1", Size: "302.0 MiB", SizeBytes: 316628008, Location: "system"},
		{DesktopID: "org.gimp.GIMP", Name: "GIMP", Categories: []string{}, DesktopFile: "/var/lib/flatpak/exports/share/applications/org.gimp.GIMP.desktop",
			Package: "org.gimp.GIMP", Source: model.SourceFlatpak, Location: "flatpak"},
		{DesktopID: "dev.zed.Zed", Name: "Zed", Categories: []string{}, DesktopFile: "/home/u/.local/share/applications/dev.zed.Zed.desktop",
			Package: "Zed", Source: model.SourceLocal, SizeBytes: 400, Size: "400 B", Location: "user"},
	}
}

func appsMgrs() []manager.Manager {
	var out []manager.Manager
	for _, s := range []model.Source{model.SourcePacman, model.SourceAUR, model.SourceFlatpak, model.SourceLocal} {
		out = append(out, &fakeManager{name: s, available: true})
	}
	return out
}

func TestAppsJSON(t *testing.T) {
	defer func(f func(bool) []manager.DesktopApp) { loadApps = f }(loadApps)
	var gotNoCache bool
	loadApps = func(noCache bool) []manager.DesktopApp { gotNoCache = noCache; return fixtureApps(noCache) }

	var out, errOut bytes.Buffer
	if code := Dispatch([]string{"apps", "--json", "--no-cache"}, appsMgrs(), "test", &out, &errOut, nil); code != ExitOK {
		t.Fatalf("exit %d (stderr=%q)", code, errOut.String())
	}
	if !gotNoCache {
		t.Error("--no-cache not passed through")
	}
	var env struct {
		Schema int              `json:"schema"`
		Data   []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Schema != 1 || len(env.Data) != 3 {
		t.Fatalf("schema %d, %d rows", env.Schema, len(env.Data))
	}
	ff := env.Data[0]
	for _, k := range []string{"desktop_id", "name", "categories", "no_display", "terminal", "desktop_file", "package", "source", "version", "size_bytes", "location"} {
		if _, ok := ff[k]; !ok {
			t.Errorf("firefox row lacks %q: %v", k, ff)
		}
	}
	gimp := env.Data[1]
	if cats, ok := gimp["categories"].([]any); !ok || len(cats) != 0 {
		t.Errorf("empty categories should encode as [], got %v", gimp["categories"])
	}
	if _, ok := gimp["size_bytes"]; ok {
		t.Errorf("unknown size should be omitted: %v", gimp)
	}
}

func TestAppsManagerFilterAndTerms(t *testing.T) {
	defer func(f func(bool) []manager.DesktopApp) { loadApps = f }(loadApps)
	loadApps = fixtureApps

	var out, errOut bytes.Buffer
	if code := Dispatch([]string{"apps", "--json", "-m", "flatpak,local"}, appsMgrs(), "test", &out, &errOut, nil); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(out.String(), `"firefox"`) || !strings.Contains(out.String(), "org.gimp.GIMP") || !strings.Contains(out.String(), "dev.zed.Zed") {
		t.Errorf("filter output: %s", out.String())
	}

	out.Reset()
	if code := Dispatch([]string{"apps", "zed"}, appsMgrs(), "test", &out, &errOut, nil); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "Zed") || strings.Contains(out.String(), "GIMP") {
		t.Errorf("term filter output: %s", out.String())
	}

	if code := Dispatch([]string{"apps", "-m", "nope"}, appsMgrs(), "test", &out, &errOut, nil); code != ExitErr {
		t.Errorf("unknown manager exit %d", code)
	}
}
