package parsing

import (
	"encoding/json"
	"testing"
)

func TestBrewOutdatedJSON(t *testing.T) {
	data := `{
		"formulae": [
			{
				"name": "git",
				"installed_versions": ["2.44.0"],
				"current_version": "2.45.0",
				"pinned": false,
				"pinned_version": null
			},
			{
				"name": "curl",
				"installed_versions": ["8.6.0"],
				"current_version": "8.7.1",
				"pinned": false,
				"pinned_version": null
			}
		],
		"casks": []
	}`

	var outdated struct {
		Formulae []struct {
			Name              string   `json:"name"`
			CurrentVersion    string   `json:"current_version"`
			InstalledVersions []string `json:"installed_versions"`
		} `json:"formulae"`
	}
	if err := json.Unmarshal([]byte(data), &outdated); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(outdated.Formulae) != 2 {
		t.Fatalf("expected 2 formulae, got %d", len(outdated.Formulae))
	}
	if outdated.Formulae[0].Name != "git" {
		t.Errorf("expected name=git, got %q", outdated.Formulae[0].Name)
	}
	if outdated.Formulae[0].CurrentVersion != "2.45.0" {
		t.Errorf("expected current_version=2.45.0, got %q", outdated.Formulae[0].CurrentVersion)
	}
	if outdated.Formulae[0].InstalledVersions[0] != "2.44.0" {
		t.Errorf("expected installed_versions[0]=2.44.0, got %q", outdated.Formulae[0].InstalledVersions[0])
	}
}

func TestBrewOutdatedFlatArrayFails(t *testing.T) {
	// brew outdated --json returns an object, not a flat array
	data := `{"formulae": [], "casks": []}`

	var arr []struct {
		Name           string `json:"name"`
		CurrentVersion string `json:"current_version"`
	}
	err := json.Unmarshal([]byte(data), &arr)
	if err == nil {
		t.Error("flat array parse should fail on object input")
	}
}
