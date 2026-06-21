package manager

import "testing"

func TestParseNpmListFiltersBundled(t *testing.T) {
	data := []byte(`{
		"dependencies": {
			"npm": {"version": "10.8.2"},
			"corepack": {"version": "0.29.4"},
			"typescript": {"version": "5.5.4"},
			"prettier": {"version": "3.3.3"}
		}
	}`)

	pkgs, err := parseNpmList(data)
	if err != nil {
		t.Fatalf("parseNpmList: %v", err)
	}

	byName := map[string]string{}
	for _, p := range pkgs {
		byName[p.Name] = p.Version
	}

	if _, ok := byName["npm"]; ok {
		t.Error("npm should be filtered out")
	}
	if _, ok := byName["corepack"]; ok {
		t.Error("corepack should be filtered out")
	}
	if byName["typescript"] != "5.5.4" {
		t.Errorf("typescript: got %q", byName["typescript"])
	}
	if byName["prettier"] != "3.3.3" {
		t.Errorf("prettier: got %q", byName["prettier"])
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages after filtering, got %d", len(pkgs))
	}
}

func TestParseNpmListEmpty(t *testing.T) {
	pkgs, err := parseNpmList([]byte(`{"dependencies": {}}`))
	if err != nil {
		t.Fatalf("parseNpmList: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(pkgs))
	}
}
