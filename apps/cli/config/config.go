package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ConfigFileName = "shuttl.json"

// Config represents the structure of shuttl.json
type Config struct {
	App string `json:"app"`
}

// LoadConfig looks for shuttl.json in the current directory and parent directories
func LoadConfig() (*Config, error) {
	configPath, err := FindConfigFile()
	if err != nil {
		return nil, err
	}

	return LoadConfigFromPath(configPath)
}

// LoadConfigFromPath loads configuration from a specific file path
func LoadConfigFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// FindConfigFile searches for shuttl.json in the current directory and parent directories
func FindConfigFile() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	return FindConfigFileFrom(cwd)
}

// FindConfigFileFrom searches for shuttl.json starting from the given directory
func FindConfigFileFrom(startDir string) (string, error) {
	dir := startDir

	for {
		configPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root directory
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("%s not found in %s or any parent directory", ConfigFileName, startDir)
}

// GetConfigDir returns the directory containing the config file
func GetConfigDir(configPath string) string {
	return filepath.Dir(configPath)
}












