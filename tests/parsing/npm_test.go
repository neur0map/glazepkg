package parsing

import (
	"encoding/json"
	"testing"
)

func TestNpmListJSON(t *testing.T) {
	data := `{
		"dependencies": {
			"typescript": {"version": "5.3.3"},
			"prettier": {"version": "3.2.4"},
			"eslint": {"version": "8.56.0"}
		}
	}`

	var result struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(result.Dependencies) != 3 {
		t.Fatalf("expected 3 deps, got %d", len(result.Dependencies))
	}
	if result.Dependencies["typescript"].Version != "5.3.3" {
		t.Errorf("typescript: got %q", result.Dependencies["typescript"].Version)
	}
}

func TestNpmOutdatedJSON(t *testing.T) {
	data := `{
		"typescript": {"current": "5.3.3", "wanted": "5.3.3", "latest": "5.4.0", "location": ""},
		"prettier": {"current": "3.2.4", "wanted": "3.2.5", "latest": "3.2.5", "location": ""}
	}`

	var outdated map[string]struct {
		Latest string `json:"latest"`
	}
	if err := json.Unmarshal([]byte(data), &outdated); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if outdated["typescript"].Latest != "5.4.0" {
		t.Errorf("typescript: got %q", outdated["typescript"].Latest)
	}
	if outdated["prettier"].Latest != "3.2.5" {
		t.Errorf("prettier: got %q", outdated["prettier"].Latest)
	}
}
