package main

import (
	"os"

	"github.com/shuttl-ai/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}






















