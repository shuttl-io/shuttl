package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetAuthConfigPath(t *testing.T) {
	path, err := GetAuthConfigPath()
	if err != nil {
		t.Fatalf("GetAuthConfigPath failed: %v", err)
	}

	if path == "" {
		t.Error("Expected non-empty path")
	}

	// Verify the path contains expected components
	if filepath.Base(path) != AuthConfigFile {
		t.Errorf("Expected path to end with '%s', got '%s'", AuthConfigFile, filepath.Base(path))
	}
}

func TestSaveAndLoadTokens(t *testing.T) {
	// Save original HOME and restore after test
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	tmpDir, err := os.MkdirTemp("", "auth_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)

	t.Run("save and load tokens", func(t *testing.T) {
		expiresAt := time.Now().Add(time.Hour)
		tokens := &Tokens{
			IDToken:     "test-id-token",
			AccessToken: "test-access-token",
			ExpiresAt:   expiresAt,
		}

		err := SaveTokens(tokens)
		if err != nil {
			t.Fatalf("SaveTokens failed: %v", err)
		}

		loaded, err := LoadTokens()
		if err != nil {
			t.Fatalf("LoadTokens failed: %v", err)
		}

		if loaded.IDToken != tokens.IDToken {
			t.Errorf("Expected IDToken '%s', got '%s'", tokens.IDToken, loaded.IDToken)
		}

		if loaded.AccessToken != tokens.AccessToken {
			t.Errorf("Expected AccessToken '%s', got '%s'", tokens.AccessToken, loaded.AccessToken)
		}

		// Time comparison with some tolerance
		if loaded.ExpiresAt.Sub(tokens.ExpiresAt) > time.Second {
			t.Errorf("ExpiresAt mismatch: expected %v, got %v", tokens.ExpiresAt, loaded.ExpiresAt)
		}
	})

	t.Run("load tokens when not logged in", func(t *testing.T) {
		// Create a fresh temp directory
		freshDir, err := os.MkdirTemp("", "auth_test_fresh")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(freshDir)

		os.Setenv("HOME", freshDir)

		_, err = LoadTokens()
		if err == nil {
			t.Error("Expected error when not logged in, got nil")
		}
	})
}

func TestDeleteTokens(t *testing.T) {
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	tmpDir, err := os.MkdirTemp("", "auth_delete_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)

	// Save some tokens first
	tokens := &Tokens{
		IDToken:     "to-delete",
		AccessToken: "to-delete-access",
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	if err := SaveTokens(tokens); err != nil {
		t.Fatalf("SaveTokens failed: %v", err)
	}

	// Verify they exist
	configPath, _ := GetAuthConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Tokens file should exist before deletion")
	}

	// Note: DeleteTokens also tries to delete from keyring, which may fail in test environment
	// We're mainly testing the file deletion here
	_ = DeleteTokens()

	// Verify file is deleted
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("Tokens file should be deleted")
	}
}

func TestIsLoggedIn(t *testing.T) {
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	t.Run("returns false when no tokens exist", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "auth_logged_in_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		os.Setenv("HOME", tmpDir)

		if IsLoggedIn() {
			t.Error("Expected IsLoggedIn to return false when no tokens exist")
		}
	})

	t.Run("returns false when tokens are expired", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "auth_expired_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		os.Setenv("HOME", tmpDir)

		tokens := &Tokens{
			IDToken:     "expired-token",
			AccessToken: "expired-access",
			ExpiresAt:   time.Now().Add(-time.Hour), // Expired 1 hour ago
		}
		if err := SaveTokens(tokens); err != nil {
			t.Fatalf("SaveTokens failed: %v", err)
		}

		if IsLoggedIn() {
			t.Error("Expected IsLoggedIn to return false for expired tokens")
		}
	})

	t.Run("returns true when tokens are valid", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "auth_valid_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		os.Setenv("HOME", tmpDir)

		tokens := &Tokens{
			IDToken:     "valid-token",
			AccessToken: "valid-access",
			ExpiresAt:   time.Now().Add(time.Hour), // Valid for 1 more hour
		}
		if err := SaveTokens(tokens); err != nil {
			t.Fatalf("SaveTokens failed: %v", err)
		}

		if !IsLoggedIn() {
			t.Error("Expected IsLoggedIn to return true for valid tokens")
		}
	})
}

func TestTokensJSONMarshaling(t *testing.T) {
	expiresAt := time.Date(2024, 12, 25, 12, 0, 0, 0, time.UTC)
	tokens := &Tokens{
		IDToken:     "test-id",
		AccessToken: "test-access",
		ExpiresAt:   expiresAt,
	}

	data, err := json.Marshal(tokens)
	if err != nil {
		t.Fatalf("Failed to marshal tokens: %v", err)
	}

	var unmarshaled Tokens
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal tokens: %v", err)
	}

	if unmarshaled.IDToken != tokens.IDToken {
		t.Errorf("IDToken mismatch: expected '%s', got '%s'", tokens.IDToken, unmarshaled.IDToken)
	}

	if unmarshaled.AccessToken != tokens.AccessToken {
		t.Errorf("AccessToken mismatch: expected '%s', got '%s'", tokens.AccessToken, unmarshaled.AccessToken)
	}

	if !unmarshaled.ExpiresAt.Equal(tokens.ExpiresAt) {
		t.Errorf("ExpiresAt mismatch: expected %v, got %v", tokens.ExpiresAt, unmarshaled.ExpiresAt)
	}
}

func TestCognitoTokenResponse(t *testing.T) {
	jsonData := `{
		"id_token": "cognito-id",
		"access_token": "cognito-access",
		"refresh_token": "cognito-refresh",
		"expires_in": 3600,
		"token_type": "Bearer"
	}`

	var response CognitoTokenResponse
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		t.Fatalf("Failed to unmarshal CognitoTokenResponse: %v", err)
	}

	if response.IDToken != "cognito-id" {
		t.Errorf("Expected IDToken 'cognito-id', got '%s'", response.IDToken)
	}

	if response.AccessToken != "cognito-access" {
		t.Errorf("Expected AccessToken 'cognito-access', got '%s'", response.AccessToken)
	}

	if response.RefreshToken != "cognito-refresh" {
		t.Errorf("Expected RefreshToken 'cognito-refresh', got '%s'", response.RefreshToken)
	}

	if response.ExpiresIn != 3600 {
		t.Errorf("Expected ExpiresIn 3600, got %d", response.ExpiresIn)
	}

	if response.TokenType != "Bearer" {
		t.Errorf("Expected TokenType 'Bearer', got '%s'", response.TokenType)
	}
}

func TestConstants(t *testing.T) {
	if KeyringService != "shuttl-cli" {
		t.Errorf("Expected KeyringService 'shuttl-cli', got '%s'", KeyringService)
	}

	if KeyringUser != "refresh_token" {
		t.Errorf("Expected KeyringUser 'refresh_token', got '%s'", KeyringUser)
	}

	if AuthConfigDir != ".config/shuttl" {
		t.Errorf("Expected AuthConfigDir '.config/shuttl', got '%s'", AuthConfigDir)
	}

	if AuthConfigFile != "auth.json" {
		t.Errorf("Expected AuthConfigFile 'auth.json', got '%s'", AuthConfigFile)
	}
}

