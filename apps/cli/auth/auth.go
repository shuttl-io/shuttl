package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	// KeyringService is the service name for keyring storage
	KeyringService = "shuttl-cli"
	// KeyringUser is the user name for keyring storage
	KeyringUser = "refresh_token"
	// AuthConfigDir is the directory for auth config
	AuthConfigDir = ".config/shuttl"
	// AuthConfigFile is the filename for auth config
	AuthConfigFile = "auth.json"
)

// Tokens represents the authentication tokens
type Tokens struct {
	IDToken     string    `json:"id_token"`
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// CognitoTokenResponse represents the response from Cognito token endpoint
type CognitoTokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// GetAuthConfigPath returns the path to the auth config file
func GetAuthConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, AuthConfigDir, AuthConfigFile), nil
}

// SaveTokens saves ID token and access token to the config file
func SaveTokens(tokens *Tokens) error {
	configPath, err := GetAuthConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write auth config: %w", err)
	}

	return nil
}

// LoadTokens loads tokens from the config file
func LoadTokens() (*Tokens, error) {
	configPath, err := GetAuthConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in - run 'shuttl login' first")
		}
		return nil, fmt.Errorf("failed to read auth config: %w", err)
	}

	var tokens Tokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse auth config: %w", err)
	}

	return &tokens, nil
}

// SaveRefreshToken saves the refresh token to the system keychain
func SaveRefreshToken(refreshToken string) error {
	return keyring.Set(KeyringService, KeyringUser, refreshToken)
}

// GetRefreshToken retrieves the refresh token from the system keychain
func GetRefreshToken() (string, error) {
	token, err := keyring.Get(KeyringService, KeyringUser)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", fmt.Errorf("not logged in - run 'shuttl login' first")
		}
		return "", fmt.Errorf("failed to get refresh token from keychain: %w", err)
	}
	return token, nil
}

// DeleteRefreshToken removes the refresh token from the system keychain
func DeleteRefreshToken() error {
	err := keyring.Delete(KeyringService, KeyringUser)
	if err != nil && err != keyring.ErrNotFound {
		return fmt.Errorf("failed to delete refresh token from keychain: %w", err)
	}
	return nil
}

// DeleteTokens removes all stored auth tokens
func DeleteTokens() error {
	configPath, err := GetAuthConfigPath()
	if err != nil {
		return err
	}

	// Remove the config file
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete auth config: %w", err)
	}

	// Remove refresh token from keychain
	return DeleteRefreshToken()
}

// IsLoggedIn checks if the user is currently logged in
func IsLoggedIn() bool {
	tokens, err := LoadTokens()
	if err != nil {
		return false
	}
	return tokens.ExpiresAt.After(time.Now())
}
