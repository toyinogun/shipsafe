package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var diffFile string

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan code for issues and generate a trust score",
	Long: `Scan analyzes code changes and produces a trust score report.

Scan a diff file directly:
  shipsafe scan --diff ./path/to/file.diff

Scan a directory (compares against git HEAD):
  shipsafe scan ./path/to/repo`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScan,
}

func init() {
	scanCmd.Flags().StringVar(&diffFile, "diff", "", "path to a unified diff file to analyze")
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	var target string
	if len(args) > 0 {
		target = args[0]
	}

	if diffFile == "" && target == "" {
		return fmt.Errorf("scan: provide either --diff <file> or a target path")
	}

	slog.Info("scan started",
		"diff", diffFile,
		"target", target,
		"format", format,
		"config", cfgFile,
	)

	// TODO: Wire up the analysis pipeline:
	// 1. Load config from cfgFile (or default .shipsafe.yml)
	// 2. Parse diff (from --diff file or generate from target directory)
	// 3. Run analyzers via the Pipeline interface
	// 4. Output the report in the requested format

	fmt.Println("shipsafe scan: not yet implemented")
	return nil
}
