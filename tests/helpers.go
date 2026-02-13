// Package tests provides integration test utilities for the ShipSafe pipeline.
package tests

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/analyzer"
	"github.com/toyinlola/shipsafe/pkg/interfaces"
	"github.com/toyinlola/shipsafe/pkg/report"
	"github.com/toyinlola/shipsafe/pkg/scorer"
	"github.com/toyinlola/shipsafe/pkg/vcs"
)

// fixturesDir returns the absolute path to the test fixtures/diffs directory.
func fixturesDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "fixtures", "diffs")
}

// LoadFixtureDiff parses a fixture diff file by name (e.g., "clean" loads "clean.diff").
func LoadFixtureDiff(t *testing.T, name string) *interfaces.Diff {
	t.Helper()

	path := filepath.Join(fixturesDir(), name+".diff")
	parser := vcs.NewDiffParser()

	diff, err := parser.ParseFile(context.Background(), path)
	if err != nil {
		t.Fatalf("LoadFixtureDiff(%q): %v", name, err)
	}

	return diff
}

// PipelineResult holds the output of a full pipeline run.
type PipelineResult struct {
	Diff    *interfaces.Diff
	Results []*interfaces.AnalysisResult
	Score   *interfaces.TrustScore
	Report  *interfaces.Report
}

// RunPipeline executes the full analysis pipeline (parse → analyze → score → report)
// and returns all intermediate results.
func RunPipeline(t *testing.T, diff *interfaces.Diff) *PipelineResult {
	t.Helper()
	ctx := context.Background()

	// Set up the analyzer registry with all analyzers.
	registry := analyzer.NewRegistry()
	for _, a := range []analyzer.Analyzer{
		analyzer.NewComplexityAnalyzer(),
		analyzer.NewCoverageAnalyzer(),
		analyzer.NewSecretsAnalyzer(),
		analyzer.NewPatternsAnalyzer(),
		analyzer.NewImportsAnalyzer(),
	} {
		if err := registry.Register(a); err != nil {
			t.Fatalf("registering analyzer %s: %v", a.Name(), err)
		}
	}

	// Run all analyzers.
	engine := analyzer.NewEngine(registry)
	results, err := engine.Run(ctx, diff)
	if err != nil {
		t.Fatalf("engine.Run: %v", err)
	}

	// Calculate trust score.
	calc := scorer.NewCalculator()
	score := calc.Score(results)

	// Generate report.
	gen := report.NewGenerator()
	rpt := gen.Generate(results, score, diff)

	return &PipelineResult{
		Diff:    diff,
		Results: results,
		Score:   score,
		Report:  rpt,
	}
}

// AssertScoreInRange asserts that the trust score falls within [min, max] inclusive.
func AssertScoreInRange(t *testing.T, score int, min, max int) {
	t.Helper()
	if score < min || score > max {
		t.Errorf("score %d is outside expected range [%d, %d]", score, min, max)
	}
}

// AssertRating asserts that the trust score has the expected rating.
func AssertRating(t *testing.T, got, want interfaces.Rating) {
	t.Helper()
	if got != want {
		t.Errorf("rating = %q, want %q", got, want)
	}
}

// AssertHasFindingCategory checks that at least one finding has the given category.
func AssertHasFindingCategory(t *testing.T, findings []interfaces.Finding, cat interfaces.Category) {
	t.Helper()
	for _, f := range findings {
		if f.Category == cat {
			return
		}
	}
	t.Errorf("no finding with category %q found in %d findings", cat, len(findings))
}

// AssertHasFindingSeverity checks that at least one finding has the given severity.
func AssertHasFindingSeverity(t *testing.T, findings []interfaces.Finding, sev interfaces.Severity) {
	t.Helper()
	for _, f := range findings {
		if f.Severity == sev {
			return
		}
	}
	t.Errorf("no finding with severity %q found in %d findings", sev, len(findings))
}

// CountFindingsWithCategory returns the number of findings matching the given category.
func CountFindingsWithCategory(findings []interfaces.Finding, cat interfaces.Category) int {
	count := 0
	for _, f := range findings {
		if f.Category == cat {
			count++
		}
	}
	return count
}

// FindingCategories returns the unique set of categories present in findings.
func FindingCategories(findings []interfaces.Finding) map[interfaces.Category]bool {
	cats := make(map[interfaces.Category]bool)
	for _, f := range findings {
		cats[f.Category] = true
	}
	return cats
}

// Formatter is the interface shared by all report formatters.
type Formatter interface {
	Format(w io.Writer, report *interfaces.Report) error
}

// FormatReport formats a report using the given formatter and returns the output as a string.
func FormatReport(t *testing.T, formatter Formatter, rpt *interfaces.Report) string {
	t.Helper()
	var buf bytes.Buffer
	if err := formatter.Format(&buf, rpt); err != nil {
		t.Fatalf("formatter.Format: %v", err)
	}
	return buf.String()
}
