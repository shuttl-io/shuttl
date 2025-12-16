package cmd

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/shuttl-ai/cli/auth"
	"github.com/spf13/cobra"
)

const (
	// Cognito configuration
	cognitoDomain = "auth.shuttl.io"
	clientID      = "1i50n7sggqur1rpvp7tkv63oou" // TODO: Set your CLI client ID from Cognito
	redirectURI   = "http://localhost:7812/auth/callback"
	callbackPort  = "7812"

	// OAuth scopes
	oauthScopes = "email openid"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Shuttl",
	Long: `Authenticate with Shuttl using your browser.

This command will open your default browser to the Shuttl login page.
After authenticating, your tokens will be securely stored locally.`,
	Run: runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from Shuttl",
	Long:  `Remove stored authentication tokens and log out of Shuttl.`,
	Run:   runLogout,
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
}

// generateCodeVerifier creates a cryptographically random code verifier for PKCE
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge creates the code challenge from the verifier
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// generateState creates a random state parameter for CSRF protection
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func runLogin(cmd *cobra.Command, args []string) {
	// Check if already logged in
	if auth.IsLoggedIn() {
		fmt.Println("✓ You are already logged in.")
		fmt.Println("  Run 'shuttl logout' first if you want to login with a different account.")
		return
	}

	// Generate PKCE code verifier and challenge
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error generating code verifier: %v\n", err)
		os.Exit(1)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)
	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error generating state: %v\n", err)
		os.Exit(1)
	}

	// Channel to receive the authorization code
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start local HTTP server for OAuth callback
	server := startCallbackServer(state, codeChan, errChan)
	defer server.Shutdown(context.Background())

	// Build the authorization URL
	authURL := buildAuthURL(state, codeChallenge)

	fmt.Println("🔐 Opening browser for authentication...")
	fmt.Println()
	fmt.Printf("If your browser doesn't open automatically, visit:\n%s\n\n", authURL)

	// Open the browser
	if err := browser.OpenURL(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not open browser automatically: %v\n", err)
	}

	fmt.Println("Waiting for authentication...")

	// Wait for the callback
	select {
	case code := <-codeChan:
		// Exchange the code for tokens
		tokens, err := exchangeCodeForTokens(code, codeVerifier)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error exchanging code for tokens: %v\n", err)
			os.Exit(1)
		}

		// Save tokens
		if err := saveAuthTokens(tokens); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error saving tokens: %v\n", err)
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println("✓ Successfully logged in!")
	case err := <-errChan:
		fmt.Fprintf(os.Stderr, "❌ Authentication error: %v\n", err)
		os.Exit(1)
	case <-time.After(5 * time.Minute):
		fmt.Fprintf(os.Stderr, "❌ Authentication timed out\n")
		os.Exit(1)
	}
}

func runLogout(cmd *cobra.Command, args []string) {
	if err := auth.DeleteTokens(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error logging out: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Successfully logged out")
}

func startCallbackServer(expectedState string, codeChan chan<- string, errChan chan<- error) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		// Check for errors from Cognito
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errDesc := r.URL.Query().Get("error_description")
			errChan <- fmt.Errorf("%s: %s", errMsg, errDesc)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body style="font-family: system-ui; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0;">
<div style="text-align: center;">
<h1 style="color: #e53e3e;">❌ Authentication Failed</h1>
<p>%s</p>
<p>You can close this window.</p>
</div>
</body>
</html>`, errDesc)
			return
		}

		// Verify state to prevent CSRF attacks
		state := r.URL.Query().Get("state")
		if state != expectedState {
			errChan <- fmt.Errorf("state mismatch - possible CSRF attack")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "State mismatch")
			return
		}

		// Get the authorization code
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "No authorization code")
			return
		}

		// Send the code back
		codeChan <- code

		// Send success response
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authentication Successful</title></head>
<body style="font-family: system-ui; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);">
<div style="text-align: center; color: white;">
<h1>✓ Authentication Successful!</h1>
<p>You can close this window and return to your terminal.</p>
</div>
</body>
</html>`)
	})

	server := &http.Server{
		Addr:    ":" + callbackPort,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %v", err)
		}
	}()

	return server
}

func buildAuthURL(state, codeChallenge string) string {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", oauthScopes)
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")

	return fmt.Sprintf("https://%s/oauth2/authorize?%s", cognitoDomain, params.Encode())
}

func exchangeCodeForTokens(code, codeVerifier string) (*auth.CognitoTokenResponse, error) {
	tokenURL := fmt.Sprintf("https://%s/oauth2/token", cognitoDomain)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokens auth.CognitoTokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokens, nil
}

func saveAuthTokens(cognitoTokens *auth.CognitoTokenResponse) error {
	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(cognitoTokens.ExpiresIn) * time.Second)

	// Save ID token and access token to file
	tokens := &auth.Tokens{
		IDToken:     cognitoTokens.IDToken,
		AccessToken: cognitoTokens.AccessToken,
		ExpiresAt:   expiresAt,
	}

	if err := auth.SaveTokens(tokens); err != nil {
		return fmt.Errorf("failed to save tokens to config: %w", err)
	}

	// Save refresh token to keychain
	if cognitoTokens.RefreshToken != "" {
		if err := auth.SaveRefreshToken(cognitoTokens.RefreshToken); err != nil {
			return fmt.Errorf("failed to save refresh token to keychain: %w", err)
		}
	}

	return nil
}
