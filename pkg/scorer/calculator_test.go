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

func TestCalculator_OneCriticalSecretsFinding_CappedPenalty(t *testing.T) {
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

	// penalty = 25 * 2.0 * 1.0 = 50, capped at 40 (secrets with critical) → score = 60
	// Has critical → no floor. A single critical secret should cause a significant drop.
	if ts.Score > 65 {
		t.Errorf("expected score <= 65 for critical secrets finding (capped), got %d", ts.Score)
	}
	if ts.Score < 55 {
		t.Errorf("expected score >= 55 for single critical secrets finding, got %d", ts.Score)
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

	// Critical findings across multiple categories to exceed 100 penalty after caps.
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "secrets",
			Findings: []interfaces.Finding{
				{Category: interfaces.CategorySecrets, Severity: interfaces.SeverityCritical, Confidence: 1.0},
			},
		},
		{
			AnalyzerName: "security",
			Findings: []interfaces.Finding{
				{Category: interfaces.CategorySecurity, Severity: interfaces.SeverityCritical, Confidence: 1.0},
			},
		},
		{
			AnalyzerName: "logic",
			Findings: []interfaces.Finding{
				{Category: interfaces.CategoryLogic, Severity: interfaces.SeverityCritical, Confidence: 1.0},
			},
		},
	}

	ts := calc.Score(results)

	// secrets: 25 * 2.0 = 50, capped at 40
	// security: 25 * 1.5 = 37.5, cap 40 (uncapped)
	// logic: 25 * 1.3 = 32.5, capped at 25
	// total = 40 + 37.5 + 25 = 102.5 → score = 100 - 103 = -3 → clamped to 0
	// Has critical → no floor.
	if ts.Score < 0 {
		t.Errorf("score must never be negative, got %d", ts.Score)
	}
	if ts.Score != 0 {
		t.Errorf("expected score 0 for critical findings across categories, got %d", ts.Score)
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
	// penalty = 50 * 1.5 * 1.0 = 75, capped at 40 (security with critical) → score = 60
	// Has critical → no floor.
	if ts.Score != 60 {
		t.Errorf("expected score 60 (custom weight capped by category cap), got %d", ts.Score)
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

func TestCalculator_ManyCoverageFindings_NoCriticalNoHigh_CappedScore(t *testing.T) {
	calc := NewCalculator()

	// 20 medium coverage findings — the exact scenario that was scoring 0 before the fix.
	findings := make([]interfaces.Finding, 20)
	for i := range findings {
		findings[i] = interfaces.Finding{
			Category:   interfaces.CategoryCoverage,
			Severity:   interfaces.SeverityMedium,
			Confidence: 1.0,
		}
	}

	results := []*interfaces.AnalysisResult{
		{AnalyzerName: "coverage", Findings: findings},
	}

	ts := calc.Score(results)

	// Before fix: 20 * 8 * 0.7 = 112 → score 0 RED
	// After fix: 20 * 8 * 0.4 = 64, capped at 25 → score 75 GREEN
	// No critical/high → floor at 30. Score 75 > 30 so floor doesn't apply.
	if ts.Score < 30 {
		t.Errorf("20 coverage findings with no critical/high must score >= 30, got %d", ts.Score)
	}
	if ts.Score < 70 {
		t.Errorf("expected score >= 70 for capped coverage-only findings, got %d", ts.Score)
	}
	if ts.Rating == interfaces.RatingRed {
		t.Errorf("coverage-only findings with no critical/high must not be RED, got %s (score: %d)", ts.Rating, ts.Score)
	}
}

func TestCalculator_ManyMediumAcrossCategories_FloorAt30(t *testing.T) {
	calc := NewCalculator()

	// Many medium findings across 4 categories, all hitting caps.
	// No critical or high findings → floor at 30.
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "coverage",
			Findings:     repeatFinding(interfaces.CategoryCoverage, interfaces.SeverityMedium, 20),
		},
		{
			AnalyzerName: "complexity",
			Findings:     repeatFinding(interfaces.CategoryComplexity, interfaces.SeverityMedium, 10),
		},
		{
			AnalyzerName: "patterns",
			Findings:     repeatFinding(interfaces.CategoryPattern, interfaces.SeverityMedium, 10),
		},
		{
			AnalyzerName: "conventions",
			Findings:     repeatFinding(interfaces.CategoryConvention, interfaces.SeverityMedium, 10),
		},
	}

	ts := calc.Score(results)

	// coverage: 20 * 8 * 0.4 = 64, capped at 25
	// complexity: 10 * 8 * 0.8 = 64, capped at 25
	// pattern: 10 * 8 * 0.5 = 40, capped at 25
	// convention: 10 * 8 * 0.3 = 24, not capped (< 25)
	// total = 25 + 25 + 25 + 24 = 99 → raw score = 1
	// No critical/high → floor at 30.
	if ts.Score != MinScoreNoCriticalNoHigh {
		t.Errorf("expected floor score %d for many medium findings with no critical/high, got %d",
			MinScoreNoCriticalNoHigh, ts.Score)
	}
	if ts.Rating != interfaces.RatingRed {
		t.Errorf("expected RED (score 30 < yellow threshold 50), got %s", ts.Rating)
	}
}

func TestCalculator_HighWithMedium_FloorAt15(t *testing.T) {
	calc := NewCalculator()

	// High security findings + medium findings across categories.
	// Has high but no critical → floor at 15.
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "security",
			Findings: []interfaces.Finding{
				{Category: interfaces.CategorySecurity, Severity: interfaces.SeverityHigh, Confidence: 1.0},
				{Category: interfaces.CategorySecurity, Severity: interfaces.SeverityHigh, Confidence: 1.0},
				{Category: interfaces.CategorySecurity, Severity: interfaces.SeverityHigh, Confidence: 1.0},
			},
		},
		{
			AnalyzerName: "complexity",
			Findings:     repeatFinding(interfaces.CategoryComplexity, interfaces.SeverityMedium, 10),
		},
		{
			AnalyzerName: "coverage",
			Findings:     repeatFinding(interfaces.CategoryCoverage, interfaces.SeverityMedium, 10),
		},
	}

	ts := calc.Score(results)

	// security: 3 * 15 * 1.5 = 67.5, capped at 40 (security with high)
	// complexity: 10 * 8 * 0.8 = 64, capped at 25
	// coverage: 10 * 8 * 0.4 = 32, capped at 25
	// total = 40 + 25 + 25 = 90 → raw score = 10
	// Has high, no critical → floor at 15.
	if ts.Score != MinScoreNoCritical {
		t.Errorf("expected floor score %d for high+medium findings with no critical, got %d",
			MinScoreNoCritical, ts.Score)
	}
	if ts.Rating != interfaces.RatingRed {
		t.Errorf("expected RED for score %d, got %s", ts.Score, ts.Rating)
	}
}

func TestCalculator_CategoryCap_LimitsSingleCategoryPenalty(t *testing.T) {
	calc := NewCalculator()

	// 50 medium complexity findings — penalty should be capped, not 50x.
	findings := repeatFinding(interfaces.CategoryComplexity, interfaces.SeverityMedium, 50)

	results := []*interfaces.AnalysisResult{
		{AnalyzerName: "complexity", Findings: findings},
	}

	ts := calc.Score(results)

	// Raw: 50 * 8 * 0.8 = 320. Capped at 25. Score = 75.
	// No critical/high → floor at 30. 75 > 30, floor doesn't apply.
	if ts.Score != 75 {
		t.Errorf("expected score 75 (50 findings capped to 25 penalty), got %d", ts.Score)
	}
	if ts.Breakdown[interfaces.CategoryComplexity] != CategoryPenaltyCap {
		t.Errorf("expected breakdown capped at %d, got %d",
			CategoryPenaltyCap, ts.Breakdown[interfaces.CategoryComplexity])
	}
}

func TestCalculator_CriticalFindings_NoFloor(t *testing.T) {
	calc := NewCalculator()

	// Critical findings can drive the score to 0 — no floor protection.
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "secrets",
			Findings: []interfaces.Finding{
				{Category: interfaces.CategorySecrets, Severity: interfaces.SeverityCritical, Confidence: 1.0},
			},
		},
		{
			AnalyzerName: "security",
			Findings: []interfaces.Finding{
				{Category: interfaces.CategorySecurity, Severity: interfaces.SeverityCritical, Confidence: 1.0},
			},
		},
		{
			AnalyzerName: "logic",
			Findings: []interfaces.Finding{
				{Category: interfaces.CategoryLogic, Severity: interfaces.SeverityCritical, Confidence: 1.0},
			},
		},
	}

	ts := calc.Score(results)

	// Has critical → no floor. Score can reach 0.
	if ts.Score != 0 {
		t.Errorf("critical findings should be able to drive score to 0, got %d", ts.Score)
	}
}

// repeatFinding creates n identical findings for testing.
func repeatFinding(cat interfaces.Category, sev interfaces.Severity, n int) []interfaces.Finding {
	findings := make([]interfaces.Finding, n)
	for i := range findings {
		findings[i] = interfaces.Finding{
			Category:   cat,
			Severity:   sev,
			Confidence: 1.0,
		}
	}
	return findings
}

// sentinel error for tests
var errTestAnalyzer = errorString("test: analyzer failed")

type errorString string

func (e errorString) Error() string { return string(e) }
