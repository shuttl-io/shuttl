package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the Shuttl CLI to the latest version",
	Long:  `Check for the latest version of Shuttl CLI and update the executable if a newer version is available.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

var updateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for updates and show release notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔍 Checking for updates...")

		releases, err := fetchReleases()
		if err != nil {
			return err
		}

		if len(releases) == 0 {
			fmt.Printf("✅ Shuttl CLI is already up to date (version %s).\n", Version)
			return nil
		}

		latestVersion := releases[0].TagName
		if !isNewer(Version, latestVersion) {
			fmt.Printf("✅ Shuttl CLI is already up to date (version %s).\n", Version)
			return nil
		}

		fmt.Printf("✨ A new version of Shuttl CLI is available: %s (current: %s)\n", latestVersion, Version)

		fmt.Println("\nRelease Notes between versions:")
		fmt.Println("===============================")

		var newerReleases []GitHubRelease
		foundCurrent := false
		for _, release := range releases {
			if !isNewer(Version, release.TagName) {
				foundCurrent = true
				break
			}
			newerReleases = append(newerReleases, release)
		}

		// Print in ascending order (earliest first, latest at bottom)
		for i := len(newerReleases) - 1; i >= 0; i-- {
			release := newerReleases[i]
			fmt.Printf("\n--- %s ---\n", release.TagName)
			if release.Body != "" {
				fmt.Println(release.Body)
			} else {
				fmt.Println("(No release notes provided)")
			}
		}

		if !foundCurrent {
			fmt.Println("\n... (older releases not shown)")
		}

		fmt.Println("===============================")
		fmt.Println("\nRun 'shuttl update' to install the new version.")
		return nil
	},
}

func fetchReleases() ([]GitHubRelease, error) {
	client := &http.Client{}
	// Fetch multiple releases to cover the gap
	resp, err := client.Get("https://api.github.com/repos/shuttl-io/shuttl/releases")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases info: %w", err)
	}
	return releases, nil
}

func fetchLatestRelease() (*GitHubRelease, error) {
	releases, err := fetchReleases()
	if err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}
	return &releases[0], nil
}

func runUpdate() error {
	fmt.Println("🔍 Checking for updates...")

	release, err := fetchLatestRelease()
	if err != nil {
		return err
	}

	latestVersion := release.TagName
	if !isNewer(Version, latestVersion) {
		fmt.Printf("✅ Shuttl CLI is already up to date (version %s).\n", Version)
		return nil
	}

	fmt.Printf("🚀 Updating Shuttl CLI from %s to %s...\n", Version, latestVersion)

	// Determine the asset name to look for
	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH

	assetName := fmt.Sprintf("shuttl-cli-%s-shuttl-%s-%s.zip", latestVersion, targetOS, targetArch)

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("could not find a compatible release asset for %s/%s (expected %s)", targetOS, targetArch, assetName)
	}

	fmt.Printf("📦 Downloading %s...\n", assetName)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download update: status %d", resp.StatusCode)
	}

	// Read the entire zip into memory
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read update data: %w", err)
	}

	// Extract the binary from the zip
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}

	var binData []byte
	binaryMatchName := fmt.Sprintf("shuttl-%s-%s", targetOS, targetArch)
	if targetOS == "windows" {
		binaryMatchName += ".exe"
	}

	for _, file := range zipReader.File {
		if file.Name == binaryMatchName || strings.HasSuffix(file.Name, "/"+binaryMatchName) {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open binary in zip: %w", err)
			}
			binData, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return fmt.Errorf("failed to read binary from zip: %w", err)
			}
			break
		}
	}

	if len(binData) == 0 {
		return fmt.Errorf("could not find binary '%s' in the downloaded zip. Found files: %s", binaryMatchName, func() string {
			var names []string
			for _, f := range zipReader.File {
				names = append(names, f.Name)
			}
			return strings.Join(names, ", ")
		}())
	}

	// Get the current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// On Windows, we can't overwrite the running executable directly.
	oldExePath := exePath + ".old"
	if err := os.Rename(exePath, oldExePath); err != nil {
		return fmt.Errorf("failed to move current executable: %w", err)
	}

	// Write the new binary
	if err := os.WriteFile(exePath, binData, 0755); err != nil {
		// Try to restore the old one if writing fails
		os.Rename(oldExePath, exePath)
		return fmt.Errorf("failed to write new executable: %w", err)
	}

	// On Unix-like systems, we should clean up the .old file.
	if targetOS != "windows" {
		os.Remove(oldExePath)
	}

	fmt.Printf("✨ Shuttl CLI has been successfully updated to %s!\n", latestVersion)
	return nil
}

func init() {
	updateCmd.AddCommand(updateCheckCmd)
	rootCmd.AddCommand(updateCmd)
}
