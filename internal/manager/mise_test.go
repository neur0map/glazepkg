package manager

import "testing"

func TestParseMiseList(t *testing.T) {
	data := []byte(`{
  "jq": [
    { "version": "1.7.1", "install_path": "/home/u/.local/share/mise/installs/jq/1.7.1", "installed": true, "active": false },
    { "version": "1.8.1", "install_path": "/home/u/.local/share/mise/installs/jq/1.8.1", "installed": true, "active": true }
  ],
  "node": [
    { "version": "20.18.1", "install_path": "/home/u/.local/share/mise/installs/node/20.18.1", "installed": true, "active": true }
  ]
}`)
	pkgs, err := parseMiseList(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	// sorted by name: jq, node
	if pkgs[0].Name != "jq" || pkgs[1].Name != "node" {
		t.Fatalf("not sorted by name: %q, %q", pkgs[0].Name, pkgs[1].Name)
	}
	if pkgs[0].Version != "1.8.1" {
		t.Errorf("jq: active version should win, got %q", pkgs[0].Version)
	}
	if pkgs[0].Location != "/home/u/.local/share/mise/installs/jq/1.8.1" {
		t.Errorf("jq: unexpected location %q", pkgs[0].Location)
	}
	if pkgs[1].Version != "20.18.1" {
		t.Errorf("node: got version %q", pkgs[1].Version)
	}
}

func TestParseMiseListEmpty(t *testing.T) {
	pkgs, err := parseMiseList([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParseMiseListLastInstalledFallback(t *testing.T) {
	data := []byte(`{
  "go": [
    { "version": "1.21.0", "install_path": "/p/go/1.21.0", "installed": true, "active": false },
    { "version": "1.22.0", "install_path": "/p/go/1.22.0", "installed": true, "active": false }
  ]
}`)
	pkgs, err := parseMiseList(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Version != "1.22.0" {
		t.Errorf("expected last installed version 1.22.0, got %q", pkgs[0].Version)
	}
}

func TestParseMiseListSkipNoneInstalled(t *testing.T) {
	data := []byte(`{
  "python": [
    { "version": "3.12.0", "install_path": "", "installed": false, "active": false }
  ]
}`)
	pkgs, err := parseMiseList(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected tool with no installed versions to be skipped, got %d", len(pkgs))
	}
}

func TestParseMiseOutdated(t *testing.T) {
	data := []byte(`{ "node": { "name": "node", "requested": "20", "current": "20.18.1", "latest": "20.18.2", "bump": "20.18.2" } }`)
	got := parseMiseOutdated(data)
	if got["node"] != "20.18.2" {
		t.Errorf("node: got %q, want 20.18.2", got["node"])
	}

	empty := parseMiseOutdated([]byte(`{}`))
	if len(empty) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(empty))
	}
}
