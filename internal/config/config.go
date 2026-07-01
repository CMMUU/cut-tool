// Package config persists user-configurable settings to a JSON file in the
// per-app config directory (the same directory as the history database).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DefaultRetentionDays is how long an unpinned entry is kept before the
// cleanup pass deletes it. Pinned entries are never expired.
const DefaultRetentionDays = 15

const fileName = "config.json"

// Config holds user-adjustable settings.
type Config struct {
	// RetentionDays is the age (in days) after which unpinned entries are
	// deleted. Values <= 0 are treated as DefaultRetentionDays on load.
	RetentionDays int `json:"retention_days"`

	// HotkeyMods is the set of modifier names (e.g. "ctrl", "shift", "cmd",
	// "alt") for the global show-history shortcut. Empty falls back to the
	// platform default supplied by the caller.
	HotkeyMods []string `json:"hotkey_mods"`

	// HotkeyKey is the main key name (e.g. "V", "C", "1") for the shortcut.
	// Empty falls back to the platform default.
	HotkeyKey string `json:"hotkey_key"`
}

// Default returns the built-in configuration. Hotkey fields are left empty
// so the caller (which knows the platform-appropriate default) can fill them
// via SetHotkeyDefaults.
func Default() Config {
	return Config{RetentionDays: DefaultRetentionDays}
}

// Store reads and writes the config file, guarding access with a mutex so it
// is safe to Save from the UI while a background loop reads it.
type Store struct {
	mu   sync.RWMutex
	path string
	cfg  Config
}

// Open loads the config from the user config dir, creating it with defaults
// if it does not exist yet.
func Open() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("config: locate user config dir: %w", err)
	}
	appDir := filepath.Join(dir, "cut-tool")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return nil, fmt.Errorf("config: mkdir %s: %w", appDir, err)
	}
	return OpenAt(filepath.Join(appDir, fileName))
}

// OpenAt loads the config from an explicit path (useful in tests).
func OpenAt(path string) (*Store, error) {
	s := &Store{path: path, cfg: Default()}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// First run: persist defaults so the file exists.
			if werr := s.save(); werr != nil {
				return nil, werr
			}
			return s, nil
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &s.cfg); err != nil {
		// Corrupt file: fall back to defaults rather than failing startup.
		s.cfg = Default()
	}
	s.cfg.normalize()
	return s, nil
}

// Get returns a copy of the current config.
func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// SetRetentionDays updates the retention period and persists it.
func (s *Store) SetRetentionDays(days int) error {
	s.mu.Lock()
	s.cfg.RetentionDays = days
	s.cfg.normalize()
	err := s.save()
	s.mu.Unlock()
	return err
}

// SetHotkey updates the hotkey combination and persists it.
func (s *Store) SetHotkey(mods []string, key string) error {
	s.mu.Lock()
	s.cfg.HotkeyMods = mods
	s.cfg.HotkeyKey = key
	err := s.save()
	s.mu.Unlock()
	return err
}

// save writes the current config to disk. Caller must hold the lock.
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("config: write %s: %w", s.path, err)
	}
	return nil
}

// normalize clamps out-of-range values to sane defaults.
func (c *Config) normalize() {
	if c.RetentionDays <= 0 {
		c.RetentionDays = DefaultRetentionDays
	}
}
