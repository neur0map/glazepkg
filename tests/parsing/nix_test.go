package parsing

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// mirrors parseNixStorePath
func parseNixStorePath(storePath string) (string, string) {
	base := filepath.Base(storePath)
	if idx := strings.Index(base, "-"); idx >= 0 {
		base = base[idx+1:]
	} else {
		return base, ""
	}
	return splitNixNameVersion(base)
}

// mirrors splitNixNameVersion
func splitNixNameVersion(s string) (string, string) {
	for i := len(s) - 1; i > 0; i-- {
		if s[i] == '-' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

func TestNixEnvListParsing(t *testing.T) {
	output := `firefox-122.0
git-2.43.0
htop-3.3.0
python3-3.11.7`

	type pkg struct{ name, version string }
	var pkgs []pkg
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, ver := splitNixNameVersion(line)
		pkgs = append(pkgs, pkg{name, ver})
	}

	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "firefox" || pkgs[0].version != "122.0" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[3].name != "python3" || pkgs[3].version != "3.11.7" {
		t.Errorf("pkg 3: %+v", pkgs[3])
	}
}

func TestNixStorePathParsing(t *testing.T) {
	tests := []struct {
		path        string
		wantName    string
		wantVersion string
	}{
		{"/nix/store/r7if10kgajw3wccdj5ci-firefox-122.0", "firefox", "122.0"},
		{"/nix/store/abc123def456-git-2.43.0", "git", "2.43.0"},
		{"/nix/store/xyz789-glibc-2.38-44", "glibc-2.38", "44"},
		{"/nix/store/hash-bash-interactive-5.2-p26", "bash-interactive", "5.2-p26"},
	}
	for _, tt := range tests {
		name, ver := parseNixStorePath(tt.path)
		if name != tt.wantName || ver != tt.wantVersion {
			t.Errorf("parseNixStorePath(%q) = (%q, %q), want (%q, %q)",
				tt.path, name, ver, tt.wantName, tt.wantVersion)
		}
	}
}

func TestNixProfileJSON(t *testing.T) {
	// Simulated nix profile list --json output
	data := `{"elements":{"firefox":{"storePaths":["/nix/store/abc-firefox-122.0"]},"git":{"storePaths":["/nix/store/def-git-2.43.0"]}}}`

	// Parse the store paths the same way the manager does
	type elem struct {
		StorePaths []string `json:"storePaths"`
	}
	type result struct {
		Elements map[string]elem `json:"elements"`
	}

	var r result
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, e := range r.Elements {
		for _, sp := range e.StorePaths {
			name, _ := parseNixStorePath(sp)
			names = append(names, name)
		}
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(names))
	}
	// Order is non-deterministic from map
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["firefox"] || !found["git"] {
		t.Errorf("missing expected packages: %v", names)
	}
}
