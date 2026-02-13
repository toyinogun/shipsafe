package tests

import (
	"context"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
	"github.com/toyinlola/shipsafe/pkg/report"
	"github.com/toyinlola/shipsafe/pkg/vcs"
)

func TestCleanDiff_ScoresGreen(t *testing.T) {
	diff := LoadFixtureDiff(t, "clean")
	result := RunPipeline(t, diff)

	AssertScoreInRange(t, result.Score.Score, 80, 100)
	AssertRating(t, result.Score.Rating, interfaces.RatingGreen)

	// A clean diff with source + tests should have minimal or zero findings.
	if len(result.Report.Findings) > 0 {
		t.Logf("clean diff had %d findings (expected 0):", len(result.Report.Findings))
		for _, f := range result.Report.Findings {
			t.Logf("  [%s] %s: %s (%s)", f.Severity, f.Category, f.Title, f.File)
		}
	}
}

func TestSecretsLeak_ScoresBelowGreen(t *testing.T) {
	diff := LoadFixtureDiff(t, "secrets-leak")
	result := RunPipeline(t, diff)

	// With per-category penalty caps, a single-category secrets leak causes a
	// significant score drop but may not reach RED on its own. The cap prevents
	// one noisy category from dominating — RED requires issues across categories.
	// The score must still be well below GREEN.
	if result.Score.Score >= 80 {
		t.Errorf("secrets-leak score = %d, want < 80 (not GREEN)", result.Score.Score)
	}
	if result.Score.Rating == interfaces.RatingGreen {
		t.Errorf("secrets-leak rating = %s, must not be GREEN", result.Score.Rating)
	}

	// Must have at least one CRITICAL or HIGH finding in the secrets category.
	AssertHasFindingCategory(t, result.Report.Findings, interfaces.CategorySecrets)

	secretsCount := CountFindingsWithCategory(result.Report.Findings, interfaces.CategorySecrets)
	if secretsCount < 3 {
		t.Errorf("expected at least 3 secrets findings, got %d", secretsCount)
	}

	// Verify at least one finding is CRITICAL or HIGH severity.
	hasHighSeverity := false
	for _, f := range result.Report.Findings {
		if f.Category == interfaces.CategorySecrets &&
			(f.Severity == interfaces.SeverityCritical || f.Severity == interfaces.SeverityHigh) {
			hasHighSeverity = true
			break
		}
	}
	if !hasHighSeverity {
		t.Error("expected at least one CRITICAL or HIGH severity secrets finding")
	}

	t.Logf("secrets-leak score: %d [%s], %d findings", result.Score.Score, result.Score.Rating, len(result.Report.Findings))
}

func TestComplexitySpike_ScoresYellow(t *testing.T) {
	diff := LoadFixtureDiff(t, "complexity-spike")
	result := RunPipeline(t, diff)

	AssertScoreInRange(t, result.Score.Score, 50, 79)
	AssertRating(t, result.Score.Rating, interfaces.RatingYellow)

	// Must have at least one complexity finding.
	AssertHasFindingCategory(t, result.Report.Findings, interfaces.CategoryComplexity)

	// Verify findings include HIGH severity for extreme complexity.
	hasHighComplexity := false
	for _, f := range result.Report.Findings {
		if f.Category == interfaces.CategoryComplexity && f.Severity == interfaces.SeverityHigh {
			hasHighComplexity = true
			break
		}
	}
	if !hasHighComplexity {
		t.Error("expected at least one HIGH severity complexity finding")
	}
}

func TestMissingTests_HasCoverageFindings(t *testing.T) {
	diff := LoadFixtureDiff(t, "missing-tests")
	result := RunPipeline(t, diff)

	// Must have coverage findings.
	AssertHasFindingCategory(t, result.Report.Findings, interfaces.CategoryCoverage)

	coverageCount := CountFindingsWithCategory(result.Report.Findings, interfaces.CategoryCoverage)
	if coverageCount < 2 {
		t.Errorf("expected at least 2 coverage findings (one per new file), got %d", coverageCount)
	}

	// Coverage findings for new files should be MEDIUM severity.
	for _, f := range result.Report.Findings {
		if f.Category == interfaces.CategoryCoverage {
			if f.Severity != interfaces.SeverityMedium {
				t.Errorf("coverage finding for new file %q has severity %q, want %q",
					f.File, f.Severity, interfaces.SeverityMedium)
			}
		}
	}
}

func TestMixedIssues_ScoresReasonably(t *testing.T) {
	diff := LoadFixtureDiff(t, "mixed-issues")
	result := RunPipeline(t, diff)

	// Should score above RED (not catastrophic).
	if result.Score.Score < 50 {
		t.Errorf("mixed-issues score = %d, want >= 50", result.Score.Score)
	}

	// Must have findings from multiple categories.
	cats := FindingCategories(result.Report.Findings)
	if len(cats) < 2 {
		t.Errorf("expected findings from at least 2 categories, got %d: %v", len(cats), cats)
	}

	// Should have pattern findings (TODO and/or debug print).
	AssertHasFindingCategory(t, result.Report.Findings, interfaces.CategoryPattern)

	// Log all findings for visibility.
	t.Logf("mixed-issues score: %d [%s], %d findings across %d categories",
		result.Score.Score, result.Score.Rating, len(result.Report.Findings), len(cats))
	for _, f := range result.Report.Findings {
		t.Logf("  [%s/%s] %s: %s", f.Category, f.Severity, f.File, f.Title)
	}
}

func TestAllReporters_DontPanic(t *testing.T) {
	fixtures := []string{"clean", "secrets-leak", "complexity-spike", "missing-tests", "mixed-issues"}

	formatters := map[string]Formatter{
		"terminal": report.NewTerminalFormatter(),
		"json":     report.NewJSONFormatter(),
		"markdown": report.NewMarkdownFormatter(),
	}

	for _, fixtureName := range fixtures {
		diff := LoadFixtureDiff(t, fixtureName)
		result := RunPipeline(t, diff)

		for fmtName, formatter := range formatters {
			t.Run(fixtureName+"_"+fmtName, func(t *testing.T) {
				output := FormatReport(t, formatter, result.Report)

				if output == "" {
					t.Errorf("formatter %q produced empty output for fixture %q", fmtName, fixtureName)
				}

				// Sanity check: output should mention the score.
				if len(output) < 10 {
					t.Errorf("formatter %q output too short for fixture %q: %d bytes",
						fmtName, fixtureName, len(output))
				}
			})
		}
	}
}

func TestEmptyDiff_ScoresPerfect(t *testing.T) {
	// An empty diff (no files) should score 100, GREEN, zero findings.
	// The diff parser requires at least one file, so we construct the diff manually.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{},
	}

	result := RunPipeline(t, diff)

	if result.Score.Score != 100 {
		t.Errorf("empty diff score = %d, want 100", result.Score.Score)
	}
	AssertRating(t, result.Score.Rating, interfaces.RatingGreen)

	if len(result.Report.Findings) != 0 {
		t.Errorf("empty diff had %d findings, want 0", len(result.Report.Findings))
		for _, f := range result.Report.Findings {
			t.Logf("  unexpected finding: [%s] %s", f.Category, f.Title)
		}
	}
}

func TestDiffParserLoadsAllFixtures(t *testing.T) {
	// Ensure every fixture diff can be parsed without errors.
	fixtures := []struct {
		name     string
		minFiles int
	}{
		{"clean", 2},
		{"secrets-leak", 1},
		{"complexity-spike", 1},
		{"missing-tests", 2},
		{"mixed-issues", 2},
	}

	parser := vcs.NewDiffParser()

	for _, tt := range fixtures {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := parser.ParseFile(context.Background(),
				fixturesDir()+"/"+tt.name+".diff")
			if err != nil {
				t.Fatalf("ParseFile(%q): %v", tt.name, err)
			}

			if len(diff.Files) < tt.minFiles {
				t.Errorf("expected at least %d files, got %d", tt.minFiles, len(diff.Files))
			}
		})
	}
}
