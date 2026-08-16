package manager

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Correctness: the batched query must agree with asking one at a time, and must
// not mark an unowned path owned because a longer sibling path is owned.
func TestOwnedByPackageMatchesPerPathProbe(t *testing.T) {
	if _, err := exec.LookPath("pacman"); err != nil {
		t.Skip("needs pacman")
	}
	unowned := filepath.Join(t.TempDir(), "not-a-package-file")
	if err := os.WriteFile(unowned, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	paths := []string{"/usr/bin/ls", "/usr/bin/lsblk", "/usr/bin/cp", unowned}

	got := ownedByPackage(paths)
	for _, p := range paths {
		want := exec.Command("pacman", "-Qo", p).Run() == nil
		if got[p] != want {
			t.Errorf("%s: batched=%v per-path=%v", p, got[p], want)
		}
	}
	if got[unowned] {
		t.Error("a file no package owns must not be reported owned")
	}
}

// The substring trap: /usr/bin/ls is a prefix of /usr/bin/lsblk.
func TestOwnedByPackageNoPrefixFalsePositive(t *testing.T) {
	if _, err := exec.LookPath("pacman"); err != nil {
		t.Skip("needs pacman")
	}
	dir := t.TempDir()
	short := filepath.Join(dir, "tool")
	long := filepath.Join(dir, "toolbox")
	for _, p := range []string{short, long} {
		if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got := ownedByPackage([]string{short, long})
	if got[short] || got[long] {
		t.Errorf("neither temp file is packaged, got %v", got)
	}
}

func TestPacmanOwnedPath(t *testing.T) {
	if got := pacmanOwnedPath("/usr/bin/ls is owned by coreutils 9.11-2"); got != "/usr/bin/ls" {
		t.Errorf("got %q", got)
	}
	if got := pacmanOwnedPath("error: No package owns /tmp/x"); got != "" {
		t.Errorf("error line parsed as a path: %q", got)
	}
}

func TestDpkgOwnedPath(t *testing.T) {
	cases := map[string]string{
		"coreutils: /usr/bin/ls":        "/usr/bin/ls",
		"libc-bin, libc6: /usr/bin/foo": "/usr/bin/foo",
		"garbage":                       "",
	}
	for line, want := range cases {
		if got := dpkgOwnedPath(line); got != want {
			t.Errorf("dpkgOwnedPath(%q) = %q, want %q", line, got, want)
		}
	}
}

// Scale: the number of subprocesses must not grow with the number of paths.
// Three hundred paths is a realistic /usr/local/bin on a CI runner.
func TestOwnedByPackageScales(t *testing.T) {
	if _, err := exec.LookPath("pacman"); err != nil {
		t.Skip("needs pacman")
	}
	entries, err := os.ReadDir("/usr/bin")
	if err != nil {
		t.Skip(err)
	}
	var paths []string
	for _, e := range entries {
		if len(paths) == 300 {
			break
		}
		if !e.IsDir() {
			paths = append(paths, filepath.Join("/usr/bin", e.Name()))
		}
	}

	start := time.Now()
	got := ownedByPackage(paths)
	batched := time.Since(start)

	start = time.Now()
	for _, p := range paths[:20] {
		exec.Command("pacman", "-Qo", p).Run()
	}
	perPath20 := time.Since(start)
	projected := perPath20 * time.Duration(len(paths)) / 20

	t.Logf("%d paths: batched %v, one at a time projected %v (%d owned)",
		len(paths), batched, projected, len(got))
	if batched > projected/4 {
		t.Errorf("batching saved too little: %v vs projected %v", batched, projected)
	}
}
