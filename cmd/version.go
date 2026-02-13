package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build-time variables, injected via ldflags:
//
//	go build -ldflags "-X github.com/toyinlola/shipsafe/cmd.Version=1.0.0
//	  -X github.com/toyinlola/shipsafe/cmd.Commit=$(git rev-parse --short HEAD)
//	  -X github.com/toyinlola/shipsafe/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("shipsafe %s\n", Version)
		fmt.Printf("  commit:  %s\n", Commit)
		fmt.Printf("  built:   %s\n", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
