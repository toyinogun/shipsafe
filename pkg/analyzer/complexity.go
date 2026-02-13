package analyzer

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// Default complexity thresholds.
const (
	defaultComplexityThreshold = 15
	highComplexityThreshold    = 20
	testFileThresholdBoost     = 10
)

// Decision-point patterns matched against added lines.
// Each pattern counts as one increment to cyclomatic complexity.
var decisionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b(if|else\s+if|elif)\b`),
	regexp.MustCompile(`\bfor\b`),
	regexp.MustCompile(`\bwhile\b`),
	regexp.MustCompile(`\bcase\b`),
	regexp.MustCompile(`\bcatch\b`),
	regexp.MustCompile(`\bexcept\b`),
}

// Logical operators that add branching complexity.
var logicalOpPattern = regexp.MustCompile(`(&&|\|\|)`)

// Ternary operator.
var ternaryPattern = regexp.MustCompile(`\?[^?].*:`)

// Function definition patterns across languages.
var funcDefPatterns = []*regexp.Regexp{
	// Go: func name(...) or func (receiver) name(...)
	regexp.MustCompile(`^\s*func\s+(?:\([^)]*\)\s*)?\w+\s*\(`),
	// JavaScript/TypeScript: function name(, const name = (...) =>, async function
	regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+\w+\s*\(`),
	regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+\w+\s*=\s*(?:async\s+)?(?:\([^)]*\)|[a-zA-Z_]\w*)\s*=>`),
	// Python: def name(
	regexp.MustCompile(`^\s*(?:async\s+)?def\s+\w+\s*\(`),
	// Java/C#: access modifier + return type + name(
	regexp.MustCompile(`^\s*(?:(?:public|private|protected|static|final|abstract)\s+)+\w+\s+\w+\s*\(`),
	// Ruby: def name
	regexp.MustCompile(`^\s*def\s+\w+`),
	// Rust: fn name(
	regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+\w+`),
}

// Compiled regex for function name extraction (compiled once).
var (
	goFuncNameRe   = regexp.MustCompile(`func\s+(?:\([^)]*\)\s*)?(\w+)\s*\(`)
	jsFuncNameRe   = regexp.MustCompile(`function\s+(\w+)\s*\(`)
	arrowFuncRe    = regexp.MustCompile(`(?:const|let|var)\s+(\w+)\s*=`)
	pyFuncNameRe   = regexp.MustCompile(`def\s+(\w+)\s*\(`)
	rbFuncNameRe   = regexp.MustCompile(`def\s+(\w+)`)
	rsFuncNameRe   = regexp.MustCompile(`fn\s+(\w+)`)
	javaFuncNameRe = regexp.MustCompile(`\s(\w+)\s*\(`)
)

// ComplexityAnalyzer measures cyclomatic complexity of functions in diffs.
type ComplexityAnalyzer struct {
	threshold int
}

// ComplexityOption configures the complexity analyzer.
type ComplexityOption func(*ComplexityAnalyzer)

// WithComplexityThreshold sets the maximum allowed complexity.
func WithComplexityThreshold(t int) ComplexityOption {
	return func(a *ComplexityAnalyzer) {
		if t > 0 {
			a.threshold = t
		}
	}
}

// NewComplexityAnalyzer creates a complexity analyzer with optional configuration.
func NewComplexityAnalyzer(opts ...ComplexityOption) *ComplexityAnalyzer {
	a := &ComplexityAnalyzer{
		threshold: defaultComplexityThreshold,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Name returns the analyzer identifier.
func (c *ComplexityAnalyzer) Name() string {
	return "complexity"
}

// Analyze calculates per-function complexity from added lines in the diff.
func (c *ComplexityAnalyzer) Analyze(ctx context.Context, diff *interfaces.Diff) (*interfaces.AnalysisResult, error) {
	result := &interfaces.AnalysisResult{
		AnalyzerName: c.Name(),
	}

	for i := range diff.Files {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		file := &diff.Files[i]
		if file.IsBinary || file.Status == interfaces.FileDeleted {
			continue
		}

		findings := c.analyzeFile(file)
		result.Findings = append(result.Findings, findings...)
	}

	return result, nil
}

// addedLine pairs a line number with its content for complexity analysis.
type addedLine struct {
	number  int
	content string
}

// funcRegion tracks a function's added lines for complexity counting.
type funcRegion struct {
	name      string
	startLine int
	endLine   int
	lines     []string
}

// analyzeFile extracts function regions from added lines and scores them.
func (c *ComplexityAnalyzer) analyzeFile(file *interfaces.FileDiff) []interfaces.Finding {
	var lines []addedLine
	for j := range file.Hunks {
		for _, line := range file.Hunks[j].AddedLines {
			lines = append(lines, addedLine{number: line.Number, content: line.Content})
		}
	}

	if len(lines) == 0 {
		return nil
	}

	// Test files get a higher threshold — complex fixtures are intentional.
	threshold := c.threshold
	highThreshold := highComplexityThreshold
	if isTestFile(file.Path) {
		threshold += testFileThresholdBoost
		highThreshold += testFileThresholdBoost
	}

	regions := extractFuncRegions(lines)

	var findings []interfaces.Finding
	for _, region := range regions {
		complexity := countComplexity(region.lines)
		if complexity > threshold {
			severity := interfaces.SeverityMedium
			if complexity > highThreshold {
				severity = interfaces.SeverityHigh
			}

			findings = append(findings, interfaces.Finding{
				ID:        fmt.Sprintf("CX-%s-%d", sanitizeID(region.name), region.startLine),
				Category:  interfaces.CategoryComplexity,
				Severity:  severity,
				File:      file.Path,
				StartLine: region.startLine,
				EndLine:   region.endLine,
				Title:     fmt.Sprintf("High cyclomatic complexity in %s (%d)", region.name, complexity),
				Description: fmt.Sprintf(
					"Function %s has a cyclomatic complexity of %d (threshold: %d). Complex functions are harder to test and maintain.",
					region.name, complexity, threshold,
				),
				Suggestion: "Break the function into smaller, focused functions with single responsibilities.",
				Source:     "complexity",
				Confidence: 0.80,
				Metadata: map[string]any{
					"complexity": complexity,
					"threshold":  threshold,
				},
			})
		}
	}

	return findings
}

// extractFuncRegions identifies function definitions in added lines and groups
// subsequent lines until the next function definition.
func extractFuncRegions(lines []addedLine) []funcRegion {
	var regions []funcRegion
	var current *funcRegion

	for _, line := range lines {
		if name, ok := extractFuncName(line.content); ok {
			if current != nil {
				current.endLine = line.number - 1
				if current.endLine < current.startLine {
					current.endLine = current.startLine
				}
				regions = append(regions, *current)
			}
			current = &funcRegion{
				name:      name,
				startLine: line.number,
				lines:     []string{line.content},
			}
		} else if current != nil {
			current.lines = append(current.lines, line.content)
			current.endLine = line.number
		}
	}

	if current != nil {
		if current.endLine == 0 {
			current.endLine = current.startLine
		}
		regions = append(regions, *current)
	}

	return regions
}

// extractFuncName tries to extract the function name from a line.
func extractFuncName(line string) (string, bool) {
	for _, re := range funcDefPatterns {
		if re.MatchString(line) {
			return parseFuncNameFromLine(line), true
		}
	}
	return "", false
}

// parseFuncNameFromLine extracts the function name identifier from the line.
func parseFuncNameFromLine(line string) string {
	if m := goFuncNameRe.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	if m := jsFuncNameRe.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	if m := arrowFuncRe.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	if m := pyFuncNameRe.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	if m := rbFuncNameRe.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	if m := rsFuncNameRe.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	if m := javaFuncNameRe.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	return "unknown"
}

// countComplexity counts decision points in the given lines.
// Base complexity is 1 (for the function itself).
func countComplexity(lines []string) int {
	complexity := 1

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip pure comments.
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		// Count decision-point keywords.
		for _, re := range decisionPatterns {
			matches := re.FindAllString(line, -1)
			complexity += len(matches)
		}

		// Count logical operators.
		matches := logicalOpPattern.FindAllString(line, -1)
		complexity += len(matches)

		// Count ternary operators.
		matches = ternaryPattern.FindAllString(line, -1)
		complexity += len(matches)
	}

	return complexity
}
