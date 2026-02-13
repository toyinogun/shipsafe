// Package cmd implements the ShipSafe CLI commands using Cobra.
package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	format  string
	output  string
)

var rootCmd = &cobra.Command{
	Use:   "shipsafe",
	Short: "AI code verification gateway",
	Long: `ShipSafe is a self-hosted AI code verification gateway.

It sits in CI/CD pipelines between code generation and merge, running
multi-layered verification on AI-generated code to produce trust scores
and verification reports.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return setupLogging()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default: .shipsafe.yml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "terminal", "output format (terminal|json|markdown|html)")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "write output to file instead of stdout")
}

func setupLogging() error {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))

	return nil
}
