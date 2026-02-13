package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/toyinlola/shipsafe/pkg/ai"
	"github.com/toyinlola/shipsafe/pkg/ai/providers"
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

	// 5. Run AI review if enabled.
	if aiResult := runAIReview(ctx, cfg, diff); aiResult != nil {
		results = append(results, aiResult)
	}

	// 5b. Deduplicate findings across static analyzers and AI review.
	results = deduplicateCrossAnalyzer(results)

	// 6. Calculate trust score.
	calc := scorer.NewCalculator(
		scorer.WithThresholds(cfg.Thresholds.Green, cfg.Thresholds.Yellow),
	)
	trustScore := calc.Score(results)

	// 7. Generate report.
	gen := report.NewGenerator()
	rpt := gen.Generate(results, trustScore, diff)

	// 8. Select formatter and write output.
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

	// 9. Exit with code 1 for RED rating.
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

// runAIReview creates an AI reviewer from config and runs it against the diff.
// Returns nil if AI review is disabled, unavailable, or fails.
func runAIReview(ctx context.Context, cfg *cli.Config, diff *interfaces.Diff) *interfaces.AnalysisResult {
	if !cfg.AI.Enabled {
		slog.Debug("AI review disabled, skipping")
		return nil
	}

	if cfg.AI.Endpoint == "" || cfg.AI.Model == "" {
		slog.Warn("AI review enabled but endpoint or model not configured, skipping")
		return nil
	}

	apiKey := os.Getenv(cfg.AI.APIKeyEnv)

	providerCfg := ai.ProviderConfig{
		Endpoint: cfg.AI.Endpoint,
		Model:    cfg.AI.Model,
		APIKey:   apiKey,
		Type:     ai.ProviderType(cfg.AI.Provider),
	}

	var provider ai.LLMProvider
	switch ai.ProviderType(cfg.AI.Provider) {
	case ai.ProviderOpenAICompatible, "":
		provider = providers.NewOpenAIProvider(providerCfg, 0)
	default:
		slog.Warn("AI review: unsupported provider, skipping", "provider", cfg.AI.Provider)
		return nil
	}

	reviewer := ai.NewReviewer(provider)

	if !reviewer.Available(ctx) {
		slog.Warn("AI review: LLM endpoint unreachable, skipping", "endpoint", cfg.AI.Endpoint)
		return nil
	}

	slog.Info("running AI review", "endpoint", cfg.AI.Endpoint, "model", cfg.AI.Model)

	result, err := reviewer.Review(ctx, diff, nil)
	if err != nil {
		slog.Error("AI review failed", "error", err)
		return nil
	}

	slog.Info("AI review complete", "findings", len(result.Findings), "duration", result.Duration)
	return result
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

// deduplicateCrossAnalyzer removes findings from AI review that overlap with
// static analyzer findings. When both a static analyzer and the AI reviewer
// flag the same issue (same file, nearby lines, similar description), the
// static analyzer finding is kept because it has more precise line numbers.
func deduplicateCrossAnalyzer(results []*interfaces.AnalysisResult) []*interfaces.AnalysisResult {
	if len(results) <= 1 {
		return results
	}

	// Collect all static (non-AI) findings.
	var staticFindings []interfaces.Finding
	for _, res := range results {
		if res.AnalyzerName == "ai-reviewer" {
			continue
		}
		staticFindings = append(staticFindings, res.Findings...)
	}

	if len(staticFindings) == 0 {
		return results
	}

	// For each AI result, remove findings that duplicate a static finding.
	for i, res := range results {
		if res.AnalyzerName != "ai-reviewer" {
			continue
		}

		kept := make([]interfaces.Finding, 0, len(res.Findings))
		removed := 0
		for _, aiFinding := range res.Findings {
			isDup := false
			for _, sf := range staticFindings {
				if ai.IsDuplicate(aiFinding, sf) {
					isDup = true
					break
				}
			}
			if isDup {
				removed++
			} else {
				kept = append(kept, aiFinding)
			}
		}

		if removed > 0 {
			slog.Info("cross-analyzer dedup removed AI findings duplicated by static analyzers",
				"removed", removed, "kept", len(kept))
			updated := *res
			updated.Findings = kept
			results[i] = &updated
		}
	}

	return results
}
