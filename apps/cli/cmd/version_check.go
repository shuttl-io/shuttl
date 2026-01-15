package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/shuttl-ai/cli/config"
)

// GitHubAsset represents a file attached to a release
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// GitHubRelease represents the latest release from GitHub API
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Body    string        `json:"body"`
	Assets  []GitHubAsset `json:"assets"`
}

func isNewer(current, latest string) bool {
	// Basic semver comparison
	// current: 0.1.0, latest: v0.2.0 or 0.2.0
	c := strings.TrimPrefix(current, "v")
	l := strings.TrimPrefix(latest, "v")

	cParts := strings.Split(c, ".")
	lParts := strings.Split(l, ".")

	for i := 0; i < len(cParts) && i < len(lParts); i++ {
		cv, _ := strconv.Atoi(cParts[i])
		lv, _ := strconv.Atoi(lParts[i])
		if lv > cv {
			return true
		}
		if cv > lv {
			return false
		}
	}
	return len(lParts) > len(cParts)
}

// CheckForUpdates checks if a new version is available or if the current version is revoked
func CheckForUpdates() {
	userConfig, err := config.LoadUserConfig()
	if err != nil {
		return
	}

	now := time.Now().Unix()
	// Check once every 24 hours
	if now-userConfig.LastVersionCheck < 86400 && userConfig.LatestVersion != "" {
		showVersionWarnings(userConfig)
		return
	}

	// Perform check
	newestVersion, revoked, err := fetchLatestInfo()
	if err != nil {
		// Silently fail if offline or API error, but still show cached warnings if any
		if userConfig.LatestVersion != "" {
			showVersionWarnings(userConfig)
		}
		return
	}

	userConfig.LatestVersion = newestVersion
	userConfig.RevokedVersions = revoked
	userConfig.LastVersionCheck = now
	_ = config.SaveUserConfig(userConfig)

	showVersionWarnings(userConfig)
}

func fetchLatestInfo() (string, []string, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	// Fetch latest release
	resp, err := client.Get("https://api.github.com/repos/shuttl-io/shuttl/releases/latest")
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", nil, err
	}

	// Fetch revocations
	respRev, err := client.Get("https://raw.githubusercontent.com/shuttl-io/shuttl/main/revocations.json")
	var revoked []string
	if err == nil && respRev.StatusCode == http.StatusOK {
		defer respRev.Body.Close()
		_ = json.NewDecoder(respRev.Body).Decode(&revoked)
	}

	return release.TagName, revoked, nil
}

func showVersionWarnings(userConfig *config.UserConfig) {
	current := Version
	latest := userConfig.LatestVersion

	// Check revocation first
	isRevoked := false
	currentClean := strings.TrimPrefix(current, "v")
	for _, r := range userConfig.RevokedVersions {
		rClean := strings.TrimPrefix(r, "v")
		if currentClean == rClean {
			isRevoked = true
			break
		}
	}

	if isRevoked {
		warnStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true).
			Padding(1, 2)

		fmt.Println(warnStyle.Render(fmt.Sprintf("⚠️  WARNING: Your current version (%s) has been REVOKED.", current)))
		fmt.Println(warnStyle.Render(fmt.Sprintf("   Please update to the latest good version (%s) immediately.", latest)))
		fmt.Println()
		return
	}

	if latest != "" && isNewer(current, latest) {
		infoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true).
			Padding(0, 1)

		fmt.Println(infoStyle.Render(fmt.Sprintf("✨ A new version of Shuttl CLI is available: %s (current: %s)", latest, current)))
		fmt.Println(infoStyle.Render("   Run the update command or download from GitHub to update."))
		fmt.Println()
	}
}
