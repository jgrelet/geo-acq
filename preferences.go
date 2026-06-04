package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jgrelet/geo-acq/config"
)

const preferencesFileName = "preferences.json"

type preferences struct {
	LastConfigPath string `json:"last_config_path"`
}

func startupConfigPath() string {
	path, err := loadLastConfigPath()
	if err == nil && strings.TrimSpace(path) != "" {
		if _, statErr := os.Stat(path); statErr == nil {
			return path
		}
	}
	return config.DefaultFile()
}

func loadLastConfigPath() (string, error) {
	path, err := preferencesFilePath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var prefs preferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return "", fmt.Errorf("decode preferences: %w", err)
	}
	return strings.TrimSpace(prefs.LastConfigPath), nil
}

func saveLastConfigPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	prefsPath, err := preferencesFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(prefsPath), 0o755); err != nil {
		return fmt.Errorf("create preferences directory: %w", err)
	}

	data, err := json.MarshalIndent(preferences{LastConfigPath: path}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode preferences: %w", err)
	}
	if err := os.WriteFile(prefsPath, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write preferences: %w", err)
	}
	return nil
}

func preferencesFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(dir) == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil || strings.TrimSpace(home) == "" {
			if err != nil {
				return "", fmt.Errorf("locate user config directory: %w", err)
			}
			return "", errors.New("locate user config directory")
		}
		dir = filepath.Join(home, ".geo-acq")
	} else {
		dir = filepath.Join(dir, "geo-acq")
	}
	return filepath.Join(dir, preferencesFileName), nil
}
