// Package config persists ai-mesh settings (API keys) to a JSON file under the
// user's config directory, so keys do not have to live in the shell environment.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the on-disk settings file.
type Config struct {
	// Keys maps a provider name (e.g. "fal", "meshy") to its API key.
	Keys map[string]string `json:"keys"`
}

// userConfigDir is indirected so tests can point it at a temp directory.
var userConfigDir = os.UserConfigDir

// Path returns the config file location (e.g. ~/.config/ai-mesh/config.json).
func Path() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating config dir: %w", err)
	}
	return filepath.Join(dir, "ai-mesh", "config.json"), nil
}

// Load reads the config file. A missing file is not an error; it returns an
// empty config.
func Load() (*Config, error) {
	cfg := &Config{Keys: map[string]string{}}
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	if cfg.Keys == nil {
		cfg.Keys = map[string]string{}
	}
	return cfg, nil
}

// Save writes the config file with 0600 permissions (it holds secrets).
func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// ResolveKey returns the API key using precedence: flag > env var > config file.
func ResolveKey(flagVal, envVal, cfgVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if envVal != "" {
		return envVal
	}
	return cfgVal
}
