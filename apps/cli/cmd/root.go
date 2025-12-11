package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "shuttl",
	Short: "Shuttl AI CLI - Deploy and manage AI agents",
	Long: `Shuttl AI CLI is a command-line tool for deploying, managing, 
and developing AI agents in the Shuttl ecosystem.

Use this tool to:
  - Deploy agents to production
  - Run agents in development mode
  - List and manage deployed agents
  - And more...`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
}
