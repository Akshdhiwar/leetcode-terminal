package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds non-sensitive settings only.
// Session and CSRF are NOT stored here — they live in the OS keychain
// via the keyring package.
type Config struct {
	Username     string `json:"username"`
	Language     string `json:"language"`
	SolutionsDir string `json:"solutions_dir"`
	Editor       string `json:"editor"`

	// Runtime-only fields (populated from keyring, never written to disk)
	Session string `json:"-"`
	CSRF    string `json:"-"`
}

// SolutionsDirResolved returns the configured path or the default.
func (c *Config) SolutionsDirResolved() (string, error) {
	if c.SolutionsDir != "" {
		if len(c.SolutionsDir) >= 2 && c.SolutionsDir[:2] == "~/" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(home, c.SolutionsDir[2:]), nil
		}
		return c.SolutionsDir, nil
	}
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "solutions"), nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".leetcode-cli")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return &Config{Language: "cpp"}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Language: "cpp"}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &Config{Language: "cpp"}, nil
	}
	if cfg.Language == "" {
		cfg.Language = "cpp"
	}
	return &cfg, nil
}

// Save writes only non-sensitive settings to disk.
// Credentials must be saved separately via keyring.Save().
func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Marshal only the safe fields (Session/CSRF tagged json:"-" so excluded)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func ConfigFilePath() (string, error) {
	return configPath()
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".leetcode-cli")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create config dir: %w", err)
	}
	return dir, nil
}
