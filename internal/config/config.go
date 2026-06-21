package config

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Appearance AppearanceConfig `toml:"appearance"`
	Install    InstallConfig    `toml:"install"`
}

type AppearanceConfig struct {
	Theme string `toml:"theme"`
}

// InstallConfig holds install-time preferences. Prefer lists manager names in
// priority order; when a package is available in several, gpk picks the
// highest-ranked one instead of asking.
type InstallConfig struct {
	Prefer []string `toml:"prefer"`
}

func configDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "glazepkg")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "glazepkg")
}

func configPath() string {
	return filepath.Join(configDir(), "config.toml")
}

// UserThemesDir returns the path to the user's custom themes directory.
func UserThemesDir() string {
	return filepath.Join(configDir(), "themes")
}

// Load reads the config file, returning defaults if it doesn't exist.
func Load() Config {
	cfg := Config{
		Appearance: AppearanceConfig{
			Theme: "tokyo-night",
		},
	}
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	_ = toml.Unmarshal(data, &cfg)
	return cfg
}

// Save writes the config to disk, creating directories as needed.
func Save(cfg Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, configPath())
}
