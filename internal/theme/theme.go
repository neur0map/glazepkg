// Package theme provides a production-ready theming system for glazepkg.
//
// Themes are defined as TOML files and can come from two sources:
//  1. Bundled defaults: embedded at compile time via go:embed (zero runtime deps)
//  2. User-defined:     ~/.glazepkg/themes/*.toml (override / extend bundled set)
//
// The active theme is persisted in ~/.glazepkg/config.toml and applied to the
// TUI at startup and whenever the user cycles or selects a theme.
package theme

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// ─── Embedded defaults ────────────────────────────────────────────────────────

// bundledFS holds all TOML files under internal/theme/themes/.
// go:embed resolves at compile time; the binary therefore ships with every
// default theme without requiring runtime filesystem access.
//
//go:embed themes/*.toml
var bundledFS embed.FS

// ─── Core types ───────────────────────────────────────────────────────────────

// Theme represents a complete color palette.
// Core keys (Background, Foreground, …) have dedicated fields so callers can
// reference them without a string lookup.  The Extra map absorbs any additional
// key the theme author defines, keeping the format fully open-ended.
type Theme struct {
	// Metadata
	Name        string `toml:"name"`
	Type        string `toml:"type"`        // "dark" | "light" | ""
	Author      string `toml:"author"`      // optional
	Description string `toml:"description"` // optional

	// Core palette — every well-formed theme should define these.
	Background string `toml:"background"`
	Foreground string `toml:"foreground"`
	Accent     string `toml:"accent"`
	Cursor     string `toml:"cursor"`
	Highlight  string `toml:"highlight"`
	Border     string `toml:"border"`
	Selection  string `toml:"selection"`
	Surface    string `toml:"surface"`
	Subtext    string `toml:"subtext"`
	Text       string `toml:"text"`

	// Semantic accent colors
	Blue   string `toml:"blue"`
	Purple string `toml:"purple"`
	Green  string `toml:"green"`
	Red    string `toml:"red"`
	Yellow string `toml:"yellow"`
	Cyan   string `toml:"cyan"`
	Orange string `toml:"orange"`
	White  string `toml:"white"`

	// Extra holds arbitrary color keys not listed above.
	// Populated during TOML decode via a secondary raw-map pass.
	Extra map[string]string `toml:"-"`
}

// FillDefaults ensures that every core color field has a non-empty value by
// falling back to the Tokyo Night palette for any missing keys.  This prevents
// broken TUI rendering when a user-defined theme is incomplete.
func (t *Theme) FillDefaults() {
	def := fallbackTheme()
	if t.Background == "" {
		t.Background = def.Background
	}
	if t.Foreground == "" {
		t.Foreground = def.Foreground
	}
	if t.Accent == "" {
		t.Accent = def.Accent
	}
	if t.Cursor == "" {
		t.Cursor = def.Cursor
	}
	if t.Highlight == "" {
		t.Highlight = def.Highlight
	}
	if t.Border == "" {
		t.Border = def.Border
	}
	if t.Selection == "" {
		t.Selection = def.Selection
	}
	if t.Surface == "" {
		t.Surface = def.Surface
	}
	if t.Subtext == "" {
		t.Subtext = def.Subtext
	}
	if t.Text == "" {
		t.Text = def.Text
	}
	if t.Blue == "" {
		t.Blue = def.Blue
	}
	if t.Purple == "" {
		t.Purple = def.Purple
	}
	if t.Green == "" {
		t.Green = def.Green
	}
	if t.Red == "" {
		t.Red = def.Red
	}
	if t.Yellow == "" {
		t.Yellow = def.Yellow
	}
	if t.Cyan == "" {
		t.Cyan = def.Cyan
	}
	if t.Orange == "" {
		t.Orange = def.Orange
	}
	if t.White == "" {
		t.White = def.White
	}
}

// Color returns the hex string for a named color key (e.g. "blue", "surface").
// Falls back to Extra, then returns a safe default so callers never get an
// empty string for a valid theme key.
func (t Theme) Color(key string) string {
	switch strings.ToLower(key) {
	case "background":
		return t.orDefault(t.Background)
	case "foreground":
		return t.orDefault(t.Foreground)
	case "accent":
		return t.orDefault(t.Accent)
	case "cursor":
		return t.orDefault(t.Cursor)
	case "highlight":
		return t.orDefault(t.Highlight)
	case "border":
		return t.orDefault(t.Border)
	case "selection":
		return t.orDefault(t.Selection)
	case "surface":
		return t.orDefault(t.Surface)
	case "subtext":
		return t.orDefault(t.Subtext)
	case "text":
		return t.orDefault(t.Text)
	case "blue":
		return t.orDefault(t.Blue)
	case "purple":
		return t.orDefault(t.Purple)
	case "green":
		return t.orDefault(t.Green)
	case "red":
		return t.orDefault(t.Red)
	case "yellow":
		return t.orDefault(t.Yellow)
	case "cyan":
		return t.orDefault(t.Cyan)
	case "orange":
		return t.orDefault(t.Orange)
	case "white":
		return t.orDefault(t.White)
	}
	if v, ok := t.Extra[key]; ok && v != "" {
		return v
	}
	return "#888888"
}

func (t Theme) orDefault(v string) string {
	if v != "" {
		return v
	}
	return "#888888"
}

// IsLight reports whether this is a light-background theme.
func (t Theme) IsLight() bool {
	return strings.ToLower(t.Type) == "light"
}

// ─── Theme Snapshot ───────────────────────────────────────────────────────────

// Snapshot preserves a named copy of a theme at a point in time.
// Stored as TOML under ~/.glazepkg/theme_snapshots/<name>_<timestamp>.toml.
type Snapshot struct {
	Name        string    `toml:"name"`
	SavedAt     time.Time `toml:"saved_at"`
	SourceTheme string    `toml:"source_theme"`
	Theme       Theme     `toml:"theme"`
}

// ─── Registry ─────────────────────────────────────────────────────────────────

// registry is the in-memory map of all loaded themes.  Guarded by no mutex
// because it is written once at startup and read-only thereafter (writes only
// happen through SetActive which also calls loadUserThemes under no lock — this
// is fine because the TUI is single-threaded via bubbletea's event loop).
var registry = map[string]Theme{}

// order preserves the display order: bundled themes first, user themes last.
var order []string

// activeTheme is the currently applied theme.
var activeTheme Theme

// ─── Initialisation ───────────────────────────────────────────────────────────

// Load initialises the theme system:
//  1. Parses all bundled themes from the embedded FS.
//  2. Merges user themes from ~/.glazepkg/themes/ (user themes win on name clash).
//  3. Reads the active theme name from ~/.glazepkg/config.toml and applies it.
//  4. Falls back to "Tokyo Night" if the config names an unknown theme.
//
// Call Load() once at program startup before calling any other function.
func Load() error {
	// 1. Bundled themes
	if err := loadFromFS(bundledFS, "themes"); err != nil {
		return fmt.Errorf("theme: loading bundled themes: %w", err)
	}

	// 2. User themes (silently skip missing dir)
	userDir, err := userThemesDir()
	if err == nil {
		_ = loadFromDir(userDir) // warnings logged inside; never fatal
	}

	// Freeze display order
	buildOrder()

	// 3. Apply active theme from config
	cfg, err := LoadConfig()
	if err != nil {
		// Config missing/corrupt — apply default and continue.
		log.Printf("theme: config error (%v); using default theme", err)
		applyByName(defaultThemeName())
		return nil
	}

	if cfg.ActiveTheme != "" {
		applyByName(cfg.ActiveTheme)
	} else {
		applyByName(defaultThemeName())
	}

	return nil
}

// loadFromFS walks an embed.FS rooted at dir and loads all *.toml files.
func loadFromFS(efs embed.FS, dir string) error {
	return fs.WalkDir(efs, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".toml") {
			return nil
		}
		data, err := efs.ReadFile(path)
		if err != nil {
			log.Printf("theme: skipping embedded file %s: %v", path, err)
			return nil
		}
		if t, ok := parseTheme(data, path); ok {
			registry[t.Name] = t
		}
		return nil
	})
}

// loadFromDir loads all *.toml files from a filesystem directory.
func loadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("theme: skipping user theme %s: %v", path, err)
			continue
		}
		if t, ok := parseTheme(data, path); ok {
			registry[t.Name] = t
			log.Printf("theme: loaded user theme %q from %s", t.Name, path)
		}
	}
	return nil
}

// parseTheme decodes a TOML byte slice into a Theme.  Invalid files are
// warned and skipped (ok=false); callers should not treat this as fatal.
func parseTheme(data []byte, src string) (Theme, bool) {
	// First decode into the typed struct for well-known fields.
	var t Theme
	if _, err := toml.Decode(string(data), &t); err != nil {
		log.Printf("theme: invalid TOML in %s: %v", src, err)
		return Theme{}, false
	}
	if t.Name == "" {
		log.Printf("theme: %s has no 'name' field — skipping", src)
		return Theme{}, false
	}

	// Apply sensible defaults for any fields the theme author omitted.
	t.FillDefaults()

	// Second pass: capture arbitrary extra keys not covered by the struct.
	var raw map[string]interface{}
	if _, err := toml.Decode(string(data), &raw); err == nil {
		knownKeys := map[string]bool{
			"name": true, "type": true, "author": true, "description": true,
			"background": true, "foreground": true, "accent": true, "cursor": true,
			"highlight": true, "border": true, "selection": true, "surface": true,
			"subtext": true, "text": true, "blue": true, "purple": true,
			"green": true, "red": true, "yellow": true, "cyan": true,
			"orange": true, "white": true,
		}
		extra := map[string]string{}
		for k, v := range raw {
			if !knownKeys[k] {
				if s, ok := v.(string); ok {
					extra[k] = s
				}
			}
		}
		if len(extra) > 0 {
			t.Extra = extra
		}
	}

	return t, true
}

// buildOrder constructs the ordered name list: alphabetical within each tier.
func buildOrder() {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	order = names
}

// applyByName sets the active theme; falls back to the first available theme
// if name is not found, and to a hard-coded fallback if the registry is empty.
func applyByName(name string) {
	if t, ok := registry[name]; ok {
		activeTheme = t
		return
	}
	if len(order) > 0 {
		activeTheme = registry[order[0]]
		return
	}
	// Ultimate fallback — bare-minimum Tokyo Night so the UI never breaks.
	activeTheme = fallbackTheme()
}

// ─── Public API ───────────────────────────────────────────────────────────────

// Active returns the currently applied Theme.
func Active() Theme {
	return activeTheme
}

// ListThemes returns all available theme names in display order.
func ListThemes() []string {
	out := make([]string, len(order))
	copy(out, order)
	return out
}

// GetTheme returns the theme registered under name, plus a bool indicating
// whether it was found.
func GetTheme(name string) (Theme, bool) {
	t, ok := registry[name]
	return t, ok
}

// ActiveIndex returns the index of the active theme in ListThemes().
func ActiveIndex() int {
	for i, name := range order {
		if name == activeTheme.Name {
			return i
		}
	}
	return 0
}

// SetActive applies the named theme, persists the choice to config, and
// returns the new active Theme so callers can immediately re-style the TUI.
// Returns an error if the name is unknown or config save fails; the theme
// is still applied in-memory so the TUI remains usable.
func SetActive(name string) (Theme, error) {
	t, ok := registry[name]
	if !ok {
		return activeTheme, fmt.Errorf("theme %q not found", name)
	}
	activeTheme = t

	cfg, err := LoadConfig()
	if err != nil {
		cfg = Config{}
	}
	cfg.ActiveTheme = name
	if saveErr := SaveConfig(cfg); saveErr != nil {
		return activeTheme, fmt.Errorf("theme: saving config: %w", saveErr)
	}
	return activeTheme, nil
}

// CycleNext advances to the next theme in display order and persists the
// change.  Returns the new active Theme.
func CycleNext() Theme {
	if len(order) == 0 {
		return activeTheme
	}
	idx := (ActiveIndex() + 1) % len(order)
	name := order[idx]
	t, _ := SetActive(name) //nolint:errcheck — best-effort persist
	return t
}

// CyclePrev moves to the previous theme in display order and persists.
func CyclePrev() Theme {
	if len(order) == 0 {
		return activeTheme
	}
	idx := ActiveIndex() - 1
	if idx < 0 {
		idx = len(order) - 1
	}
	name := order[idx]
	t, _ := SetActive(name)
	return t
}

// ─── Snapshots ────────────────────────────────────────────────────────────────

// SaveSnapshot captures the active theme under a user-supplied label and writes
// it to ~/.glazepkg/theme_snapshots/<name>_<timestamp>.toml.
func SaveSnapshot(label string) error {
	dir, err := snapshotDir()
	if err != nil {
		return err
	}
	ts := time.Now().Format("20060102T150405")
	safe := strings.ReplaceAll(label, " ", "_")
	filename := fmt.Sprintf("%s_%s.toml", safe, ts)
	path := filepath.Join(dir, filename)

	snap := Snapshot{
		Name:        label,
		SavedAt:     time.Now(),
		SourceTheme: activeTheme.Name,
		Theme:       activeTheme,
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(snap); err != nil {
		return fmt.Errorf("theme: encoding snapshot: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("theme: writing snapshot %s: %w", path, err)
	}
	log.Printf("theme: snapshot saved to %s", path)
	return nil
}

// LoadSnapshot reads a snapshot TOML file and applies its embedded Theme,
// persisting it as the active theme.  The snapshot's theme name is used as the
// new active name.
func LoadSnapshot(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("theme: reading snapshot %s: %w", path, err)
	}
	var snap Snapshot
	if _, err := toml.Decode(string(data), &snap); err != nil {
		return fmt.Errorf("theme: decoding snapshot %s: %w", path, err)
	}
	// Register the snapshot's theme so it participates in the registry.
	snapName := fmt.Sprintf("%s (snapshot)", snap.Name)
	snap.Theme.Name = snapName
	registry[snapName] = snap.Theme

	buildOrder()
	activeTheme = snap.Theme

	// Persist the choice.
	cfg, _ := LoadConfig()
	cfg.ActiveTheme = snapName
	_ = SaveConfig(cfg)

	return nil
}

// ListSnapshots returns paths of all theme snapshot files, newest first.
func ListSnapshots() ([]string, error) {
	dir, err := snapshotDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".toml") {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	// Reverse so newest (lexicographically last timestamp) appears first.
	for i, j := 0, len(paths)-1; i < j; i, j = i+1, j-1 {
		paths[i], paths[j] = paths[j], paths[i]
	}
	return paths, nil
}

// ─── Directory helpers ────────────────────────────────────────────────────────

func glazepkgDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".glazepkg")
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", err
	}
	return dir, nil
}

func userThemesDir() (string, error) {
	base, err := glazepkgDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "themes")
	// Do not create if missing — absence is valid; user just has no custom themes.
	return dir, nil
}

func snapshotDir() (string, error) {
	base, err := glazepkgDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "theme_snapshots")
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", err
	}
	return dir, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func defaultThemeName() string {
	if _, ok := registry["Tokyo Night"]; ok {
		return "Tokyo Night"
	}
	if len(order) > 0 {
		return order[0]
	}
	return ""
}

// fallbackTheme returns a hard-coded minimal theme so the TUI never crashes
// even if no TOML files are available at all.
func fallbackTheme() Theme {
	return Theme{
		Name:       "Tokyo Night",
		Type:       "dark",
		Background: "#1a1b26",
		Foreground: "#c0caf5",
		Accent:     "#7aa2f7",
		Cursor:     "#ff9e64",
		Highlight:  "#9d7cd8",
		Border:     "#414868",
		Selection:  "#33467c",
		Surface:    "#3b4261",
		Subtext:    "#565f89",
		Text:       "#a9b1d6",
		Blue:       "#7aa2f7",
		Purple:     "#bb9af7",
		Green:      "#9ece6a",
		Red:        "#f7768e",
		Yellow:     "#e0af68",
		Cyan:       "#7dcfff",
		Orange:     "#ff9e64",
		White:      "#c6c6df",
	}
}
