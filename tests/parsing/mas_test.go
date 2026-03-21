package parsing

import (
	"strings"
	"testing"
)

func TestMasListParsing(t *testing.T) {
	output := `497799835  Xcode            (15.4)
640199958  Developer        (10.6.5)
1295203466 Microsoft Remote Desktop (10.9.5)`

	type app struct{ name, version string }
	var apps []app
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		parenIdx := strings.LastIndex(line, "(")
		if parenIdx < 0 {
			continue
		}
		version := strings.TrimSuffix(strings.TrimSpace(line[parenIdx+1:]), ")")
		prefix := strings.TrimSpace(line[:parenIdx])
		fields := strings.Fields(prefix)
		if len(fields) < 2 {
			continue
		}
		name := strings.Join(fields[1:], " ")
		apps = append(apps, app{name, version})
	}

	if len(apps) != 3 {
		t.Fatalf("expected 3 apps, got %d", len(apps))
	}
	if apps[0].name != "Xcode" || apps[0].version != "15.4" {
		t.Errorf("app 0: %+v", apps[0])
	}
	if apps[2].name != "Microsoft Remote Desktop" || apps[2].version != "10.9.5" {
		t.Errorf("app 2: %+v", apps[2])
	}
}

func TestMasOutdatedParsing(t *testing.T) {
	output := `497799835  Xcode            (15.4 -> 16.0)
640199958  Developer        (10.6.5 -> 10.6.6)`

	updates := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		parenIdx := strings.LastIndex(line, "(")
		if parenIdx < 0 {
			continue
		}
		prefix := strings.TrimSpace(line[:parenIdx])
		fields := strings.Fields(prefix)
		if len(fields) < 2 {
			continue
		}
		name := strings.Join(fields[1:], " ")
		inner := strings.TrimSuffix(strings.TrimSpace(line[parenIdx+1:]), ")")
		parts := strings.Split(inner, " -> ")
		if len(parts) == 2 {
			updates[name] = strings.TrimSpace(parts[1])
		}
	}
	if updates["Xcode"] != "16.0" {
		t.Errorf("Xcode: got %q", updates["Xcode"])
	}
	if updates["Developer"] != "10.6.6" {
		t.Errorf("Developer: got %q", updates["Developer"])
	}
}
