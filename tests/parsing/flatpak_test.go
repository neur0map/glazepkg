package parsing

import (
	"bufio"
	"strings"
	"testing"
)

func TestFlatpakListParsing(t *testing.T) {
	// flatpak list --app --columns=application,version
	output := `com.spotify.Client	1.2.30.688
org.mozilla.firefox	122.0.1
org.gimp.GIMP	2.10.36`

	type pkg struct{ name, version string }
	var pkgs []pkg
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		pkgs = append(pkgs, pkg{parts[0], parts[1]})
	}

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 flatpaks, got %d", len(pkgs))
	}
	if pkgs[0].name != "com.spotify.Client" || pkgs[0].version != "1.2.30.688" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[2].name != "org.gimp.GIMP" || pkgs[2].version != "2.10.36" {
		t.Errorf("pkg 2: %+v", pkgs[2])
	}
}
