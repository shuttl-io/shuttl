package cmd

import (
	"fmt"
	"os"

	"github.com/shuttl-ai/cli/config"
	"github.com/shuttl-ai/cli/ipc"
	"github.com/shuttl-ai/cli/tui"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev [app]",
	Short: "Run the app in development mode",
	Long: `Run an app in development mode with an interactive TUI.

The TUI provides three screens:
  - Agents: Select an agent to chat with
  - Chat: Interactive chat with selected agents (supports multiple sessions)
  - Logs: View logs from all agents with filtering

If a shuttl.json file is found, the "app" field will be used by default.
You can also specify the app path directly as an argument.

Example shuttl.json:
  {
    "app": "./my-agent"
  }

Examples:
  shuttl dev
  shuttl dev ./my-app`,
	Args: cobra.MaximumNArgs(1),
	Run:  runDev,
}

func init() {
	devCmd.Flags().String("config", "", "Path to shuttl.json (defaults to searching current and parent directories)")
	rootCmd.AddCommand(devCmd)
}

func runDev(cmd *cobra.Command, args []string) {
	configPath, _ := cmd.Flags().GetString("config")

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

	// Create IPC client if we have an app command
	var client *ipc.Client
	if appPath != "" {
		// Parse the app command string into an array
		command := ipc.ParseCommand(appPath)
		if len(command) == 0 {
			fmt.Fprintf(os.Stderr, "❌ Error: empty app command\n")
			os.Exit(1)
		}

		client = ipc.NewClient(command)
		if err := client.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error starting app: %v\n", err)
			os.Exit(1)
		}
	}

	// Launch the TUI (client will be stopped when TUI exits)
	if err := tui.Run(client); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
}
