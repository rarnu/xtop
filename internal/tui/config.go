package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// xtopHome returns the single directory used for all user-specific xtop data:
// configuration, language files, themes and the process cache.
func xtopHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".xtop")
}

// ConfigFile holds user preferences persisted to disk.
type ConfigFile struct {
	Theme ThemeName `json:"theme"`
}

const configFileName = "config.json"

// ConfigPath returns the full path to the config file.
func ConfigPath() string {
	return filepath.Join(xtopHome(), configFileName)
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
	dir := xtopHome()
	if dir == "" {
		return fmt.Errorf("cannot determine xtop home directory")
	}
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
