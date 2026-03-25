package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Appearance AppearanceConfig `toml:"appearance"`
}

type AppearanceConfig struct {
	Theme string `toml:"theme"`
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
	f, err := os.Create(configPath())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
