package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/toyinlola/shipsafe/pkg/analyzer"
	"github.com/toyinlola/shipsafe/pkg/cli"
	"github.com/toyinlola/shipsafe/pkg/interfaces"
	"github.com/toyinlola/shipsafe/pkg/report"
	"github.com/toyinlola/shipsafe/pkg/scorer"
	"github.com/toyinlola/shipsafe/pkg/vcs"
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

// formatter writes a structured report to a writer.
type formatter interface {
	Format(w io.Writer, report *interfaces.Report) error
}

func runScan(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var target string
	if len(args) > 0 {
		target = args[0]
	}

	if diffFile == "" && target == "" {
		return fmt.Errorf("scan: provide either --diff <file> or a target path")
	}

	// 1. Load configuration.
	cfg, err := cli.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	slog.Debug("config loaded",
		"thresholds.green", cfg.Thresholds.Green,
		"thresholds.yellow", cfg.Thresholds.Yellow,
	)

	// 2. Parse the diff.
	parser := vcs.NewDiffParser()
	var diff *interfaces.Diff

	if diffFile != "" {
		slog.Info("parsing diff file", "path", diffFile)
		diff, err = parser.ParseFile(ctx, diffFile)
	} else {
		slog.Info("running git diff", "target", target)
		diff, err = diffFromGit(ctx, parser, target)
	}
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	slog.Info("diff parsed", "files", len(diff.Files))

	// 3. Build analyzer registry and register enabled analyzers.
	registry := analyzer.NewRegistry()
	registerAnalyzers(registry, cfg)

	// 4. Run all enabled analyzers.
	engine := analyzer.NewEngine(registry)
	results, err := engine.Run(ctx, diff)
	if err != nil {
		return fmt.Errorf("scan: running analysis: %w", err)
	}

	// 5. Calculate trust score.
	calc := scorer.NewCalculator(
		scorer.WithThresholds(cfg.Thresholds.Green, cfg.Thresholds.Yellow),
	)
	trustScore := calc.Score(results)

	// 6. Generate report.
	gen := report.NewGenerator()
	rpt := gen.Generate(results, trustScore, diff)

	// 7. Select formatter and write output.
	f := selectFormatter(format)

	var w io.Writer = os.Stdout
	if output != "" {
		file, fileErr := os.Create(output)
		if fileErr != nil {
			return fmt.Errorf("scan: creating output file: %w", fileErr)
		}
		defer file.Close() // best-effort cleanup
		w = file
	}

	if err := f.Format(w, rpt); err != nil {
		return fmt.Errorf("scan: writing report: %w", err)
	}

	// 8. Exit with code 1 for RED rating.
	if trustScore.Rating == interfaces.RatingRed {
		os.Exit(1)
	}

	return nil
}

// diffFromGit runs `git diff HEAD` in the given directory and parses the output.
func diffFromGit(ctx context.Context, parser interfaces.DiffParser, dir string) (*interfaces.Diff, error) {
	gitCmd := exec.CommandContext(ctx, "git", "diff", "HEAD")
	gitCmd.Dir = dir

	out, err := gitCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running git diff in %s: %w", dir, err)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no changes found in %s (git diff HEAD returned empty)", dir)
	}

	return parser.Parse(ctx, out)
}

// registerAnalyzers adds all enabled analyzers to the registry based on config.
func registerAnalyzers(registry *analyzer.Registry, cfg *cli.Config) {
	if cfg.Analyzers.Secrets.IsEnabled() {
		_ = registry.Register(analyzer.NewSecretsAnalyzer())
	}
	if cfg.Analyzers.Patterns.IsEnabled() {
		_ = registry.Register(analyzer.NewPatternsAnalyzer())
	}
	if cfg.Analyzers.Complexity.IsEnabled() {
		_ = registry.Register(analyzer.NewComplexityAnalyzer(
			analyzer.WithComplexityThreshold(cfg.Analyzers.Complexity.Threshold),
		))
	}
	if cfg.Analyzers.Coverage.IsEnabled() {
		_ = registry.Register(analyzer.NewCoverageAnalyzer())
	}
	if cfg.Analyzers.Imports.IsEnabled() {
		_ = registry.Register(analyzer.NewImportsAnalyzer())
	}
}

// selectFormatter returns the appropriate report formatter for the given format name.
func selectFormatter(name string) formatter {
	switch name {
	case "json":
		return report.NewJSONFormatter()
	case "markdown":
		return report.NewMarkdownFormatter()
	default:
		return report.NewTerminalFormatter()
	}
}
