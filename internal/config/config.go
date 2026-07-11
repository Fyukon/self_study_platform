package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const directory = "learning-roadmap"

type Config struct {
	DatabasePath    string `json:"database_path"`
	BackupPath      string `json:"backup_path"`
	ApplicationName string `json:"application_name"`
	Timezone        string `json:"timezone"`
	Theme           string `json:"theme"`
	WeekStartsOn    int    `json:"week_starts_on"`
}

func Load() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("find home directory: %w", err)
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}

	configDir := filepath.Join(configHome, directory)
	dataDir := filepath.Join(dataHome, directory)
	defaults := Config{
		DatabasePath:    filepath.Join(dataDir, "learning-roadmap.db"),
		BackupPath:      filepath.Join(dataDir, "backups"),
		ApplicationName: "Learning Roadmap",
		Timezone:        "Europe/Moscow",
		Theme:           "system",
		WeekStartsOn:    1,
	}
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return Config{}, fmt.Errorf("create config directory: %w", err)
	}
	path := filepath.Join(configDir, "config.json")
	cfg := defaults
	data, err := os.ReadFile(path)
	newFile := os.IsNotExist(err)
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("decode config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	fillDefaults(&cfg, defaults)
	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o700); err != nil {
		return Config{}, fmt.Errorf("create data directory: %w", err)
	}
	if err := os.MkdirAll(cfg.BackupPath, 0o700); err != nil {
		return Config{}, fmt.Errorf("create backup directory: %w", err)
	}
	if newFile {
		if err := save(path, cfg); err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}

func fillDefaults(cfg *Config, defaults Config) {
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = defaults.DatabasePath
	}
	if cfg.BackupPath == "" {
		cfg.BackupPath = defaults.BackupPath
	}
	if cfg.ApplicationName == "" {
		cfg.ApplicationName = defaults.ApplicationName
	}
	if cfg.Timezone == "" {
		cfg.Timezone = defaults.Timezone
	}
	if cfg.Theme == "" {
		cfg.Theme = defaults.Theme
	}
	if cfg.WeekStartsOn == 0 {
		cfg.WeekStartsOn = defaults.WeekStartsOn
	}
}

func save(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("set config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("install config: %w", err)
	}
	return nil
}
