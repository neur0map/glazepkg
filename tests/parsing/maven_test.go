package parsing

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/neur0map/glazepkg/internal/manager"
)

func TestMavenScan(t *testing.T) {
	tmpDir := t.TempDir()

	artifacts := []struct {
		relPath string
		content string
	}{
		{"org/apache/commons/commons-lang3/3.14.0/commons-lang3-3.14.0.pom", "<project/>"},
		{"org/apache/commons/commons-lang3/3.14.0/commons-lang3-3.14.0.jar", "fake-jar"},
		{"com/google/guava/guava/33.0.0-jre/guava-33.0.0-jre.pom", "<project/>"},
		{"io/netty/netty-all/4.1.107.Final/netty-all-4.1.107.Final.pom", "<project/>"},
		// Older version of commons-lang3 — should be deduplicated
		{"org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.pom", "<project/>"},
		// Same artifactId from different groupIds — names should be qualified
		{"org/sonatype/aether/aether-impl/1.7/aether-impl-1.7.pom", "<project/>"},
		{"org/eclipse/aether/aether-impl/1.0.0/aether-impl-1.0.0.pom", "<project/>"},
	}

	newer := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	older := newer.Add(-24 * time.Hour)

	for _, a := range artifacts {
		full := filepath.Join(tmpDir, a.relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(a.content), 0o644); err != nil {
			t.Fatal(err)
		}
		// Make 3.12.0 clearly older so deduplication picks 3.14.0
		ts := newer
		if filepath.Base(filepath.Dir(full)) == "3.12.0" {
			ts = older
		}
		os.Chtimes(full, ts, ts)
	}

	t.Setenv("MAVEN_REPO_LOCAL", tmpDir)

	m := &manager.Maven{}
	if !m.Available() {
		t.Fatal("expected Maven to be available with MAVEN_REPO_LOCAL set")
	}

	pkgs, err := m.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(pkgs) != 5 {
		t.Fatalf("expected 5 unique artifacts, got %d", len(pkgs))
	}

	type expected struct {
		version     string
		description string
	}
	want := map[string]expected{
		"commons-lang3": {"3.14.0", "org.apache.commons:commons-lang3"},
		"guava":         {"33.0.0-jre", "com.google.guava:guava"},
		"netty-all":     {"4.1.107.Final", "io.netty:netty-all"},
		// Colliding artifactIds get qualified with groupId
		"org.sonatype.aether:aether-impl": {"1.7", "org.sonatype.aether:aether-impl"},
		"org.eclipse.aether:aether-impl":  {"1.0.0", "org.eclipse.aether:aether-impl"},
	}

	byName := make(map[string]struct{ version, description string })
	for _, p := range pkgs {
		byName[p.Name] = struct{ version, description string }{p.Version, p.Description}
	}

	for name, w := range want {
		got, ok := byName[name]
		if !ok {
			t.Errorf("missing artifact: %s", name)
			continue
		}
		if got.version != w.version {
			t.Errorf("%s: want version %s, got %s", name, w.version, got.version)
		}
		if got.description != w.description {
			t.Errorf("%s: want description %s, got %s", name, w.description, got.description)
		}
	}
}
