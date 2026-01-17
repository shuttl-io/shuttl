package cmd

import (
	"context"
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

	archives, err := findBuildArchives(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error finding build archives: %v\n", err)
		os.Exit(1)
	}
	if len(archives) == 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: no build archives found in %s\n", filepath.Join(configDir, buildOutputDirName))
		os.Exit(1)
	}

	buildURL, err := resolveBuildURL(buildURLOverride)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🚀 Uploading %d build archive(s) to %s\n", len(archives), buildURL)
	if err := uploadArchives(cmd.Context(), buildURL, archives); err != nil {
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

func findBuildArchives(projectRoot string) ([]string, error) {
	root := filepath.Join(projectRoot, buildOutputDirName)
	var archives []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".tar.gz") {
			archives = append(archives, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return archives, nil
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

func uploadArchives(ctx context.Context, baseURL string, archives []string) error {
	client := &http.Client{Timeout: 15 * time.Minute}
	endpoint := strings.TrimRight(baseURL, "/") + "/build"
	for _, archive := range archives {
		if err := uploadArchive(ctx, client, endpoint, archive); err != nil {
			return err
		}
	}
	return nil
}

func uploadArchive(ctx context.Context, client *http.Client, endpoint string, archivePath string) error {
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

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload failed for %s: %s", filepath.Base(archivePath), strings.TrimSpace(string(body)))
	}
	fmt.Printf("✅ Uploaded %s\n", filepath.Base(archivePath))
	return nil
}
