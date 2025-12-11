package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information - these can be set at build time using ldflags
var (
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Print detailed version information including the version number, git commit, and build date.`,
	Run: func(cmd *cobra.Command, args []string) {
		short, _ := cmd.Flags().GetBool("short")
		
		if short {
			fmt.Println(Version)
			return
		}

		fmt.Printf("Shuttl CLI\n")
		fmt.Printf("  Version:    %s\n", Version)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		fmt.Printf("  Go Version: %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	versionCmd.Flags().BoolP("short", "s", false, "Print only the version number")
	rootCmd.AddCommand(versionCmd)
}











