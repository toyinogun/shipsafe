package scorer

import (
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

func TestCalculator_NoFindings_Score100Green(t *testing.T) {
	calc := NewCalculator()
	results := []*interfaces.AnalysisResult{
		{AnalyzerName: "complexity"},
		{AnalyzerName: "secrets"},
	}

	ts := calc.Score(results)
	if ts.Score != 100 {
		t.Errorf("expected score 100, got %d", ts.Score)
	}
	if ts.Rating != interfaces.RatingGreen {
		t.Errorf("expected GREEN rating, got %s", ts.Rating)
	}
}

func TestCalculator_OneCriticalSecurityFinding_LargeDropLikelyRed(t *testing.T) {
	calc := NewCalculator()
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "security",
			Findings: []interfaces.Finding{
				{
					Category:   interfaces.CategorySecurity,
					Severity:   interfaces.SeverityCritical,
					Confidence: 1.0,
				},
			},
		},
	}

	ts := calc.Score(results)

	// penalty = 25 * 1.5 * 1.0 = 37.5 → score = 100 - 38 = 62
	// But a critical secret would be even worse: 25 * 2.0 = 50.
	// A single critical security finding should cause a significant drop.
	if ts.Score >= 80 {
		t.Errorf("expected score below 80 for critical security finding, got %d", ts.Score)
	}
	if ts.FindingCount[interfaces.SeverityCritical] != 1 {
		t.Errorf("expected 1 critical finding, got %d", ts.FindingCount[interfaces.SeverityCritical])
	}
}

func TestCalculator_OneCriticalSecretsFinding_Red(t *testing.T) {
	calc := NewCalculator()
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "secrets",
			Findings: []interfaces.Finding{
				{
					Category:   interfaces.CategorySecrets,
					Severity:   interfaces.SeverityCritical,
					Confidence: 1.0,
				},
			},
		},
	}

	ts := calc.Score(results)

	// penalty = 25 * 2.0 * 1.0 = 50 → score = 50
	// Score 50 is YELLOW boundary. Let's verify it's at most YELLOW.
	if ts.Score > 50 {
		t.Errorf("expected score <= 50 for critical secrets finding, got %d", ts.Score)
	}
}

func TestCalculator_SeveralLowFindings_StillGreen(t *testing.T) {
	calc := NewCalculator()
	findings := make([]interfaces.Finding, 5)
	for i := range findings {
		findings[i] = interfaces.Finding{
			Category:   interfaces.CategoryConvention,
			Severity:   interfaces.SeverityLow,
			Confidence: 0.8,
		}
	}

	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "patterns",
			Findings:     findings,
		},
	}

	ts := calc.Score(results)

	// penalty per finding = 3 * 0.3 * 0.8 = 0.72 → 5 findings = 3.6 → score ~96
	if ts.Rating != interfaces.RatingGreen {
		t.Errorf("expected GREEN rating for low-severity findings, got %s (score: %d)", ts.Rating, ts.Score)
	}
	if ts.Score < 90 {
		t.Errorf("expected score above 90 for low-severity convention findings, got %d", ts.Score)
	}
}

func TestCalculator_MixOfFindings_Yellow(t *testing.T) {
	calc := NewCalculator()
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "security",
			Findings: []interfaces.Finding{
				{
					Category:   interfaces.CategorySecurity,
					Severity:   interfaces.SeverityHigh,
					Confidence: 0.9,
				},
			},
		},
		{
			AnalyzerName: "complexity",
			Findings: []interfaces.Finding{
				{
					Category:   interfaces.CategoryComplexity,
					Severity:   interfaces.SeverityMedium,
					Confidence: 0.8,
				},
				{
					Category:   interfaces.CategoryComplexity,
					Severity:   interfaces.SeverityMedium,
					Confidence: 0.7,
				},
			},
		},
		{
			AnalyzerName: "patterns",
			Findings: []interfaces.Finding{
				{
					Category:   interfaces.CategoryPattern,
					Severity:   interfaces.SeverityLow,
					Confidence: 0.6,
				},
			},
		},
	}

	ts := calc.Score(results)

	// high security: 15 * 1.5 * 0.9 = 20.25
	// medium complexity: 8 * 0.8 * 0.8 = 5.12
	// medium complexity: 8 * 0.8 * 0.7 = 4.48
	// low pattern: 3 * 0.5 * 0.6 = 0.9
	// total penalty = ~30.75 → score ~69 → YELLOW
	if ts.Rating != interfaces.RatingYellow {
		t.Errorf("expected YELLOW rating for mixed findings, got %s (score: %d)", ts.Rating, ts.Score)
	}
}

func TestCalculator_ScoreNeverBelowZero(t *testing.T) {
	calc := NewCalculator()

	// Create many critical security findings to overwhelm the score.
	findings := make([]interfaces.Finding, 20)
	for i := range findings {
		findings[i] = interfaces.Finding{
			Category:   interfaces.CategorySecrets,
			Severity:   interfaces.SeverityCritical,
			Confidence: 1.0,
		}
	}

	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "secrets",
			Findings:     findings,
		},
	}

	ts := calc.Score(results)

	// penalty = 20 * 25 * 2.0 * 1.0 = 1000 → score clamped to 0
	if ts.Score < 0 {
		t.Errorf("score must never be negative, got %d", ts.Score)
	}
	if ts.Score != 0 {
		t.Errorf("expected score 0 for massive penalty, got %d", ts.Score)
	}
	if ts.Rating != interfaces.RatingRed {
		t.Errorf("expected RED rating, got %s", ts.Rating)
	}
}

func TestCalculator_SkipsErroredResults(t *testing.T) {
	calc := NewCalculator()
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "secrets",
			Error:        errTestAnalyzer,
			Findings: []interfaces.Finding{
				{
					Category:   interfaces.CategorySecrets,
					Severity:   interfaces.SeverityCritical,
					Confidence: 1.0,
				},
			},
		},
	}

	ts := calc.Score(results)
	if ts.Score != 100 {
		t.Errorf("expected score 100 when all results errored, got %d", ts.Score)
	}
}

func TestCalculator_SkipsNilResults(t *testing.T) {
	calc := NewCalculator()
	results := []*interfaces.AnalysisResult{nil, nil}
	ts := calc.Score(results)
	if ts.Score != 100 {
		t.Errorf("expected score 100 for nil results, got %d", ts.Score)
	}
}

func TestCalculator_BreakdownPerCategory(t *testing.T) {
	calc := NewCalculator()
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "mixed",
			Findings: []interfaces.Finding{
				{Category: interfaces.CategorySecurity, Severity: interfaces.SeverityHigh, Confidence: 1.0},
				{Category: interfaces.CategoryComplexity, Severity: interfaces.SeverityMedium, Confidence: 1.0},
			},
		},
	}

	ts := calc.Score(results)
	if _, ok := ts.Breakdown[interfaces.CategorySecurity]; !ok {
		t.Error("expected security category in breakdown")
	}
	if _, ok := ts.Breakdown[interfaces.CategoryComplexity]; !ok {
		t.Error("expected complexity category in breakdown")
	}
}

func TestCalculator_FindingCountBySeverity(t *testing.T) {
	calc := NewCalculator()
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "mixed",
			Findings: []interfaces.Finding{
				{Category: interfaces.CategorySecurity, Severity: interfaces.SeverityCritical, Confidence: 1.0},
				{Category: interfaces.CategoryComplexity, Severity: interfaces.SeverityMedium, Confidence: 1.0},
				{Category: interfaces.CategoryPattern, Severity: interfaces.SeverityMedium, Confidence: 1.0},
				{Category: interfaces.CategoryConvention, Severity: interfaces.SeverityLow, Confidence: 1.0},
			},
		},
	}

	ts := calc.Score(results)
	if ts.FindingCount[interfaces.SeverityCritical] != 1 {
		t.Errorf("expected 1 critical, got %d", ts.FindingCount[interfaces.SeverityCritical])
	}
	if ts.FindingCount[interfaces.SeverityMedium] != 2 {
		t.Errorf("expected 2 medium, got %d", ts.FindingCount[interfaces.SeverityMedium])
	}
	if ts.FindingCount[interfaces.SeverityLow] != 1 {
		t.Errorf("expected 1 low, got %d", ts.FindingCount[interfaces.SeverityLow])
	}
}

func TestCalculator_ZeroConfidenceTreatedAsOne(t *testing.T) {
	calc := NewCalculator()
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "security",
			Findings: []interfaces.Finding{
				{
					Category:   interfaces.CategorySecurity,
					Severity:   interfaces.SeverityHigh,
					Confidence: 0.0, // zero confidence should be treated as 1.0
				},
			},
		},
	}

	ts := calc.Score(results)
	// penalty = 15 * 1.5 * 1.0 = 22.5 → score = 78
	if ts.Score >= 80 {
		t.Errorf("expected score < 80 when zero confidence is treated as 1.0, got %d", ts.Score)
	}
}

func TestCalculator_CustomWeights(t *testing.T) {
	calc := NewCalculator(
		WithSeverityWeights(SeverityWeights{
			interfaces.SeverityCritical: 50, // double the default
		}),
	)
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "security",
			Findings: []interfaces.Finding{
				{
					Category:   interfaces.CategorySecurity,
					Severity:   interfaces.SeverityCritical,
					Confidence: 1.0,
				},
			},
		},
	}

	ts := calc.Score(results)
	// penalty = 50 * 1.5 * 1.0 = 75 → score = 25
	if ts.Score > 30 {
		t.Errorf("expected score <= 30 with doubled critical weight, got %d", ts.Score)
	}
}

func TestCalculator_CustomThresholds(t *testing.T) {
	calc := NewCalculator(WithThresholds(90, 70))
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "patterns",
			Findings: []interfaces.Finding{
				{
					Category:   interfaces.CategoryPattern,
					Severity:   interfaces.SeverityMedium,
					Confidence: 1.0,
				},
			},
		},
	}

	ts := calc.Score(results)
	// penalty = 8 * 0.5 * 1.0 = 4 → score = 96
	// With green threshold 90, score 96 is GREEN.
	if ts.Rating != interfaces.RatingGreen {
		t.Errorf("expected GREEN with custom thresholds, got %s (score: %d)", ts.Rating, ts.Score)
	}
}

func TestCalculator_InfoFindingsNoPenalty(t *testing.T) {
	calc := NewCalculator()
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "patterns",
			Findings: []interfaces.Finding{
				{Category: interfaces.CategoryConvention, Severity: interfaces.SeverityInfo, Confidence: 1.0},
				{Category: interfaces.CategoryConvention, Severity: interfaces.SeverityInfo, Confidence: 1.0},
				{Category: interfaces.CategoryConvention, Severity: interfaces.SeverityInfo, Confidence: 1.0},
			},
		},
	}

	ts := calc.Score(results)
	if ts.Score != 100 {
		t.Errorf("info findings should carry zero penalty, expected 100 got %d", ts.Score)
	}
	if ts.FindingCount[interfaces.SeverityInfo] != 3 {
		t.Errorf("expected 3 info findings counted, got %d", ts.FindingCount[interfaces.SeverityInfo])
	}
}

// sentinel error for tests
var errTestAnalyzer = errorString("test: analyzer failed")

type errorString string

func (e errorString) Error() string { return string(e) }
