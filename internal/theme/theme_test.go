package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

// resetRegistry clears global state between tests so they're independent.
func resetRegistry() {
	registry = map[string]Theme{}
	order = nil
	activeTheme = Theme{}
}

// ─── parseTheme ───────────────────────────────────────────────────────────────

func TestParseTheme_Valid(t *testing.T) {
	toml := `
name        = "My Theme"
type        = "dark"
author      = "tester"
description = "A test theme"
background  = "#1a1b26"
foreground  = "#c0caf5"
accent      = "#7aa2f7"
cursor      = "#ff9e64"
highlight   = "#9d7cd8"
border      = "#414868"
selection   = "#33467c"
surface     = "#3b4261"
subtext     = "#565f89"
text        = "#a9b1d6"
blue        = "#7aa2f7"
purple      = "#bb9af7"
green       = "#9ece6a"
red         = "#f7768e"
yellow      = "#e0af68"
cyan        = "#7dcfff"
orange      = "#ff9e64"
white       = "#c6c6df"
custom_key  = "#abcdef"
`
	th, ok := parseTheme([]byte(toml), "test.toml")
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if th.Name != "My Theme" {
		t.Errorf("name = %q, want %q", th.Name, "My Theme")
	}
	if th.Background != "#1a1b26" {
		t.Errorf("background = %q", th.Background)
	}
	if th.Extra["custom_key"] != "#abcdef" {
		t.Errorf("extra custom_key = %q, want #abcdef", th.Extra["custom_key"])
	}
}

func TestParseTheme_MissingName(t *testing.T) {
	toml := `background = "#000000"`
	_, ok := parseTheme([]byte(toml), "test.toml")
	if ok {
		t.Error("expected parse to fail for theme without name")
	}
}

func TestParseTheme_InvalidTOML(t *testing.T) {
	_, ok := parseTheme([]byte(`not = valid toml [[[[`), "bad.toml")
	if ok {
		t.Error("expected parse to fail for invalid TOML")
	}
}

// ─── Theme.Color() ────────────────────────────────────────────────────────────

func TestThemeColor_KnownKeys(t *testing.T) {
	th := Theme{
		Background: "#111111",
		Foreground: "#222222",
		Accent:     "#333333",
		Blue:       "#4444ff",
	}
	cases := map[string]string{
		"background": "#111111",
		"foreground": "#222222",
		"accent":     "#333333",
		"blue":       "#4444ff",
	}
	for key, want := range cases {
		if got := th.Color(key); got != want {
			t.Errorf("Color(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestThemeColor_ExtraKey(t *testing.T) {
	th := Theme{Extra: map[string]string{"my_color": "#abcdef"}}
	if got := th.Color("my_color"); got != "#abcdef" {
		t.Errorf("Color(extra) = %q, want #abcdef", got)
	}
}

func TestThemeColor_MissingFallback(t *testing.T) {
	th := Theme{}
	got := th.Color("nonexistent")
	if got != "#888888" {
		t.Errorf("Color(missing) = %q, want #888888", got)
	}
}

func TestThemeIsLight(t *testing.T) {
	if (Theme{Type: "light"}).IsLight() != true {
		t.Error("light theme not detected")
	}
	if (Theme{Type: "dark"}).IsLight() != false {
		t.Error("dark theme incorrectly identified as light")
	}
	if (Theme{}).IsLight() != false {
		t.Error("empty type theme incorrectly identified as light")
	}
}

// ─── Registry / Load ─────────────────────────────────────────────────────────

func TestLoad_BundledThemes(t *testing.T) {
	resetRegistry()
	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	names := ListThemes()
	if len(names) == 0 {
		t.Fatal("expected at least one bundled theme after Load()")
	}
	// Tokyo Night must be present (it's the default).
	found := false
	for _, n := range names {
		if n == "Tokyo Night" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Tokyo Night not found in bundled themes; got: %v", names)
	}
}

func TestGetTheme_KnownAndUnknown(t *testing.T) {
	resetRegistry()
	_ = Load()

	_, ok := GetTheme("Tokyo Night")
	if !ok {
		t.Error("GetTheme: Tokyo Night not found after Load()")
	}
	_, ok = GetTheme("does not exist xyz")
	if ok {
		t.Error("GetTheme: expected false for nonexistent theme")
	}
}

// ─── SetActive / CycleNext / CyclePrev ───────────────────────────────────────

func TestSetActive_Valid(t *testing.T) {
	resetRegistry()
	_ = Load()

	// Use a temp dir so we don't touch real ~/.glazepkg.
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	names := ListThemes()
	if len(names) == 0 {
		t.Skip("no themes loaded")
	}
	target := names[len(names)-1] // pick the last one (not the default)
	th, err := SetActive(target)
	if err != nil {
		t.Fatalf("SetActive(%q): %v", target, err)
	}
	if th.Name != target {
		t.Errorf("SetActive returned theme %q, want %q", th.Name, target)
	}
	if Active().Name != target {
		t.Errorf("Active() = %q after SetActive, want %q", Active().Name, target)
	}
}

func TestSetActive_Unknown(t *testing.T) {
	resetRegistry()
	_ = Load()
	_, err := SetActive("__does_not_exist__")
	if err == nil {
		t.Error("expected error for unknown theme name")
	}
}

func TestCycleNext_WrapsAround(t *testing.T) {
	resetRegistry()
	_ = Load()

	dir := t.TempDir()
	t.Setenv("HOME", dir)

	names := ListThemes()
	n := len(names)
	if n < 2 {
		t.Skip("need at least 2 themes to test cycling")
	}

	// Force to last theme, then CycleNext should wrap to first.
	_, _ = SetActive(names[n-1])
	next := CycleNext()
	if next.Name != names[0] {
		t.Errorf("CycleNext from last: got %q, want %q", next.Name, names[0])
	}
}

func TestCyclePrev_WrapsAround(t *testing.T) {
	resetRegistry()
	_ = Load()

	dir := t.TempDir()
	t.Setenv("HOME", dir)

	names := ListThemes()
	n := len(names)
	if n < 2 {
		t.Skip("need at least 2 themes to test cycling")
	}

	// Force to first theme, then CyclePrev should wrap to last.
	_, _ = SetActive(names[0])
	prev := CyclePrev()
	if prev.Name != names[n-1] {
		t.Errorf("CyclePrev from first: got %q, want %q", prev.Name, names[n-1])
	}
}

// ─── Snapshots ───────────────────────────────────────────────────────────────

func TestSaveAndLoadSnapshot(t *testing.T) {
	resetRegistry()
	_ = Load()

	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Apply a known theme so we have something to snapshot.
	_, _ = SetActive("Tokyo Night")
	activeTheme.Description = "snapshot test"

	if err := SaveSnapshot("test-snap"); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	// Confirm the snapshot file exists.
	snapDir := filepath.Join(dir, ".glazepkg", "theme_snapshots")
	entries, err := os.ReadDir(snapDir)
	if err != nil {
		t.Fatalf("reading snapshot dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one snapshot file")
	}

	// Now switch theme and load the snapshot back.
	names := ListThemes()
	if len(names) > 1 {
		_, _ = SetActive(names[1])
	}

	snapPath := filepath.Join(snapDir, entries[0].Name())
	if err := LoadSnapshot(snapPath); err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}

	// Active theme should now be the snapshot variant.
	if !strings.Contains(Active().Name, "test-snap") {
		t.Errorf("after LoadSnapshot Active().Name = %q, expected it to contain 'test-snap'", Active().Name)
	}
}

func TestListSnapshots_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	paths, err := ListSnapshots()
	if err != nil {
		t.Fatalf("ListSnapshots on empty dir: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(paths))
	}
}

// ─── User theme override ──────────────────────────────────────────────────────

func TestUserThemeOverridesBundled(t *testing.T) {
	resetRegistry()

	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Write a user theme with the same name as a bundled theme to test override.
	themesDir := filepath.Join(dir, ".glazepkg", "themes")
	_ = os.MkdirAll(themesDir, 0750)
	userTOML := `
name       = "Tokyo Night"
type       = "dark"
background = "#ff0000"
foreground = "#00ff00"
text       = "#0000ff"
`
	_ = os.WriteFile(filepath.Join(themesDir, "my-tokyo.toml"), []byte(userTOML), 0600)

	if err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	th, ok := GetTheme("Tokyo Night")
	if !ok {
		t.Fatal("Tokyo Night not found after Load with user override")
	}
	// User theme should win.
	if th.Background != "#ff0000" {
		t.Errorf("user override not applied: background = %q, want #ff0000", th.Background)
	}
}

// ─── Config persistence ───────────────────────────────────────────────────────

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := Config{ActiveTheme: "Dracula"}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.ActiveTheme != "Dracula" {
		t.Errorf("loaded ActiveTheme = %q, want Dracula", loaded.ActiveTheme)
	}
}

func TestLoadConfig_Missing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig on missing file should not error: %v", err)
	}
	if cfg.ActiveTheme != "" {
		t.Errorf("expected empty ActiveTheme for missing config, got %q", cfg.ActiveTheme)
	}
}

// ─── Fallback theme ───────────────────────────────────────────────────────────

func TestFallbackTheme_NeverPanics(t *testing.T) {
	fb := fallbackTheme()
	if fb.Name == "" {
		t.Error("fallbackTheme has empty Name")
	}
	if fb.Background == "" {
		t.Error("fallbackTheme has empty Background")
	}
}

func TestApplyByName_UnknownFallsToFirst(t *testing.T) {
	resetRegistry()
	_ = loadFromFS(bundledFS, "themes")
	buildOrder()

	applyByName("__nonexistent_theme__")
	if Active().Name == "" {
		t.Error("applyByName with unknown name should fall back to first theme, not empty")
	}
}
