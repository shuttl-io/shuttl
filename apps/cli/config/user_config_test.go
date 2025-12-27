package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultUserConfig(t *testing.T) {
	config := DefaultUserConfig()

	if config.APIURL != DefaultAPIURL {
		t.Errorf("Expected APIURL to be '%s', got '%s'", DefaultAPIURL, config.APIURL)
	}
}

func TestUserConfigGetAPIURL(t *testing.T) {
	testCases := []struct {
		name     string
		apiURL   string
		expected string
	}{
		{
			name:     "empty URL uses default",
			apiURL:   "",
			expected: "https://" + DefaultAPIURL,
		},
		{
			name:     "URL without protocol gets https",
			apiURL:   "api.example.com",
			expected: "https://api.example.com",
		},
		{
			name:     "URL with https stays unchanged",
			apiURL:   "https://secure.example.com",
			expected: "https://secure.example.com",
		},
		{
			name:     "URL with http stays unchanged",
			apiURL:   "http://localhost:8080",
			expected: "http://localhost:8080",
		},
		{
			name:     "complex URL without protocol",
			apiURL:   "api.example.com:3000/v1",
			expected: "https://api.example.com:3000/v1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &UserConfig{APIURL: tc.apiURL}
			result := config.GetAPIURL()
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestStripJSONComments(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no comments",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "single line comment",
			input:    "{\n// this is a comment\n\"key\": \"value\"\n}",
			expected: "{\n\n\"key\": \"value\"\n}",
		},
		{
			name:     "multi-line comment",
			input:    "{\n/* this is\na multi-line\ncomment */\n\"key\": \"value\"\n}",
			expected: "{\n\n\"key\": \"value\"\n}",
		},
		{
			name:     "comment in string should not be stripped",
			input:    `{"url": "http://example.com"}`,
			expected: `{"url": "http://example.com"}`,
		},
		{
			name:     "double slash in string should not be stripped",
			input:    `{"path": "//network/share"}`,
			expected: `{"path": "//network/share"}`,
		},
		{
			name:     "mixed comments",
			input:    "{\n// comment 1\n\"key1\": \"value1\",\n/* comment 2 */\n\"key2\": \"value2\"\n}",
			expected: "{\n\n\"key1\": \"value1\",\n\n\"key2\": \"value2\"\n}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := stripJSONComments(tc.input)
			if result != tc.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tc.expected, result)
			}
		})
	}
}

func TestStripTrailingCommas(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no trailing commas",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "trailing comma before }",
			input:    `{"key": "value",}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "trailing comma before ]",
			input:    `["a", "b", "c",]`,
			expected: `["a", "b", "c"]`,
		},
		{
			name:     "trailing comma with whitespace",
			input:    `{"key": "value" ,  }`,
			expected: `{"key": "value" }`,
		},
		{
			name:     "nested trailing commas",
			input:    `{"arr": [1, 2,], "obj": {"a": 1,},}`,
			expected: `{"arr": [1, 2], "obj": {"a": 1}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := stripTrailingCommas(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestLoadUserConfig(t *testing.T) {
	// Save original HOME and restore after test
	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("HOME", origHome)
		if origXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	tmpDir, err := os.MkdirTemp("", "user_config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set XDG_CONFIG_HOME to our temp directory
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	t.Run("returns default when file doesn't exist", func(t *testing.T) {
		config, err := LoadUserConfig()
		if err != nil {
			t.Fatalf("LoadUserConfig failed: %v", err)
		}

		if config.APIURL != DefaultAPIURL {
			t.Errorf("Expected default APIURL '%s', got '%s'", DefaultAPIURL, config.APIURL)
		}
	})

	t.Run("loads valid JSONC config", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, UserConfigDirName)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		configPath := filepath.Join(configDir, UserConfigFileName)
		content := `{
  // This is a comment
  "api_url": "custom.api.com"
}`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		config, err := LoadUserConfig()
		if err != nil {
			t.Fatalf("LoadUserConfig failed: %v", err)
		}

		if config.APIURL != "custom.api.com" {
			t.Errorf("Expected APIURL 'custom.api.com', got '%s'", config.APIURL)
		}
	})

	t.Run("applies default for empty APIURL", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, UserConfigDirName)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		configPath := filepath.Join(configDir, UserConfigFileName)
		content := `{"api_url": ""}`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		config, err := LoadUserConfig()
		if err != nil {
			t.Fatalf("LoadUserConfig failed: %v", err)
		}

		if config.APIURL != DefaultAPIURL {
			t.Errorf("Expected default APIURL '%s', got '%s'", DefaultAPIURL, config.APIURL)
		}
	})
}

func TestSaveUserConfig(t *testing.T) {
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	tmpDir, err := os.MkdirTemp("", "save_user_config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	t.Run("creates config directory and file", func(t *testing.T) {
		config := &UserConfig{APIURL: "test.api.com"}
		err := SaveUserConfig(config)
		if err != nil {
			t.Fatalf("SaveUserConfig failed: %v", err)
		}

		configPath := filepath.Join(tmpDir, UserConfigDirName, UserConfigFileName)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}

		// Verify content
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}

		content := string(data)
		if content == "" {
			t.Error("Config file is empty")
		}
	})

	t.Run("saved config can be loaded back", func(t *testing.T) {
		config := &UserConfig{APIURL: "roundtrip.api.com"}
		err := SaveUserConfig(config)
		if err != nil {
			t.Fatalf("SaveUserConfig failed: %v", err)
		}

		loaded, err := LoadUserConfig()
		if err != nil {
			t.Fatalf("LoadUserConfig failed: %v", err)
		}

		if loaded.APIURL != "roundtrip.api.com" {
			t.Errorf("Expected APIURL 'roundtrip.api.com', got '%s'", loaded.APIURL)
		}
	})
}

func TestEnsureUserConfig(t *testing.T) {
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	t.Run("creates default config when not exists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "ensure_config_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		os.Setenv("XDG_CONFIG_HOME", tmpDir)

		config, err := EnsureUserConfig()
		if err != nil {
			t.Fatalf("EnsureUserConfig failed: %v", err)
		}

		if config.APIURL != DefaultAPIURL {
			t.Errorf("Expected default APIURL '%s', got '%s'", DefaultAPIURL, config.APIURL)
		}

		// Verify file was created
		configPath := filepath.Join(tmpDir, UserConfigDirName, UserConfigFileName)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}
	})

	t.Run("loads existing config", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "ensure_config_existing_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		os.Setenv("XDG_CONFIG_HOME", tmpDir)

		// Create existing config
		configDir := filepath.Join(tmpDir, UserConfigDirName)
		os.MkdirAll(configDir, 0755)
		configPath := filepath.Join(configDir, UserConfigFileName)
		content := `{"api_url": "existing.api.com"}`
		os.WriteFile(configPath, []byte(content), 0644)

		config, err := EnsureUserConfig()
		if err != nil {
			t.Fatalf("EnsureUserConfig failed: %v", err)
		}

		if config.APIURL != "existing.api.com" {
			t.Errorf("Expected APIURL 'existing.api.com', got '%s'", config.APIURL)
		}
	})
}

func TestConstants(t *testing.T) {
	if UserConfigDirName != "shuttl" {
		t.Errorf("Expected UserConfigDirName to be 'shuttl', got '%s'", UserConfigDirName)
	}

	if UserConfigFileName != "config.jsonc" {
		t.Errorf("Expected UserConfigFileName to be 'config.jsonc', got '%s'", UserConfigFileName)
	}

	if DefaultAPIURL != "dashboard.shuttl.io" {
		t.Errorf("Expected DefaultAPIURL to be 'dashboard.shuttl.io', got '%s'", DefaultAPIURL)
	}
}

