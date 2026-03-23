package theme

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config stores the user's persistent theming preferences.
// Saved to ~/.glazepkg/config.toml.
type Config struct {
	// ActiveTheme is the display name of the last-selected theme.
	// Matches a key in the theme registry (i.e. the TOML 'name' field).
	ActiveTheme string `toml:"active_theme"`
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := glazepkgDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// LoadConfig reads and parses ~/.glazepkg/config.toml.
// Returns an empty Config (not an error) when the file does not yet exist;
// the caller is responsible for applying a sensible default.
func LoadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, fmt.Errorf("theme: resolving config path: %w", err)
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// First launch — no config yet.  This is expected and not an error.
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("theme: reading config: %w", err)
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return Config{}, fmt.Errorf("theme: parsing config: %w", err)
	}
	return cfg, nil
}

// SaveConfig writes cfg to ~/.glazepkg/config.toml atomically (write to temp
// file, then rename) so a crash mid-write cannot corrupt the config.
func SaveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return fmt.Errorf("theme: resolving config path: %w", err)
	}

	// Write to a sibling temp file first.
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("theme: opening temp config: %w", err)
	}

	// Emit a human-readable header comment so users can see what the file is.
	_, _ = fmt.Fprintln(f, "# GlazePKG theme configuration")
	_, _ = fmt.Fprintln(f, "# Edit 'active_theme' to change the startup theme.")
	_, _ = fmt.Fprintln(f, "# Available themes: run `gpk themes` to list them.")
	_, _ = fmt.Fprintln(f)

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("theme: encoding config: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("theme: closing temp config: %w", err)
	}

	// Atomic rename.
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("theme: saving config: %w", err)
	}
	return nil
}
