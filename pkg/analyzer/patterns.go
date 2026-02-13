package analyzer

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// Anti-pattern detection rules compiled once at init.
var (
	// SQL string concatenation: "SELECT " + var, "INSERT " + var, fmt.Sprintf("SELECT ...
	sqlConcatPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)["'](?:SELECT|INSERT|UPDATE|DELETE|DROP|ALTER|CREATE)\s.*["']\s*\+`),
		regexp.MustCompile(`(?i)\+\s*["'](?:\s*(?:SELECT|INSERT|UPDATE|DELETE|DROP|ALTER|CREATE|WHERE|FROM|SET|INTO|VALUES))\b`),
		regexp.MustCompile(`(?i)fmt\.Sprintf\(\s*["'](?:SELECT|INSERT|UPDATE|DELETE|DROP|ALTER|CREATE)\s`),
		regexp.MustCompile(`(?i)f["'](?:SELECT|INSERT|UPDATE|DELETE|DROP|ALTER|CREATE)\s`),
		regexp.MustCompile(`(?i)(?:SELECT|INSERT|UPDATE|DELETE)\s.*%s`),
	}

	// Empty catch/except blocks.
	emptyCatchPatterns = []*regexp.Regexp{
		regexp.MustCompile(`catch\s*\([^)]*\)\s*\{\s*\}`),
		regexp.MustCompile(`except\s*(?:\([^)]*\))?\s*:\s*$`),
		regexp.MustCompile(`except\s+\w+\s*:\s*$`),
		regexp.MustCompile(`catch\s*\{[\s]*\}`),
	}

	// Debug/console output statements.
	debugPrintPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\bconsole\.(log|debug|info|warn|error)\s*\(`),
		regexp.MustCompile(`\bfmt\.Print(ln|f)?\s*\(`),
		regexp.MustCompile(`\bprint\s*\(`),
		regexp.MustCompile(`\bprintln\s*\(`),
		regexp.MustCompile(`\bSystem\.out\.print(ln)?\s*\(`),
		regexp.MustCompile(`\bputs\s+`),
		regexp.MustCompile(`\bpp\s+`),
	}

	// TODO/FIXME/HACK comments.
	todoPattern = regexp.MustCompile(`(?i)\b(TODO|FIXME|HACK|XXX)\b`)
)

// File extensions considered test files (debug prints are acceptable in tests).
var testFileIndicators = []string{
	"_test.go",
	".test.js", ".test.ts", ".test.jsx", ".test.tsx",
	".spec.js", ".spec.ts", ".spec.jsx", ".spec.tsx",
	"test_", "__test__",
	"_test.py", "_test.rb",
}

// File extensions that should skip empty catch/except block detection.
// YAML files don't have try/catch blocks; the regex matches indentation patterns.
var emptyCatchSkipExts = []string{".yaml", ".yml"}

// PatternsAnalyzer detects common anti-patterns in code diffs.
type PatternsAnalyzer struct{}

// NewPatternsAnalyzer creates a new anti-pattern detector.
func NewPatternsAnalyzer() *PatternsAnalyzer {
	return &PatternsAnalyzer{}
}

// Name returns the analyzer identifier.
func (p *PatternsAnalyzer) Name() string {
	return "patterns"
}

// Analyze scans added lines for anti-patterns.
func (p *PatternsAnalyzer) Analyze(ctx context.Context, diff *interfaces.Diff) (*interfaces.AnalysisResult, error) {
	result := &interfaces.AnalysisResult{
		AnalyzerName: p.Name(),
	}

	for i := range diff.Files {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		file := &diff.Files[i]
		if file.IsBinary || file.Status == interfaces.FileDeleted {
			continue
		}

		if isFixturePath(file.Path) {
			continue
		}

		isTest := isTestFile(file.Path)

		for j := range file.Hunks {
			hunk := &file.Hunks[j]
			for _, line := range hunk.AddedLines {
				if ctx.Err() != nil {
					return result, ctx.Err()
				}

				findings := p.scanLine(file.Path, line, isTest)
				result.Findings = append(result.Findings, findings...)
			}
		}
	}

	return result, nil
}

// scanLine checks a single added line for anti-patterns.
func (p *PatternsAnalyzer) scanLine(path string, line interfaces.Line, isTest bool) []interfaces.Finding {
	var findings []interfaces.Finding
	content := line.Content

	// Check SQL string concatenation (skip test files — tests intentionally contain bad patterns).
	if !isTest {
		for _, re := range sqlConcatPatterns {
			if re.MatchString(content) {
				findings = append(findings, interfaces.Finding{
					ID:        fmt.Sprintf("PAT-SQL-CONCAT-%d", line.Number),
					Category:  interfaces.CategoryPattern,
					Severity:  interfaces.SeverityMedium,
					File:      path,
					StartLine: line.Number,
					EndLine:   line.Number,
					Title:     "SQL string concatenation detected",
					Description: fmt.Sprintf(
						"Line %d builds a SQL query via string concatenation, which is vulnerable to SQL injection.",
						line.Number,
					),
					Suggestion: "Use parameterized queries or a query builder instead of string concatenation.",
					Source:     "patterns",
					Confidence: 0.80,
				})
				break // One finding per line for this category.
			}
		}
	}

	// Check empty catch/except blocks (skip test files and YAML files).
	if !isTest && !isEmptyCatchSkipFile(path) {
		for _, re := range emptyCatchPatterns {
			if re.MatchString(content) {
				findings = append(findings, interfaces.Finding{
					ID:        fmt.Sprintf("PAT-EMPTY-CATCH-%d", line.Number),
					Category:  interfaces.CategoryPattern,
					Severity:  interfaces.SeverityMedium,
					File:      path,
					StartLine: line.Number,
					EndLine:   line.Number,
					Title:     "Empty catch/except block",
					Description: fmt.Sprintf(
						"Line %d has an empty error handler. Swallowing errors silently hides bugs.",
						line.Number,
					),
					Suggestion: "Log the error or handle it explicitly. If intentionally ignoring, add a comment explaining why.",
					Source:     "patterns",
					Confidence: 0.85,
				})
				break
			}
		}
	}

	// Check debug prints (skip test files).
	if !isTest {
		for _, re := range debugPrintPatterns {
			if re.MatchString(content) {
				// Skip if it's in a comment.
				trimmed := strings.TrimSpace(content)
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "*") {
					break
				}
				findings = append(findings, interfaces.Finding{
					ID:        fmt.Sprintf("PAT-DEBUG-PRINT-%d", line.Number),
					Category:  interfaces.CategoryPattern,
					Severity:  interfaces.SeverityLow,
					File:      path,
					StartLine: line.Number,
					EndLine:   line.Number,
					Title:     "Debug print statement",
					Description: fmt.Sprintf(
						"Line %d contains a debug print statement that should not be in production code.",
						line.Number,
					),
					Suggestion: "Remove the debug statement or replace with structured logging (e.g., slog).",
					Source:     "patterns",
					Confidence: 0.75,
				})
				break
			}
		}
	}

	// Check TODO/FIXME/HACK comments (skip test files — test TODOs are lower priority).
	if !isTest && todoPattern.MatchString(content) {
		matches := todoPattern.FindStringSubmatch(content)
		tag := strings.ToUpper(matches[1])
		findings = append(findings, interfaces.Finding{
			ID:        fmt.Sprintf("PAT-TODO-%d", line.Number),
			Category:  interfaces.CategoryPattern,
			Severity:  interfaces.SeverityInfo,
			File:      path,
			StartLine: line.Number,
			EndLine:   line.Number,
			Title:     fmt.Sprintf("%s comment in new code", tag),
			Description: fmt.Sprintf(
				"Line %d contains a %s comment. Consider resolving it before merging.",
				line.Number, tag,
			),
			Suggestion: "Resolve the TODO/FIXME/HACK or create a tracked issue for follow-up.",
			Source:     "patterns",
			Confidence: 0.95,
		})
	}

	return findings
}

// isTestFile checks if a file path indicates a test file.
func isTestFile(path string) bool {
	lower := strings.ToLower(path)
	for _, indicator := range testFileIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

// isEmptyCatchSkipFile reports whether a file should skip empty catch/except detection.
func isEmptyCatchSkipFile(path string) bool {
	lower := strings.ToLower(path)
	for _, ext := range emptyCatchSkipExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// isFixturePath checks if a file path is under tests/fixtures/ or is a .diff file.
func isFixturePath(path string) bool {
	lower := strings.ToLower(path)
	if strings.Contains(lower, "tests/fixtures/") {
		return true
	}
	if strings.HasSuffix(lower, ".diff") {
		return true
	}
	return false
}
