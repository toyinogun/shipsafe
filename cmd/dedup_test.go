package cmd

import (
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

func TestDeduplicateCrossAnalyzer_RemovesAIFindingDuplicatedByStatic(t *testing.T) {
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "secrets",
			Findings: []interfaces.Finding{
				{File: "config.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "hardcoded AWS access key detected in source code"},
			},
		},
		{
			AnalyzerName: "ai-reviewer",
			Findings: []interfaces.Finding{
				// Same issue, slightly different line and wording.
				{File: "config.go", StartLine: 12, Severity: interfaces.SeverityMedium, Description: "hardcoded AWS access key detected — should use environment variable"},
				// Unique AI finding, not in static.
				{File: "handler.go", StartLine: 55, Severity: interfaces.SeverityMedium, Description: "nil pointer dereference on user object after lookup"},
			},
		},
	}

	deduped := deduplicateCrossAnalyzer(results)

	// Static result should be unchanged.
	if len(deduped[0].Findings) != 1 {
		t.Errorf("expected static findings unchanged (1), got %d", len(deduped[0].Findings))
	}

	// AI result should have the duplicate removed, keeping only the unique finding.
	if len(deduped[1].Findings) != 1 {
		t.Fatalf("expected 1 AI finding after cross-analyzer dedup, got %d", len(deduped[1].Findings))
	}
	if deduped[1].Findings[0].File != "handler.go" {
		t.Errorf("expected kept AI finding to be handler.go, got %q", deduped[1].Findings[0].File)
	}
}

func TestDeduplicateCrossAnalyzer_KeepsStaticFinding(t *testing.T) {
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "complexity",
			Findings: []interfaces.Finding{
				{File: "main.go", StartLine: 20, Severity: interfaces.SeverityMedium, Description: "function processData has cyclomatic complexity 18"},
			},
		},
		{
			AnalyzerName: "ai-reviewer",
			Findings: []interfaces.Finding{
				{File: "main.go", StartLine: 22, Severity: interfaces.SeverityHigh, Description: "function processData has cyclomatic complexity too high for maintainability"},
			},
		},
	}

	deduped := deduplicateCrossAnalyzer(results)

	// Static finding must be preserved.
	if len(deduped[0].Findings) != 1 {
		t.Fatalf("expected static finding preserved, got %d", len(deduped[0].Findings))
	}
	if deduped[0].Findings[0].Severity != interfaces.SeverityMedium {
		t.Errorf("expected static severity medium preserved, got %q", deduped[0].Findings[0].Severity)
	}

	// AI finding should be removed.
	if len(deduped[1].Findings) != 0 {
		t.Errorf("expected AI duplicate removed, got %d findings", len(deduped[1].Findings))
	}
}

func TestDeduplicateCrossAnalyzer_NoOverlap(t *testing.T) {
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "secrets",
			Findings: []interfaces.Finding{
				{File: "config.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "hardcoded AWS access key detected"},
			},
		},
		{
			AnalyzerName: "ai-reviewer",
			Findings: []interfaces.Finding{
				{File: "handler.go", StartLine: 55, Severity: interfaces.SeverityMedium, Description: "nil pointer dereference on user object"},
			},
		},
	}

	deduped := deduplicateCrossAnalyzer(results)

	if len(deduped[0].Findings) != 1 {
		t.Errorf("expected 1 static finding, got %d", len(deduped[0].Findings))
	}
	if len(deduped[1].Findings) != 1 {
		t.Errorf("expected 1 AI finding (no overlap), got %d", len(deduped[1].Findings))
	}
}

func TestDeduplicateCrossAnalyzer_NoAIResults(t *testing.T) {
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "secrets",
			Findings: []interfaces.Finding{
				{File: "config.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "hardcoded secret"},
			},
		},
		{
			AnalyzerName: "complexity",
			Findings: []interfaces.Finding{
				{File: "main.go", StartLine: 20, Severity: interfaces.SeverityMedium, Description: "high complexity"},
			},
		},
	}

	deduped := deduplicateCrossAnalyzer(results)

	// Nothing should change when there are no AI results.
	if len(deduped) != 2 {
		t.Fatalf("expected 2 results, got %d", len(deduped))
	}
	if len(deduped[0].Findings) != 1 || len(deduped[1].Findings) != 1 {
		t.Error("findings should be unchanged when no AI results present")
	}
}

func TestDeduplicateCrossAnalyzer_SingleResult(t *testing.T) {
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "ai-reviewer",
			Findings: []interfaces.Finding{
				{File: "main.go", StartLine: 10, Severity: interfaces.SeverityMedium, Description: "potential issue"},
			},
		},
	}

	deduped := deduplicateCrossAnalyzer(results)
	if len(deduped[0].Findings) != 1 {
		t.Errorf("single result should be unchanged, got %d findings", len(deduped[0].Findings))
	}
}

func TestDeduplicateCrossAnalyzer_MultipleStaticMatchSameAI(t *testing.T) {
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "secrets",
			Findings: []interfaces.Finding{
				{File: "config.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "hardcoded AWS access key detected in source"},
			},
		},
		{
			AnalyzerName: "patterns",
			Findings: []interfaces.Finding{
				{File: "config.go", StartLine: 11, Severity: interfaces.SeverityMedium, Description: "hardcoded AWS access key should use environment variable"},
			},
		},
		{
			AnalyzerName: "ai-reviewer",
			Findings: []interfaces.Finding{
				{File: "config.go", StartLine: 12, Severity: interfaces.SeverityMedium, Description: "hardcoded AWS access key detected — credential leak risk"},
			},
		},
	}

	deduped := deduplicateCrossAnalyzer(results)

	// Static results should be unchanged.
	if len(deduped[0].Findings) != 1 {
		t.Errorf("secrets findings should be unchanged, got %d", len(deduped[0].Findings))
	}
	if len(deduped[1].Findings) != 1 {
		t.Errorf("patterns findings should be unchanged, got %d", len(deduped[1].Findings))
	}

	// AI finding should be removed (matches either static finding).
	if len(deduped[2].Findings) != 0 {
		t.Errorf("expected AI duplicate removed, got %d findings", len(deduped[2].Findings))
	}
}

func TestDeduplicateCrossAnalyzer_LinesWithinThreshold(t *testing.T) {
	// AI finding 5 lines away from static — should be deduped.
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "secrets",
			Findings: []interfaces.Finding{
				{File: "config.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "hardcoded database password in source"},
			},
		},
		{
			AnalyzerName: "ai-reviewer",
			Findings: []interfaces.Finding{
				{File: "config.go", StartLine: 15, Severity: interfaces.SeverityMedium, Description: "hardcoded database password should use environment variable"},
			},
		},
	}

	deduped := deduplicateCrossAnalyzer(results)

	if len(deduped[1].Findings) != 0 {
		t.Errorf("expected AI finding 5 lines away to be deduped, got %d findings", len(deduped[1].Findings))
	}
}

func TestDeduplicateCrossAnalyzer_LinesBeyondThreshold(t *testing.T) {
	// AI finding 8 lines away from static — should NOT be deduped.
	results := []*interfaces.AnalysisResult{
		{
			AnalyzerName: "secrets",
			Findings: []interfaces.Finding{
				{File: "config.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "hardcoded database password in source"},
			},
		},
		{
			AnalyzerName: "ai-reviewer",
			Findings: []interfaces.Finding{
				{File: "config.go", StartLine: 18, Severity: interfaces.SeverityMedium, Description: "hardcoded database password should use environment variable"},
			},
		},
	}

	deduped := deduplicateCrossAnalyzer(results)

	if len(deduped[1].Findings) != 1 {
		t.Errorf("expected AI finding 8 lines away to NOT be deduped, got %d findings", len(deduped[1].Findings))
	}
}
