package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	UserConfigDirName  = "shuttl"
	UserConfigFileName = "config.jsonc"
	DefaultAPIURL      = "dashboard.shuttl.io"
)

// UserConfig represents the global user configuration stored in ~/.config/shuttl/config.jsonc
type UserConfig struct {
	APIURL string `json:"api_url"`
}

// DefaultUserConfig returns a UserConfig with default values
func DefaultUserConfig() *UserConfig {
	return &UserConfig{
		APIURL: DefaultAPIURL,
	}
}

// GetUserConfigDir returns the path to the user's shuttl config directory
// Cross-platform: uses os.UserConfigDir() which returns:
// - Linux: $XDG_CONFIG_HOME or $HOME/.config
// - macOS: $HOME/Library/Application Support
// - Windows: %AppData%
func GetUserConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, UserConfigDirName), nil
}

// GetUserConfigPath returns the full path to the user config file
func GetUserConfigPath() (string, error) {
	configDir, err := GetUserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, UserConfigFileName), nil
}

// LoadUserConfig loads the user configuration from the config file
// If the file doesn't exist, returns default configuration
func LoadUserConfig() (*UserConfig, error) {
	configPath, err := GetUserConfigPath()
	if err != nil {
		return DefaultUserConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultUserConfig(), nil
		}
		return nil, fmt.Errorf("failed to read user config file: %w", err)
	}

	// Strip JSONC comments and trailing commas before parsing
	jsonData := stripJSONComments(string(data))
	jsonData = stripTrailingCommas(jsonData)

	var config UserConfig
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		return nil, fmt.Errorf("failed to parse user config file: %w", err)
	}

	// Apply defaults for any missing values
	if config.APIURL == "" {
		config.APIURL = DefaultAPIURL
	}

	return &config, nil
}

// SaveUserConfig saves the user configuration to the config file
func SaveUserConfig(config *UserConfig) error {
	configPath, err := GetUserConfigPath()
	if err != nil {
		return err
	}

	// Ensure the config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create JSONC content with comments
	content := fmt.Sprintf(`{
  // Shuttl CLI Configuration
  // API URL for the Shuttl dashboard
  "api_url": %q
}
`, config.APIURL)

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write user config file: %w", err)
	}

	return nil
}

// EnsureUserConfig ensures that the user config file exists
// If it doesn't exist, creates it with default values
func EnsureUserConfig() (*UserConfig, error) {
	configPath, err := GetUserConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		config := DefaultUserConfig()
		if err := SaveUserConfig(config); err != nil {
			return nil, err
		}
		return config, nil
	}

	return LoadUserConfig()
}

// stripJSONComments removes single-line (//) and multi-line (/* */) comments from JSONC
func stripJSONComments(input string) string {
	// Handle strings properly - we don't want to strip "comments" inside strings
	var result strings.Builder
	inString := false
	inSingleLineComment := false
	inMultiLineComment := false
	i := 0

	for i < len(input) {
		// Check for string boundaries
		if !inSingleLineComment && !inMultiLineComment {
			if input[i] == '"' && (i == 0 || input[i-1] != '\\') {
				inString = !inString
				result.WriteByte(input[i])
				i++
				continue
			}
		}

		// Inside a string, just copy
		if inString {
			result.WriteByte(input[i])
			i++
			continue
		}

		// Check for single-line comment start
		if !inMultiLineComment && i+1 < len(input) && input[i] == '/' && input[i+1] == '/' {
			inSingleLineComment = true
			i += 2
			continue
		}

		// Check for single-line comment end
		if inSingleLineComment {
			if input[i] == '\n' {
				inSingleLineComment = false
				result.WriteByte(input[i])
			}
			i++
			continue
		}

		// Check for multi-line comment start
		if !inSingleLineComment && i+1 < len(input) && input[i] == '/' && input[i+1] == '*' {
			inMultiLineComment = true
			i += 2
			continue
		}

		// Check for multi-line comment end
		if inMultiLineComment {
			if i+1 < len(input) && input[i] == '*' && input[i+1] == '/' {
				inMultiLineComment = false
				i += 2
				continue
			}
			i++
			continue
		}

		// Regular character
		result.WriteByte(input[i])
		i++
	}

	return result.String()
}

// stripTrailingCommas removes trailing commas from JSON (another JSONC feature)
func stripTrailingCommas(input string) string {
	// Match trailing commas before } or ]
	re := regexp.MustCompile(`,\s*([\]\}])`)
	return re.ReplaceAllString(input, "$1")
}

// GetAPIURL returns the full API URL with https:// prefix
func (c *UserConfig) GetAPIURL() string {
	apiURL := c.APIURL
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	// Add https:// if not present
	if !strings.HasPrefix(apiURL, "http://") && !strings.HasPrefix(apiURL, "https://") {
		return "https://" + apiURL
	}
	return apiURL
}
