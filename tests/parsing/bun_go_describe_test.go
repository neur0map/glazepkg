package parsing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestBunPackageJSONDescriptionParsing verifies that the "description" field
// is extracted correctly from a bun-installed package.json, matching the logic
// in internal/manager/bun.go:bunLocalDescription.
func TestBunPackageJSONDescriptionParsing(t *testing.T) {
	pkgJSON := `{
  "name": "is-odd",
  "description": "Returns true if the given number is odd, and is an integer.",
  "version": "3.0.1",
  "license": "MIT"
}`
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "is-odd")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(pkgDir, "package.json")
	if err := os.WriteFile(path, []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	got := extractBunDescription(dir, "is-odd")
	want := "Returns true if the given number is odd, and is an integer."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestBunPackageJSONMissingDescription verifies graceful empty-string return
// when the package.json lacks a description field.
func TestBunPackageJSONMissingDescription(t *testing.T) {
	pkgJSON := `{"name": "nope", "version": "1.0.0"}`
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "nope")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := extractBunDescription(dir, "nope"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// TestBunPackageJSONMissingFile verifies graceful empty-string return when
// package.json does not exist for the requested package.
func TestBunPackageJSONMissingFile(t *testing.T) {
	dir := t.TempDir()
	if got := extractBunDescription(dir, "does-not-exist"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// TestBunPackageJSONMalformed verifies graceful handling of invalid JSON.
func TestBunPackageJSONMalformed(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "bad")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := extractBunDescription(dir, "bad"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// extractBunDescription mirrors bunLocalDescription from internal/manager/bun.go
// so this test has no import cycle and stays in the parsing package.
func extractBunDescription(nodeModulesDir, name string) string {
	data, err := os.ReadFile(filepath.Join(nodeModulesDir, name, "package.json"))
	if err != nil {
		return ""
	}
	var meta struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return ""
	}
	return trimSpace(meta.Description)
}

// TestGoVersionModulePathParsing verifies that the main module path is
// extracted correctly from `go version -m` output, matching the logic in
// internal/manager/go.go:goBinaryModulePath.
func TestGoVersionModulePathParsing(t *testing.T) {
	// Output format produced by `go version -m <binary>`:
	//   <path>: go1.21.0
	//   \tpath\tgithub.com/foo/bar/cmd/baz
	//   \tmod\tgithub.com/foo/bar\tv1.2.3\th1:...
	output := "/Users/foo/go/bin/gopls: go1.25.7\n" +
		"\tpath\tgolang.org/x/tools/gopls\n" +
		"\tmod\tgolang.org/x/tools/gopls\tv0.21.1\th1:XYZ=\n" +
		"\tdep\tgithub.com/BurntSushi/toml\tv1.5.0\th1:ABC=\n"

	got := extractGoModulePath(output)
	want := "golang.org/x/tools/gopls"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestGoVersionModulePathMissing verifies that non-Go binary output (no path
// line) returns empty.
func TestGoVersionModulePathMissing(t *testing.T) {
	output := "/some/random/file: not a Go binary\n"
	if got := extractGoModulePath(output); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// extractGoModulePath mirrors the parsing loop in goBinaryModulePath so this
// test stays in the parsing package without importing internal/manager.
func extractGoModulePath(output string) string {
	for _, line := range splitLines(output) {
		fields := splitFields(line)
		if len(fields) >= 2 && fields[0] == "path" {
			return fields[1]
		}
	}
	return ""
}
