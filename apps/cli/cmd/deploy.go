package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shuttl-ai/cli/auth"
	"github.com/shuttl-ai/cli/config"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [app]",
	Short: "Build and deploy Shuttl agents to Endeavour",
	Long: `Deploy builds the project, packages per-agent/per-trigger tarballs,
and uploads them to the Endeavour build service.`,
	Args: cobra.MaximumNArgs(1),
	Run:  runDeploy,
}

func init() {
	deployCmd.Flags().String("config", "", "Path to shuttl.json (defaults to searching current and parent directories)")
	deployCmd.Flags().String("architecture", "", "Target architecture for generated Dockerfile (default: amd64)")
	deployCmd.Flags().String("build-url", "", "Override Endeavour build service URL")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) {
	if !auth.IsLoggedIn() {
		fmt.Fprintf(os.Stderr, "❌ Error: not logged in - run 'shuttl login' first\n")
		os.Exit(1)
	}
	tokens, err := auth.LoadTokens()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: failed to load auth tokens: %v\n", err)
		os.Exit(1)
	}

	configPath, _ := cmd.Flags().GetString("config")
	architecture, _ := cmd.Flags().GetString("architecture")
	buildURLOverride, _ := cmd.Flags().GetString("build-url")

	configDir, err := resolveProjectDir(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	if err := runBuildCommand(configPath, architecture); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error running build: %v\n", err)
		os.Exit(1)
	}

	assets, err := loadAssetManifest(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error loading asset manifest: %v\n", err)
		os.Exit(1)
	}
	if len(assets) == 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: no assets found in %s\n", filepath.Join(configDir, buildOutputDirName))
		os.Exit(1)
	}

	buildURL, err := resolveBuildURL(buildURLOverride)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🚀 Uploading %d build archive(s) to %s\n", len(assets), buildURL)
	if err := uploadAssets(cmd.Context(), buildURL, configDir, assets, tokens.AccessToken); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error uploading archives: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Deploy complete")
}

func resolveProjectDir(configPath string) (string, error) {
	if configPath != "" {
		return config.GetConfigDir(configPath), nil
	}
	cfg, err := config.LoadConfig()
	if err == nil && cfg != nil {
		path, err := config.FindConfigFileFrom(".")
		if err == nil {
			return config.GetConfigDir(path), nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine working directory: %w", err)
	}
	return cwd, nil
}

func runBuildCommand(configPath string, architecture string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{"build"}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	if architecture != "" {
		args = append(args, "--architecture", architecture)
	}
	cmd := exec.Command(exe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func loadAssetManifest(projectRoot string) (map[string]string, error) {
	path := filepath.Join(projectRoot, buildOutputDirName, "asset-manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var assets map[string]string
	if err := json.Unmarshal(data, &assets); err != nil {
		return nil, err
	}
	return assets, nil
}

func resolveBuildURL(override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return strings.TrimRight(override, "/"), nil
	}
	userCfg, err := config.LoadUserConfig()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(userCfg.GetBuildURL(), "/"), nil
}

func uploadAssets(ctx context.Context, baseURL string, projectRoot string, assets map[string]string, accessToken string) error {
	client := &http.Client{Timeout: 15 * time.Minute}
	endpoint := strings.TrimRight(baseURL, "/") + "/build"
	for agent, relPath := range assets {
		fmt.Printf("Deploying %s\n", agent)
		archive := filepath.Join(projectRoot, buildOutputDirName, filepath.FromSlash(relPath))
		if err := uploadArchive(ctx, client, endpoint, archive, accessToken); err != nil {
			return err
		}
	}
	return nil
}

func uploadArchive(ctx context.Context, client *http.Client, endpoint string, archivePath string, accessToken string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, file)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/gzip")
	req.Header.Set("X-Shuttl-Archive", filepath.Base(archivePath))
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if isStreamingResponse(resp.Header.Get("Content-Type")) {
		if err := streamResponse(resp.Body, filepath.Base(archivePath)); err != nil {
			return err
		}
		fmt.Printf("✅ Uploaded %s\n", filepath.Base(archivePath))
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("body: %s\n", string(body))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload failed for %s: %s", filepath.Base(archivePath), strings.TrimSpace(string(body)))
	}
	if len(body) > 0 {
		fmt.Println(strings.TrimSpace(string(body)))
		return nil
	}
	fmt.Printf("✅ Uploaded %s\n", filepath.Base(archivePath))
	return nil
}

func isStreamingResponse(contentType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return normalized == "text/event-stream" || normalized == "test/event-stream"
}

func streamResponse(body io.Reader, label string) error {
	stopSpinner := make(chan struct{})
	go spinner(stopSpinner, fmt.Sprintf("Uploading %s", label))

	defer close(stopSpinner)

	scanner := bufio.NewScanner(body)
	var lastEvent string
	var deployFailed bool
	var deployError string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			lastEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			printStreamMessage(lastEvent, data)
			if strings.EqualFold(lastEvent, "error") {
				deployFailed = true
				deployError = data
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if deployFailed {
		if deployError == "" {
			return fmt.Errorf("deploy failed")
		}
		return fmt.Errorf("deploy failed: %s", deployError)
	}
	return nil
}

type StreamMessage struct {
	Message *string            `json:"message"`
	Fields  *map[string]string `json:"fields"`
	Error   *string            `json:"error"`
}

func printStreamMessage(event, data string) {
	// Clear spinner line
	// fmt.Print("\r")
	var message StreamMessage
	if err := json.Unmarshal([]byte(data), &message); err != nil {
		fmt.Printf("error unmarshalling stream message: %v\n", err)
		return
	}
	if message.Message != nil {
		fmt.Printf("%s\n", *message.Message)
	}
	if message.Error != nil {
		fmt.Printf("❌ Error: %s\n", *message.Error)
	}
}

func spinner(stop <-chan struct{}, prefix string) {
	frames := []rune{'|', '/', '-', '\\'}
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()
	index := 0
	for {
		select {
		case <-stop:
			fmt.Print("\r")
			return
		case <-ticker.C:
			fmt.Printf("\r%s %c", prefix, frames[index%len(frames)])
			index++
		}
	}
}
