package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/toyinlola/shipsafe/pkg/analyzer"
	"github.com/toyinlola/shipsafe/pkg/cli"
	"github.com/toyinlola/shipsafe/pkg/interfaces"
	"github.com/toyinlola/shipsafe/pkg/report"
	"github.com/toyinlola/shipsafe/pkg/scorer"
	"github.com/toyinlola/shipsafe/pkg/vcs"
)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Run analysis in CI mode with auto-detection and PR commenting",
	Long: `CI mode auto-detects the CI environment (GitHub Actions, Forgejo Actions,
GitLab CI) and runs the full ShipSafe analysis pipeline.

It extracts the PR diff, runs all enabled analyzers, calculates a trust score,
prints the report to stdout, and optionally posts results as a PR comment.

Exit code is determined by the fail_on config:
  fail_on: "red"    → exit 1 only on RED rating (default)
  fail_on: "yellow" → exit 1 on YELLOW or RED rating`,
	Args: cobra.NoArgs,
	RunE: runCI,
}

func init() {
	rootCmd.AddCommand(ciCmd)
}

// ciEnvironment holds detected CI platform metadata.
type ciEnvironment struct {
	Provider string // "github", "forgejo", "gitlab", "generic"
	PRNumber string
	Owner    string
	Repo     string
	SHA      string
	BaseSHA  string
	HeadSHA  string
}

// detectCIEnvironment inspects environment variables to determine the CI platform.
func detectCIEnvironment() *ciEnvironment {
	switch {
	case os.Getenv("FORGEJO_ACTIONS") == "true" || os.Getenv("GITEA_ACTIONS") == "true":
		return detectForgejo()
	case os.Getenv("GITHUB_ACTIONS") == "true":
		return detectGitHub()
	case os.Getenv("GITLAB_CI") == "true":
		return detectGitLab()
	default:
		return &ciEnvironment{Provider: "generic"}
	}
}

func detectGitHub() *ciEnvironment {
	env := &ciEnvironment{Provider: "github"}

	if repository := os.Getenv("GITHUB_REPOSITORY"); repository != "" {
		parts := strings.SplitN(repository, "/", 2)
		if len(parts) == 2 {
			env.Owner = parts[0]
			env.Repo = parts[1]
		}
	}

	env.SHA = os.Getenv("GITHUB_SHA")

	// For pull_request events, extract PR number from GITHUB_REF (refs/pull/123/merge).
	ref := os.Getenv("GITHUB_REF")
	if strings.HasPrefix(ref, "refs/pull/") {
		parts := strings.Split(ref, "/")
		if len(parts) >= 3 {
			env.PRNumber = parts[2]
		}
	}

	env.BaseSHA = os.Getenv("GITHUB_BASE_REF")
	env.HeadSHA = os.Getenv("GITHUB_HEAD_REF")

	return env
}

func detectForgejo() *ciEnvironment {
	env := &ciEnvironment{Provider: "forgejo"}

	// Forgejo Actions uses GitHub-compatible env vars for repository info.
	if repository := os.Getenv("GITHUB_REPOSITORY"); repository != "" {
		parts := strings.SplitN(repository, "/", 2)
		if len(parts) == 2 {
			env.Owner = parts[0]
			env.Repo = parts[1]
		}
	}

	env.SHA = os.Getenv("GITHUB_SHA")

	ref := os.Getenv("GITHUB_REF")
	if strings.HasPrefix(ref, "refs/pull/") {
		parts := strings.Split(ref, "/")
		if len(parts) >= 3 {
			env.PRNumber = parts[2]
		}
	}

	env.BaseSHA = os.Getenv("GITHUB_BASE_REF")
	env.HeadSHA = os.Getenv("GITHUB_HEAD_REF")

	return env
}

func detectGitLab() *ciEnvironment {
	env := &ciEnvironment{Provider: "gitlab"}

	env.PRNumber = os.Getenv("CI_MERGE_REQUEST_IID")
	env.SHA = os.Getenv("CI_COMMIT_SHA")
	env.BaseSHA = os.Getenv("CI_MERGE_REQUEST_DIFF_BASE_SHA")

	// GitLab uses CI_PROJECT_PATH for owner/repo.
	if projectPath := os.Getenv("CI_PROJECT_PATH"); projectPath != "" {
		parts := strings.SplitN(projectPath, "/", 2)
		if len(parts) == 2 {
			env.Owner = parts[0]
			env.Repo = parts[1]
		}
	}

	return env
}

// createVCSProvider creates a VCSProvider for the detected CI environment.
// Returns nil if no provider can be created (e.g., generic environment).
func createVCSProvider(env *ciEnvironment) interfaces.VCSProvider {
	switch env.Provider {
	case "github":
		provider, err := vcs.NewGitHubProviderFromEnv()
		if err != nil {
			slog.Warn("ci: could not create GitHub provider", "error", err)
			return nil
		}
		return provider
	case "forgejo":
		provider, err := vcs.NewForgejoProviderFromEnv()
		if err != nil {
			slog.Warn("ci: could not create Forgejo provider", "error", err)
			return nil
		}
		return provider
	default:
		return nil
	}
}

func runCI(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// 1. Load configuration.
	cfg, err := cli.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("ci: %w", err)
	}

	slog.Debug("config loaded",
		"thresholds.green", cfg.Thresholds.Green,
		"thresholds.yellow", cfg.Thresholds.Yellow,
		"fail_on", cfg.CI.FailOn,
	)

	// 2. Detect CI environment.
	env := detectCIEnvironment()
	slog.Info("CI environment detected",
		"provider", env.Provider,
		"pr", env.PRNumber,
		"owner", env.Owner,
		"repo", env.Repo,
		"sha", env.SHA,
	)

	// 3. Create VCS provider (may be nil for generic env).
	vcsProvider := createVCSProvider(env)

	// 4. Get the diff.
	diff, err := getCIDiff(ctx, env, vcsProvider)
	if err != nil {
		return fmt.Errorf("ci: %w", err)
	}

	slog.Info("diff parsed", "files", len(diff.Files))

	// 5. Build analyzer registry and run analyzers.
	registry := analyzer.NewRegistry()
	registerAnalyzers(registry, cfg)

	engine := analyzer.NewEngine(registry)
	results, err := engine.Run(ctx, diff)
	if err != nil {
		return fmt.Errorf("ci: running analysis: %w", err)
	}

	// 6. Run AI review if enabled.
	if aiResult := runAIReview(ctx, cfg, diff); aiResult != nil {
		results = append(results, aiResult)
	}

	// 6b. Deduplicate findings across static analyzers and AI review.
	results = deduplicateCrossAnalyzer(results)

	// 7. Calculate trust score.
	calc := scorer.NewCalculator(
		scorer.WithThresholds(cfg.Thresholds.Green, cfg.Thresholds.Yellow),
	)
	trustScore := calc.Score(results)

	// 8. Generate report.
	gen := report.NewGenerator()
	rpt := gen.Generate(results, trustScore, diff)

	// 9. Print terminal report to stdout.
	termFmt := report.NewTerminalFormatter()
	if err := termFmt.Format(os.Stdout, rpt); err != nil {
		return fmt.Errorf("ci: writing terminal report: %w", err)
	}

	// 10. Post PR comment if configured and a VCS provider is available.
	if cfg.CI.Comment && vcsProvider != nil && env.PRNumber != "" {
		postCIComment(ctx, vcsProvider, env, rpt, trustScore)
	}

	// 11. Set commit status if VCS provider is available and we have a SHA.
	if vcsProvider != nil && env.SHA != "" {
		setCIStatus(ctx, vcsProvider, env, trustScore)
	}

	// 12. Exit with appropriate code based on config.
	if shouldFail(cfg.CI.FailOn, trustScore.Rating) {
		os.Exit(1)
	}

	return nil
}

// getCIDiff obtains the diff for CI analysis.
// Priority: VCS provider (if PR ref available) → git diff with base/head → git diff HEAD~1.
func getCIDiff(ctx context.Context, env *ciEnvironment, provider interfaces.VCSProvider) (*interfaces.Diff, error) {
	parser := vcs.NewDiffParser()

	// Try VCS provider first if we have a PR number.
	if provider != nil && env.PRNumber != "" {
		slog.Info("fetching diff from VCS provider", "pr", env.PRNumber)
		diff, err := provider.GetDiff(ctx, env.PRNumber)
		if err == nil {
			return diff, nil
		}
		slog.Warn("VCS provider diff failed, falling back to git", "error", err)
	}

	// Try git diff between base and head SHAs.
	if env.BaseSHA != "" {
		slog.Info("running git diff", "base", env.BaseSHA, "head", env.HeadSHA)
		headRef := env.HeadSHA
		if headRef == "" {
			headRef = "HEAD"
		}
		return gitDiff(ctx, parser, env.BaseSHA, headRef)
	}

	// Fallback: diff against previous commit.
	slog.Info("falling back to git diff HEAD~1")
	return gitDiff(ctx, parser, "HEAD~1", "HEAD")
}

// gitDiff runs `git diff base...head` and parses the output.
func gitDiff(ctx context.Context, parser interfaces.DiffParser, base, head string) (*interfaces.Diff, error) {
	diffRange := fmt.Sprintf("%s...%s", base, head)
	gitCmd := exec.CommandContext(ctx, "git", "diff", diffRange)

	out, err := gitCmd.Output()
	if err != nil {
		// If three-dot fails (e.g., detached HEAD), try two-dot.
		gitCmd = exec.CommandContext(ctx, "git", "diff", base, head)
		out, err = gitCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("running git diff %s %s: %w", base, head, err)
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no changes found (git diff returned empty)")
	}

	return parser.Parse(ctx, out)
}

// postCIComment posts the analysis report as a PR comment.
func postCIComment(ctx context.Context, provider interfaces.VCSProvider, env *ciEnvironment, rpt *interfaces.Report, score *interfaces.TrustScore) {
	var buf bytes.Buffer

	// Write summary header line.
	badge := summaryBadge(score.Rating)
	fmt.Fprintf(&buf, "%s **ShipSafe: %d/100 %s**\n\n", badge, score.Score, score.Rating)

	// Write full markdown report.
	mdFmt := report.NewMarkdownFormatter()
	if err := mdFmt.Format(&buf, rpt); err != nil {
		slog.Error("ci: generating markdown report for comment", "error", err)
		return
	}

	if err := provider.PostComment(ctx, env.PRNumber, buf.String()); err != nil {
		slog.Error("ci: posting PR comment", "error", err)
		return
	}

	slog.Info("PR comment posted", "pr", env.PRNumber)
}

// setCIStatus sets the commit status on the head SHA.
func setCIStatus(ctx context.Context, provider interfaces.VCSProvider, env *ciEnvironment, score *interfaces.TrustScore) {
	var status interfaces.StatusState
	switch score.Rating {
	case interfaces.RatingGreen:
		status = interfaces.StatusSuccess
	case interfaces.RatingYellow:
		status = interfaces.StatusSuccess
	case interfaces.RatingRed:
		status = interfaces.StatusFailure
	default:
		status = interfaces.StatusError
	}

	description := fmt.Sprintf("ShipSafe: %d/100 %s", score.Score, score.Rating)

	if err := provider.SetStatus(ctx, env.SHA, status, description); err != nil {
		slog.Error("ci: setting commit status", "error", err)
		return
	}

	slog.Info("commit status set", "sha", env.SHA, "status", status)
}

// shouldFail determines if CI should exit with a failure code.
func shouldFail(failOn string, rating interfaces.Rating) bool {
	switch failOn {
	case "yellow":
		return rating == interfaces.RatingRed || rating == interfaces.RatingYellow
	default: // "red"
		return rating == interfaces.RatingRed
	}
}

// summaryBadge returns the emoji badge for a rating.
func summaryBadge(rating interfaces.Rating) string {
	switch rating {
	case interfaces.RatingGreen:
		return "🟢"
	case interfaces.RatingYellow:
		return "🟡"
	case interfaces.RatingRed:
		return "🔴"
	default:
		return "⚪"
	}
}
