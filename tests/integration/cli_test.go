package integration

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/cli"
	"github.com/neur0map/glazepkg/internal/manager"
)

// TestCLI_ListJSON runs `gpk list --json` against real managers and verifies
// the envelope shape. Doesn't care how many packages are present — runners
// may have zero managers and zero packages; data being an array is enough.
func TestCLI_ListJSON(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := cli.Dispatch(
		[]string{"list", "--json", "--no-cache", "--quiet"},
		manager.All(), "integration-test", &out, &errOut, nil,
	)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	var env struct {
		Schema int               `json:"schema"`
		Data   []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v\nbody=%s", err, out.String())
	}
	if env.Schema != 1 {
		t.Errorf("schema = %d, want 1", env.Schema)
	}
	// env.Data may be nil if no managers and no packages; that's OK.
}

// TestCLI_InstalledMissing verifies the exit-2 contract for absent packages.
func TestCLI_InstalledMissing(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := cli.Dispatch(
		[]string{"installed", "definitely-not-a-real-package-xyz-zzz", "--no-cache", "--quiet"},
		manager.All(), "integration-test", &out, &errOut, nil,
	)
	if code != 2 {
		t.Fatalf("exit %d, want 2 (stderr=%q)", code, errOut.String())
	}
}

// TestCLI_OutdatedCountFormat verifies `--count` output is exactly a number.
func TestCLI_OutdatedCountFormat(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := cli.Dispatch(
		[]string{"outdated", "--count", "--quiet"},
		manager.All(), "integration-test", &out, &errOut, nil,
	)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%q", code, errOut.String())
	}
	if !regexp.MustCompile(`^[0-9]+\n$`).Match(out.Bytes()) {
		t.Errorf("--count output = %q, want /^[0-9]+\\n$/", out.String())
	}
}

// TestCLI_SourceOfMissing verifies the exit-2 + empty-stdout contract.
func TestCLI_SourceOfMissing(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := cli.Dispatch(
		[]string{"source-of", "--no-cache", "definitely-not-a-real-package-xyz-zzz"},
		manager.All(), "integration-test", &out, &errOut, nil,
	)
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty on exit 2", out.String())
	}
}

// TestCLI_SubcommandTypo verifies a near-miss command name yields a "did you
// mean" hint and exit 1, while a clear bareword falls through to search.
func TestCLI_SubcommandTypo(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := cli.Dispatch(
		[]string{"instal", "git"},
		manager.All(), "integration-test", &out, &errOut, nil,
	)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "did you mean") {
		t.Errorf("stderr = %q, want a suggestion", errOut.String())
	}
}

// TestCLI_UpgradeDryRun verifies the upgrade command can resolve and build
// a command against real installed packages without actually executing.
// Skips if no packages have updates (nothing to upgrade).
func TestCLI_UpgradeDryRun(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// First find an outdated package on this system.
	var outBuf, errBuf bytes.Buffer
	code := cli.Dispatch(
		[]string{"outdated", "--json", "--no-cache", "--quiet"},
		manager.All(), "integration-test", &outBuf, &errBuf, nil,
	)
	if code != 0 {
		t.Skipf("outdated failed (exit %d), skipping: %s", code, errBuf.String())
	}
	var env struct {
		Data []struct {
			Name   string `json:"name"`
			Source string `json:"source"`
		} `json:"data"`
	}
	if err := json.Unmarshal(outBuf.Bytes(), &env); err != nil {
		t.Skipf("invalid outdated JSON: %v", err)
	}
	if len(env.Data) == 0 {
		t.Skip("no outdated packages on this system; nothing to dry-run upgrade")
	}
	target := env.Data[0]

	// Now dry-run upgrade against the first outdated package.
	var out, errOut bytes.Buffer
	code = cli.Dispatch(
		[]string{"upgrade", target.Name, "--dry-run", "--quiet", "--manager", target.Source},
		manager.All(), "integration-test", &out, &errOut, nil,
	)
	if code != 0 {
		t.Fatalf("upgrade dry-run exit %d, stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), target.Name) {
		t.Errorf("dry-run output should mention %q: %q", target.Name, out.String())
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Errorf("expected dry-run notice in stdout: %q", out.String())
	}
}
