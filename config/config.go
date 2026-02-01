package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

const appName = "igcMailImap"

// Config holds application settings (saved to a single JSON file).
type Config struct {
	IMAPServer           string `json:"imap_server"` // host:port, e.g. "imap.gmail.com:993"
	IMAPUser             string `json:"imap_user"`
	IMAPPassword         string `json:"imap_password"`
	OutputFolder         string `json:"output_folder"`    // local folder for extracted IGC files
	IntervalSec          int    `json:"interval_seconds"` // poll every N seconds
	RunAtStartup         bool   `json:"run_at_startup"`
	PollingEnabled       bool   `json:"polling_enabled"`       // if true, polling runs at launch and stays on until Stop
	LoggingEnabled       bool   `json:"logging_enabled"`       // if true, logging is enabled
	NotificationsEnabled bool   `json:"notifications_enabled"` // if true, desktop notifications are enabled
}

// Default returns a config with sensible defaults (Gmail IMAP, 61s interval, polling on).
func Default() *Config {
	return &Config{
		IMAPServer:           "imap.gmail.com:993",
		OutputFolder:         "",
		IntervalSec:          60,
		RunAtStartup:         false,
		PollingEnabled:       true,
		LoggingEnabled:       false,
		NotificationsEnabled: true,
	}
}

// configDir returns the directory for config and state files (OS-specific).
func configDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", appName), nil
	case "windows":
		exe, err := os.Executable()
		if err != nil {
			return "", err
		}
		return filepath.Dir(exe), nil
	default:
		// Linux and others: use XDG_CONFIG_HOME or ~/.config
		if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
			return filepath.Join(dir, appName), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", appName), nil
	}
}

// ConfigPath returns the path to the config file.
func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// StatePath returns the path to the state file (last UID for incremental fetch).
func StatePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

// Load reads config from the JSON file. If the file does not exist, returns Default() and nil error.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}

	// Use a map to detect which fields are present in the JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	// Set defaults for missing fields (for backward compatibility)
	if c.IntervalSec <= 0 {
		c.IntervalSec = 60
	}

	// If NotificationsEnabled field is not present in JSON, set it to default (true)
	if _, exists := raw["notifications_enabled"]; !exists {
		c.NotificationsEnabled = true
	}

	return &c, nil
}

// Save writes config to the JSON file. Creates the config directory if needed (macOS/Linux).
func Save(c *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
