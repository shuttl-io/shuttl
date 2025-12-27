package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromPath(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("valid config file", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "valid_shuttl.json")
		content := `{"app": "my-app", "organization_id": 123}`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		config, err := LoadConfigFromPath(configPath)
		if err != nil {
			t.Fatalf("LoadConfigFromPath failed: %v", err)
		}

		if config.App != "my-app" {
			t.Errorf("Expected App to be 'my-app', got '%s'", config.App)
		}
		if config.OrganizationID == nil || *config.OrganizationID != 123 {
			t.Errorf("Expected OrganizationID to be 123, got %v", config.OrganizationID)
		}
	})

	t.Run("config without organization_id", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "no_org_shuttl.json")
		content := `{"app": "another-app"}`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		config, err := LoadConfigFromPath(configPath)
		if err != nil {
			t.Fatalf("LoadConfigFromPath failed: %v", err)
		}

		if config.App != "another-app" {
			t.Errorf("Expected App to be 'another-app', got '%s'", config.App)
		}
		if config.OrganizationID != nil {
			t.Errorf("Expected OrganizationID to be nil, got %v", config.OrganizationID)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := LoadConfigFromPath(filepath.Join(tmpDir, "nonexistent.json"))
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "invalid.json")
		content := `{"app": "test", invalid json}`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		_, err := LoadConfigFromPath(configPath)
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
	})

	t.Run("empty JSON object", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "empty.json")
		content := `{}`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		config, err := LoadConfigFromPath(configPath)
		if err != nil {
			t.Fatalf("LoadConfigFromPath failed: %v", err)
		}

		if config.App != "" {
			t.Errorf("Expected App to be empty, got '%s'", config.App)
		}
		if config.OrganizationID != nil {
			t.Errorf("Expected OrganizationID to be nil, got %v", config.OrganizationID)
		}
	})
}

func TestFindConfigFileFrom(t *testing.T) {
	// Create a nested directory structure
	tmpDir, err := os.MkdirTemp("", "find_config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("config in current directory", func(t *testing.T) {
		subDir := filepath.Join(tmpDir, "test1")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}

		configPath := filepath.Join(subDir, ConfigFileName)
		if err := os.WriteFile(configPath, []byte(`{"app": "test"}`), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		found, err := FindConfigFileFrom(subDir)
		if err != nil {
			t.Fatalf("FindConfigFileFrom failed: %v", err)
		}

		if found != configPath {
			t.Errorf("Expected to find '%s', got '%s'", configPath, found)
		}
	})

	t.Run("config in parent directory", func(t *testing.T) {
		parentDir := filepath.Join(tmpDir, "test2")
		childDir := filepath.Join(parentDir, "child", "grandchild")
		if err := os.MkdirAll(childDir, 0755); err != nil {
			t.Fatalf("Failed to create dirs: %v", err)
		}

		configPath := filepath.Join(parentDir, ConfigFileName)
		if err := os.WriteFile(configPath, []byte(`{"app": "parent"}`), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		found, err := FindConfigFileFrom(childDir)
		if err != nil {
			t.Fatalf("FindConfigFileFrom failed: %v", err)
		}

		if found != configPath {
			t.Errorf("Expected to find '%s', got '%s'", configPath, found)
		}
	})

	t.Run("no config found", func(t *testing.T) {
		noConfigDir := filepath.Join(tmpDir, "no_config_here")
		if err := os.MkdirAll(noConfigDir, 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}

		_, err := FindConfigFileFrom(noConfigDir)
		if err == nil {
			t.Error("Expected error when no config found, got nil")
		}
	})
}

func TestGetConfigDir(t *testing.T) {
	testCases := []struct {
		name       string
		configPath string
		expected   string
	}{
		{
			name:       "simple path",
			configPath: "/home/user/project/shuttl.json",
			expected:   "/home/user/project",
		},
		{
			name:       "root path",
			configPath: "/shuttl.json",
			expected:   "/",
		},
		{
			name:       "nested path",
			configPath: "/a/b/c/d/shuttl.json",
			expected:   "/a/b/c/d",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetConfigDir(tc.configPath)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestConfigFileName(t *testing.T) {
	if ConfigFileName != "shuttl.json" {
		t.Errorf("Expected ConfigFileName to be 'shuttl.json', got '%s'", ConfigFileName)
	}
}

