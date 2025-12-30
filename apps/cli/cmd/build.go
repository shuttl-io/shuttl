package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shuttl-ai/cli/config"
	"github.com/shuttl-ai/cli/ipc"
	"github.com/spf13/cobra"
)

// Manifest represents the complete manifest structure
type Manifest struct {
	Version   string               `json:"version"`
	BuildTime string               `json:"buildTime"`
	App       string               `json:"app"`
	Agents    []ipc.AgentInfo      `json:"agents"`
	Toolkits  []ipc.ToolkitInfo    `json:"toolkits"`
	Tools     []ipc.SingleToolInfo `json:"tools"`
	Triggers  []ipc.TriggerInfo    `json:"triggers"`
	Models    []ipc.ModelInfo      `json:"models"`
	Prompts   []ipc.PromptInfo     `json:"prompts"`
}

var buildCmd = &cobra.Command{
	Use:   "build [app]",
	Short: "Build a manifest file from the app",
	Long: `Build command introspects the app via IPC and creates a manifest file
containing all agents, triggers, tools, models, and prompts.

The manifest file is written as shuttl-manifest.json in the current directory.

If a shuttl.json file is found, the "app" field will be used by default.
You can also specify the app path directly as an argument.

Examples:
  shuttl build
  shuttl build ./my-app
  shuttl build --output custom-manifest.json`,
	Args: cobra.MaximumNArgs(1),
	Run:  runBuild,
}

func init() {
	buildCmd.Flags().String("config", "", "Path to shuttl.json (defaults to searching current and parent directories)")
	buildCmd.Flags().StringP("output", "o", "shuttl-manifest.json", "Output file path for the manifest")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) {
	configPath, _ := cmd.Flags().GetString("config")
	outputPath, _ := cmd.Flags().GetString("output")

	var appPath string

	// If app is provided as argument, use it directly
	if len(args) > 0 {
		appPath = args[0]
	} else {
		// Try to load configuration
		var cfg *config.Config
		var err error

		if configPath != "" {
			cfg, err = config.LoadConfigFromPath(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			cfg, _ = config.LoadConfig()
			// Config file is optional
		}

		if cfg != nil && cfg.App != "" {
			appPath = cfg.App
		}
	}

	if appPath == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: no app specified. Provide an app path as argument or create a shuttl.json config file.\n")
		os.Exit(1)
	}

	fmt.Printf("🔧 Building manifest for: %s\n", appPath)

	// Parse the app command string into an array
	command := ipc.ParseCommand(appPath)
	if len(command) == 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: empty app command\n")
		os.Exit(1)
	}

	// Create IPC client
	client := ipc.NewClient(command)
	if err := client.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error starting app: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Give the app time to initialize
	time.Sleep(500 * time.Millisecond)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(client.Context(), 30*time.Second)
	defer cancel()

	// Fetch all data via IPC
	fmt.Println("📡 Fetching agents...")
	agents, err := client.GetAgents(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error fetching agents: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Found %d agent(s)\n", len(agents))

	fmt.Println("📡 Fetching toolkits...")
	toolkits, err := client.GetToolkits(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error fetching toolkits: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Found %d toolkit(s)\n", len(toolkits))

	fmt.Println("📡 Fetching tools...")
	tools, err := client.GetTools(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error fetching tools: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Found %d tool(s)\n", len(tools))

	fmt.Println("📡 Fetching triggers...")
	triggers, err := client.GetTriggers(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error fetching triggers: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Found %d trigger(s)\n", len(triggers))

	fmt.Println("📡 Fetching models...")
	models, err := client.GetModels(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error fetching models: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Found %d model(s)\n", len(models))

	fmt.Println("📡 Fetching prompts...")
	prompts, err := client.GetPrompts(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error fetching prompts: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Found %d prompt(s)\n", len(prompts))

	// Build manifest
	manifest := Manifest{
		Version:   "1.0",
		BuildTime: time.Now().UTC().Format(time.RFC3339),
		App:       appPath,
		Agents:    agents,
		Toolkits:  toolkits,
		Tools:     tools,
		Triggers:  triggers,
		Models:    models,
		Prompts:   prompts,
	}

	// Marshal to JSON with indentation
	jsonData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error marshaling manifest: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		absPath = outputPath
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error writing manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Manifest written to: %s\n", absPath)
}
