package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/analyzer"
	"github.com/toyinlola/shipsafe/pkg/cli"
	"github.com/toyinlola/shipsafe/pkg/interfaces"
	"github.com/toyinlola/shipsafe/pkg/report"
	"github.com/toyinlola/shipsafe/pkg/scorer"
	"github.com/toyinlola/shipsafe/pkg/vcs"
)

// -- Mock VCS Provider for CI integration tests --

type ciMockComment struct {
	prRef string
	body  string
}

type ciMockStatus struct {
	sha         string
	state       interfaces.StatusState
	description string
}

type ciMockVCSProvider struct {
	mu sync.Mutex

	diffToReturn *interfaces.Diff
	diffError    error

	comments     []ciMockComment
	commentError error

	statuses    []ciMockStatus
	statusError error
}

func (m *ciMockVCSProvider) GetDiff(ctx context.Context, prRef string) (*interfaces.Diff, error) {
	if m.diffError != nil {
		return nil, m.diffError
	}
	if m.diffToReturn == nil {
		return nil, errors.New("mock: no diff configured")
	}
	return m.diffToReturn, nil
}

func (m *ciMockVCSProvider) PostComment(ctx context.Context, prRef string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.commentError != nil {
		return m.commentError
	}
	m.comments = append(m.comments, ciMockComment{prRef: prRef, body: body})
	return nil
}

func (m *ciMockVCSProvider) SetStatus(ctx context.Context, sha string, status interfaces.StatusState, description string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.statusError != nil {
		return m.statusError
	}
	m.statuses = append(m.statuses, ciMockStatus{sha: sha, state: status, description: description})
	return nil
}

var _ interfaces.VCSProvider = (*ciMockVCSProvider)(nil)

// -- Canned diff content --

// cleanDiffRaw is a minimal clean diff with source + matching test file.
// Should produce GREEN (100) with zero findings.
const cleanDiffRaw = "diff --git a/pkg/utils/math.go b/pkg/utils/math.go\n" +
	"new file mode 100644\n" +
	"index 0000000..1234567\n" +
	"--- /dev/null\n" +
	"+++ b/pkg/utils/math.go\n" +
	"@@ -0,0 +1,11 @@\n" +
	"+package utils\n" +
	"+\n" +
	"+// Add returns the sum of two integers.\n" +
	"+func Add(a, b int) int {\n" +
	"+\treturn a + b\n" +
	"+}\n" +
	"+\n" +
	"+// Subtract returns the difference of two integers.\n" +
	"+func Subtract(a, b int) int {\n" +
	"+\treturn a - b\n" +
	"+}\n" +
	"diff --git a/pkg/utils/math_test.go b/pkg/utils/math_test.go\n" +
	"new file mode 100644\n" +
	"index 0000000..2345678\n" +
	"--- /dev/null\n" +
	"+++ b/pkg/utils/math_test.go\n" +
	"@@ -0,0 +1,15 @@\n" +
	"+package utils\n" +
	"+\n" +
	"+import \"testing\"\n" +
	"+\n" +
	"+func TestAdd(t *testing.T) {\n" +
	"+\tif Add(1, 2) != 3 {\n" +
	"+\t\tt.Error(\"expected 3\")\n" +
	"+\t}\n" +
	"+}\n" +
	"+\n" +
	"+func TestSubtract(t *testing.T) {\n" +
	"+\tif Subtract(3, 1) != 2 {\n" +
	"+\t\tt.Error(\"expected 2\")\n" +
	"+\t}\n" +
	"+}\n"

// secretsLeakDiffRaw contains hardcoded secrets with no test file.
// Should produce YELLOW or RED with secrets findings.
const secretsLeakDiffRaw = "diff --git a/internal/config/database.go b/internal/config/database.go\n" +
	"new file mode 100644\n" +
	"index 0000000..b7c8d9e\n" +
	"--- /dev/null\n" +
	"+++ b/internal/config/database.go\n" +
	"@@ -0,0 +1,8 @@\n" +
	"+package config\n" +
	"+\n" +
	"+const awsAccessKeyID = \"AKIAIOSFODNN7TJQMRWZ\"\n" +
	"+const awsSecretKey = \"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\"\n" +
	"+\n" +
	"+var DatabaseURL = \"postgres://admin:s3cr3tP4ss@db.prod.internal:5432/myapp\"\n" +
	"+\n" +
	"+var ServiceAuthToken = \"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N\"\n"

// -- Test Helpers --

func parseTestDiff(t *testing.T, raw string) *interfaces.Diff {
	t.Helper()
	parser := vcs.NewDiffParser()
	diff, err := parser.Parse(context.Background(), []byte(raw))
	if err != nil {
		t.Fatalf("parseTestDiff: %v", err)
	}
	return diff
}

type ciTestResult struct {
	score   *interfaces.TrustScore
	report  *interfaces.Report
	results []*interfaces.AnalysisResult
}

// runTestCIPipeline exercises the same logic as runCI without os.Exit.
// It runs: analyzers → score → report → post comment → set status.
func runTestCIPipeline(t *testing.T, provider interfaces.VCSProvider, diff *interfaces.Diff, cfg *cli.Config, env *ciEnvironment) *ciTestResult {
	t.Helper()
	ctx := context.Background()

	// Build analyzer registry and run analyzers.
	registry := analyzer.NewRegistry()
	registerAnalyzers(registry, cfg)

	engine := analyzer.NewEngine(registry)
	results, err := engine.Run(ctx, diff)
	if err != nil {
		t.Fatalf("engine.Run: %v", err)
	}

	// AI review is not run in tests (no API key, AI disabled).

	// Calculate trust score.
	calc := scorer.NewCalculator(
		scorer.WithThresholds(cfg.Thresholds.Green, cfg.Thresholds.Yellow),
	)
	trustScore := calc.Score(results)

	// Generate report.
	gen := report.NewGenerator()
	rpt := gen.Generate(results, trustScore, diff)

	// Post PR comment if configured.
	if cfg.CI.Comment && provider != nil && env.PRNumber != "" {
		postCIComment(ctx, provider, env, rpt, trustScore)
	}

	// Set commit status.
	if provider != nil && env.SHA != "" {
		setCIStatus(ctx, provider, env, trustScore)
	}

	return &ciTestResult{
		score:   trustScore,
		report:  rpt,
		results: results,
	}
}

func defaultTestConfig() *cli.Config {
	cfg := cli.DefaultConfig()
	cfg.CI.Comment = true
	cfg.AI.Enabled = false
	return cfg
}

func defaultTestEnv() *ciEnvironment {
	return &ciEnvironment{
		Provider: "github",
		PRNumber: "42",
		Owner:    "testorg",
		Repo:     "testrepo",
		SHA:      "abc123def456",
	}
}

// -- Integration Tests --

// TestCIFlow_FullPipeline_CleanDiff tests the entire CI flow end-to-end:
// detect environment → fetch diff → run analyzers → calculate score → post comment → set status.
func TestCIFlow_FullPipeline_CleanDiff(t *testing.T) {
	diff := parseTestDiff(t, cleanDiffRaw)
	mock := &ciMockVCSProvider{diffToReturn: diff}
	cfg := defaultTestConfig()
	env := defaultTestEnv()

	result := runTestCIPipeline(t, mock, diff, cfg, env)

	// Score should be GREEN.
	if result.score.Rating != interfaces.RatingGreen {
		t.Errorf("expected GREEN, got %s (score %d)", result.score.Rating, result.score.Score)
	}

	// Report should have been generated.
	if result.report == nil {
		t.Fatal("report is nil")
	}

	// PR comment should have been posted.
	if len(mock.comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(mock.comments))
	}
	if mock.comments[0].prRef != "42" {
		t.Errorf("comment posted to wrong PR: %q", mock.comments[0].prRef)
	}

	// Commit status should have been set to success.
	if len(mock.statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(mock.statuses))
	}
	if mock.statuses[0].state != interfaces.StatusSuccess {
		t.Errorf("expected status success, got %q", mock.statuses[0].state)
	}
	if mock.statuses[0].sha != "abc123def456" {
		t.Errorf("status set on wrong SHA: %q", mock.statuses[0].sha)
	}
}

// TestCIFlow_CleanDiff_ScoresGreen verifies a clean diff with source + tests
// produces a perfect score.
func TestCIFlow_CleanDiff_ScoresGreen(t *testing.T) {
	diff := parseTestDiff(t, cleanDiffRaw)
	mock := &ciMockVCSProvider{}
	cfg := defaultTestConfig()
	env := &ciEnvironment{Provider: "generic"}

	result := runTestCIPipeline(t, mock, diff, cfg, env)

	if result.score.Score != 100 {
		t.Errorf("clean diff score = %d, want 100", result.score.Score)
	}
	if result.score.Rating != interfaces.RatingGreen {
		t.Errorf("clean diff rating = %s, want GREEN", result.score.Rating)
	}
	if len(result.report.Findings) != 0 {
		t.Errorf("clean diff had %d findings, want 0:", len(result.report.Findings))
		for _, f := range result.report.Findings {
			t.Logf("  [%s/%s] %s: %s", f.Category, f.Severity, f.File, f.Title)
		}
	}
}

// TestCIFlow_SecretsLeak_FindingPresent verifies that a diff with hardcoded
// secrets produces a non-GREEN score with secrets findings.
func TestCIFlow_SecretsLeak_FindingPresent(t *testing.T) {
	diff := parseTestDiff(t, secretsLeakDiffRaw)
	mock := &ciMockVCSProvider{}
	cfg := defaultTestConfig()
	env := &ciEnvironment{Provider: "generic"}

	result := runTestCIPipeline(t, mock, diff, cfg, env)

	// Should not be GREEN.
	if result.score.Rating == interfaces.RatingGreen {
		t.Errorf("secrets-leak should not be GREEN, got score %d", result.score.Score)
	}

	// Must have at least one secrets finding.
	hasSecrets := false
	for _, f := range result.report.Findings {
		if f.Category == interfaces.CategorySecrets {
			hasSecrets = true
			break
		}
	}
	if !hasSecrets {
		t.Error("expected at least one secrets finding")
	}

	// Verify at least one HIGH severity secrets finding.
	hasHighSecrets := false
	for _, f := range result.report.Findings {
		if f.Category == interfaces.CategorySecrets && f.Severity == interfaces.SeverityHigh {
			hasHighSecrets = true
			break
		}
	}
	if !hasHighSecrets {
		t.Error("expected at least one HIGH severity secrets finding")
	}

	t.Logf("secrets-leak score: %d [%s], %d findings",
		result.score.Score, result.score.Rating, len(result.report.Findings))
}

// TestCIFlow_AIDisabled_OnlyStaticResults verifies that when AI is disabled,
// results contain only static analyzer output.
func TestCIFlow_AIDisabled_OnlyStaticResults(t *testing.T) {
	diff := parseTestDiff(t, cleanDiffRaw)
	mock := &ciMockVCSProvider{}
	cfg := defaultTestConfig()
	cfg.AI.Enabled = false
	env := &ciEnvironment{Provider: "generic"}

	result := runTestCIPipeline(t, mock, diff, cfg, env)

	// Verify no AI review results are present.
	for _, r := range result.results {
		if r.AnalyzerName == "ai-review" {
			t.Errorf("found AI review result when AI is disabled")
		}
	}

	// All five static analyzers should be present in results.
	expected := map[string]bool{
		"secrets":    false,
		"patterns":   false,
		"complexity": false,
		"coverage":   false,
		"imports":    false,
	}
	for _, r := range result.results {
		if _, ok := expected[r.AnalyzerName]; ok {
			expected[r.AnalyzerName] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected static analyzer %q in results", name)
		}
	}
}

// TestCIFlow_PRComment_ContainsTrustScore verifies the PR comment contains
// the trust score and rating.
func TestCIFlow_PRComment_ContainsTrustScore(t *testing.T) {
	diff := parseTestDiff(t, cleanDiffRaw)
	mock := &ciMockVCSProvider{}
	cfg := defaultTestConfig()
	env := defaultTestEnv()

	result := runTestCIPipeline(t, mock, diff, cfg, env)

	if len(mock.comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(mock.comments))
	}

	comment := mock.comments[0].body
	scoreStr := fmt.Sprintf("%d/100", result.score.Score)
	if !strings.Contains(comment, scoreStr) {
		t.Errorf("comment does not contain score %q:\n%.200s", scoreStr, comment)
	}

	ratingStr := string(result.score.Rating)
	if !strings.Contains(comment, ratingStr) {
		t.Errorf("comment does not contain rating %q", ratingStr)
	}
}

// TestCIFlow_CommitStatus_MappedCorrectly verifies commit status is set
// correctly for different trust ratings: GREEN→success, YELLOW→success, RED→failure.
func TestCIFlow_CommitStatus_MappedCorrectly(t *testing.T) {
	tests := []struct {
		name           string
		rating         interfaces.Rating
		score          int
		expectedStatus interfaces.StatusState
	}{
		{"GREEN_to_success", interfaces.RatingGreen, 90, interfaces.StatusSuccess},
		{"YELLOW_to_success", interfaces.RatingYellow, 65, interfaces.StatusSuccess},
		{"RED_to_failure", interfaces.RatingRed, 30, interfaces.StatusFailure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &ciMockVCSProvider{}
			env := defaultTestEnv()
			trustScore := &interfaces.TrustScore{
				Score:  tt.score,
				Rating: tt.rating,
			}

			setCIStatus(context.Background(), mock, env, trustScore)

			if len(mock.statuses) != 1 {
				t.Fatalf("expected 1 status, got %d", len(mock.statuses))
			}
			if mock.statuses[0].state != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, mock.statuses[0].state)
			}

			expectedDesc := fmt.Sprintf("ShipSafe: %d/100 %s", tt.score, tt.rating)
			if mock.statuses[0].description != expectedDesc {
				t.Errorf("expected description %q, got %q", expectedDesc, mock.statuses[0].description)
			}
		})
	}
}

// TestCIFlow_VCSUnreachable_GracefulHandling verifies the pipeline completes
// gracefully even when VCS operations fail.
func TestCIFlow_VCSUnreachable_GracefulHandling(t *testing.T) {
	diff := parseTestDiff(t, cleanDiffRaw)

	t.Run("comment_error", func(t *testing.T) {
		mock := &ciMockVCSProvider{
			commentError: errors.New("connection refused"),
		}
		cfg := defaultTestConfig()
		env := defaultTestEnv()

		// Pipeline should complete without panic.
		result := runTestCIPipeline(t, mock, diff, cfg, env)
		if result.score == nil {
			t.Fatal("score is nil")
		}
		if result.report == nil {
			t.Fatal("report is nil")
		}
		// Comment should NOT have been recorded (error occurred).
		if len(mock.comments) != 0 {
			t.Errorf("expected 0 comments after error, got %d", len(mock.comments))
		}
		// Status should still have been set (independent operation).
		if len(mock.statuses) != 1 {
			t.Errorf("expected 1 status even with comment error, got %d", len(mock.statuses))
		}
	})

	t.Run("status_error", func(t *testing.T) {
		mock := &ciMockVCSProvider{
			statusError: errors.New("unauthorized"),
		}
		cfg := defaultTestConfig()
		env := defaultTestEnv()

		// Pipeline should complete without panic.
		result := runTestCIPipeline(t, mock, diff, cfg, env)
		if result.score == nil {
			t.Fatal("score is nil")
		}
		// Status should NOT have been recorded (error occurred).
		if len(mock.statuses) != 0 {
			t.Errorf("expected 0 statuses after error, got %d", len(mock.statuses))
		}
		// Comment should still have been posted (independent operation).
		if len(mock.comments) != 1 {
			t.Errorf("expected 1 comment even with status error, got %d", len(mock.comments))
		}
	})

	t.Run("nil_provider", func(t *testing.T) {
		cfg := defaultTestConfig()
		env := &ciEnvironment{Provider: "generic"}

		// Pipeline should complete with nil provider (no VCS operations).
		result := runTestCIPipeline(t, nil, diff, cfg, env)
		if result.score == nil {
			t.Fatal("score is nil")
		}
		if result.report == nil {
			t.Fatal("report is nil")
		}
	})

	t.Run("all_vcs_errors", func(t *testing.T) {
		mock := &ciMockVCSProvider{
			commentError: errors.New("server unavailable"),
			statusError:  errors.New("server unavailable"),
		}
		cfg := defaultTestConfig()
		env := defaultTestEnv()

		// Pipeline should complete even when all VCS operations fail.
		result := runTestCIPipeline(t, mock, diff, cfg, env)
		if result.score == nil {
			t.Fatal("score is nil")
		}
		if result.report == nil {
			t.Fatal("report is nil")
		}
		if len(mock.comments) != 0 {
			t.Errorf("expected 0 comments, got %d", len(mock.comments))
		}
		if len(mock.statuses) != 0 {
			t.Errorf("expected 0 statuses, got %d", len(mock.statuses))
		}
	})
}
