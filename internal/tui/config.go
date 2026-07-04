package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigFile holds user preferences persisted to disk.
type ConfigFile struct {
	Theme ThemeName `json:"theme"`
}

const configFileName = "config.json"

// ConfigDir returns the directory used for xtop user configuration.
func ConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "xtop")
}

// ConfigPath returns the full path to the config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), configFileName)
}

// LoadConfig reads the persisted config file, returning defaults on error.
func LoadConfig() ConfigFile {
	var cfg ConfigFile
	b, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(b, &cfg)
	if cfg.Theme == "" {
		cfg.Theme = ThemeDark
	}
	return cfg
}

// SaveConfig persists the config file to disk.
func SaveConfig(cfg ConfigFile) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(ConfigPath(), b, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
