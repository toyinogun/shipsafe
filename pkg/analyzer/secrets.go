package analyzer

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// secretPattern defines a single regex-based secret detection rule.
type secretPattern struct {
	name     string
	regex    *regexp.Regexp
	severity interfaces.Severity
}

// Compiled secret detection patterns.
var secretPatterns = []secretPattern{
	{
		name:     "AWS Access Key ID",
		regex:    regexp.MustCompile(`(?:^|[^A-Za-z0-9/+=])AKIA[0-9A-Z]{16}(?:[^A-Za-z0-9/+=]|$)`),
		severity: interfaces.SeverityHigh,
	},
	{
		name:     "AWS Secret Access Key",
		regex:    regexp.MustCompile(`(?i)(?:aws_secret_access_key|aws_secret_key|secret_access_key)\s*[:=]\s*[A-Za-z0-9/+=]{40}`),
		severity: interfaces.SeverityCritical,
	},
	{
		name:     "RSA/SSH Private Key",
		regex:    regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
		severity: interfaces.SeverityCritical,
	},
	{
		name:     "Generic API Key Assignment",
		regex:    regexp.MustCompile(`(?i)(?:api_key|apikey|api-key|api_secret|apisecret)\s*[:=]\s*["']?[A-Za-z0-9_\-]{16,}["']?`),
		severity: interfaces.SeverityHigh,
	},
	{
		name:     "Bearer Token",
		regex:    regexp.MustCompile(`(?i)(?:bearer\s+)[A-Za-z0-9_\-.]{20,}`),
		severity: interfaces.SeverityHigh,
	},
	{
		name:     "Database Connection String",
		regex:    regexp.MustCompile(`(?i)(?:postgres|postgresql|mysql|mongodb|redis|amqp)://[^\s"'` + "`" + `]{10,}`),
		severity: interfaces.SeverityHigh,
	},
	{
		name:     "Password Assignment",
		regex:    regexp.MustCompile(`(?i)(?:password|passwd|pwd)\s*[:=]\s*["'][^"']{8,}["']`),
		severity: interfaces.SeverityHigh,
	},
	{
		name:     "GitHub Personal Access Token",
		regex:    regexp.MustCompile(`ghp_[A-Za-z0-9]{36}`),
		severity: interfaces.SeverityHigh,
	},
	{
		name:     "Generic Secret Assignment",
		regex:    regexp.MustCompile(`(?i)(?:secret|token|auth)_?(?:key)?\s*[:=]\s*["'][A-Za-z0-9_\-/+=]{20,}["']`),
		severity: interfaces.SeverityHigh,
	},
}

// Substrings that indicate a line is a false positive (test data, examples, placeholders).
var falsePositiveIndicators = []string{
	"example",
	"placeholder",
	"your-",
	"your_",
	"xxx",
	"changeme",
	"replace_me",
	"insert_here",
	"todo",
	"fixme",
	"dummy",
	"fake",
	"test_",
	"mock_",
	"sample",
	"<your",
	"${",
	"{{",
}

// File paths that typically contain example/test data and should be skipped.
var falsePositivePathSuffixes = []string{
	"_test.go",
	".test.ts",
	".test.js",
	".spec.ts",
	".spec.js",
	".example",
	".example.yml",
	".example.yaml",
	".example.json",
	".example.env",
	".sample",
	"testdata/",
	"fixtures/",
	"__mocks__/",
	".diff",
	"go.sum",
	".lock",
	"package-lock.json",
}

// Prefixes indicating a line is a checksum, not a secret.
var checksumPrefixes = []string{
	"h1:",
	"sha256:",
	"sha512:",
	"sha1:",
	"sha384:",
}

// Frontend file extensions where entropy thresholds should be relaxed.
var frontendFileExts = []string{".tsx", ".jsx"}

// Patterns indicating a line contains CSS/styling attributes (Tailwind, JSX).
// Lines matching these are almost never secrets.
var frontendStylingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bclass(?:Name)?\s*=`),
	regexp.MustCompile(`(?i)\bcn\s*\(`),
	regexp.MustCompile(`(?i)\b(?:src|href|alt|placeholder)\s*=`),
}

// jsxAttributeValueRe matches HTML/JSX attribute="long-string-here" patterns.
var jsxAttributeValueRe = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9-]*\s*=\s*["'][^"']{20,}["']`)

// isFrontendFile reports whether the file is a frontend file (.tsx, .jsx).
func isFrontendFile(path string) bool {
	lower := strings.ToLower(path)
	for _, ext := range frontendFileExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// isFrontendStylingLine reports whether a line contains CSS class or styling attribute patterns.
func isFrontendStylingLine(line string) bool {
	for _, re := range frontendStylingPatterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// isJSXAttributeValue reports whether the high-entropy portion of the line
// is inside an HTML/JSX attribute value (e.g., data-id="long-string-here").
func isJSXAttributeValue(line string) bool {
	return jsxAttributeValueRe.MatchString(line)
}

// isChecksumLine reports whether the line looks like a checksum entry.
func isChecksumLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, prefix := range checksumPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// SecretsAnalyzer detects hardcoded secrets, API keys, and credentials in diffs.
type SecretsAnalyzer struct {
	entropyThreshold float64
	entropyMinLength int
}

// NewSecretsAnalyzer creates a secrets analyzer with default settings.
func NewSecretsAnalyzer() *SecretsAnalyzer {
	return &SecretsAnalyzer{
		entropyThreshold: 4.5,
		entropyMinLength: 20,
	}
}

// Name returns the analyzer identifier.
func (s *SecretsAnalyzer) Name() string {
	return "secrets"
}

// Analyze scans added lines in the diff for hardcoded secrets.
func (s *SecretsAnalyzer) Analyze(ctx context.Context, diff *interfaces.Diff) (*interfaces.AnalysisResult, error) {
	result := &interfaces.AnalysisResult{
		AnalyzerName: s.Name(),
	}

	for i := range diff.Files {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		file := &diff.Files[i]

		if file.IsBinary || file.Status == interfaces.FileDeleted {
			continue
		}

		if isTestFixturePath(file.Path) {
			continue
		}

		for j := range file.Hunks {
			hunk := &file.Hunks[j]
			// Only scan added lines.
			for _, line := range hunk.AddedLines {
				if ctx.Err() != nil {
					return result, ctx.Err()
				}

				findings := s.scanLine(file.Path, line)
				result.Findings = append(result.Findings, findings...)
			}
		}
	}

	return result, nil
}

// scanLine checks a single added line against all secret patterns and entropy.
func (s *SecretsAnalyzer) scanLine(path string, line interfaces.Line) []interfaces.Finding {
	content := line.Content
	if isFalsePositive(content) {
		return nil
	}

	// Skip lines that contain CSS/styling attributes — these generate
	// false positives from Tailwind classes and JSX attribute values.
	if isFrontendStylingLine(content) {
		return nil
	}

	var findings []interfaces.Finding

	// Regex pattern matching.
	for _, p := range secretPatterns {
		if p.regex.MatchString(content) {
			findings = append(findings, interfaces.Finding{
				ID:       fmt.Sprintf("SEC-%s-%d", sanitizeID(p.name), line.Number),
				Category: interfaces.CategorySecrets,
				Severity: p.severity,
				File:     path,
				StartLine: line.Number,
				EndLine:   line.Number,
				Title:     fmt.Sprintf("Possible %s detected", p.name),
				Description: fmt.Sprintf(
					"Line %d may contain a hardcoded %s. Secrets should be stored in environment variables or a secrets manager.",
					line.Number, p.name,
				),
				Suggestion: "Remove the hardcoded secret and use an environment variable or secrets manager instead.",
				Source:     "secrets",
				Confidence: 0.85,
			})
		}
	}

	// Shannon entropy check for high-entropy strings.
	if f, ok := s.checkEntropy(path, line); ok {
		// Don't add an entropy finding if we already matched a specific pattern.
		if len(findings) == 0 {
			findings = append(findings, f)
		}
	}

	return findings
}

// checkEntropy looks for high-entropy tokens on a line that may be secrets.
func (s *SecretsAnalyzer) checkEntropy(path string, line interfaces.Line) (interfaces.Finding, bool) {
	if isChecksumLine(line.Content) {
		return interfaces.Finding{}, false
	}

	// Skip lines that are clearly CSS/styling or JSX attribute values.
	if isFrontendStylingLine(line.Content) {
		return interfaces.Finding{}, false
	}
	if isFrontendFile(path) && isJSXAttributeValue(line.Content) {
		return interfaces.Finding{}, false
	}

	// Use a higher entropy threshold for frontend files because CSS class
	// strings typically fall in the 4.5-5.5 range while real secrets are 5.5+.
	threshold := s.entropyThreshold
	if isFrontendFile(path) {
		threshold = 5.5
	}

	tokens := extractTokens(line.Content)
	for _, token := range tokens {
		if len(token) < s.entropyMinLength {
			continue
		}
		// Strings with 3+ spaces are text content (prose, descriptions,
		// testimonials), not secrets. Real API keys/tokens never contain spaces.
		if strings.Count(token, " ") >= 3 {
			continue
		}
		entropy := shannonEntropy(token)
		if entropy > threshold {
			return interfaces.Finding{
				ID:        fmt.Sprintf("SEC-ENTROPY-%d", line.Number),
				Category:  interfaces.CategorySecrets,
				Severity:  interfaces.SeverityMedium,
				File:      path,
				StartLine: line.Number,
				EndLine:   line.Number,
				Title:     "High-entropy string detected",
				Description: fmt.Sprintf(
					"Line %d contains a high-entropy string (%.2f bits/char) that may be a secret.",
					line.Number, entropy,
				),
				Suggestion: "Verify this is not a hardcoded secret. If it is, move it to environment variables.",
				Source:     "secrets",
				Confidence: 0.60,
				Metadata: map[string]any{
					"entropy": entropy,
					"length":  len(token),
				},
			}, true
		}
	}
	return interfaces.Finding{}, false
}

// shannonEntropy calculates the Shannon entropy of a string in bits per character.
func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	freq := make(map[rune]int)
	for _, c := range s {
		freq[c]++
	}

	length := float64(len([]rune(s)))
	entropy := 0.0
	for _, count := range freq {
		p := float64(count) / length
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

// extractTokens splits a line into candidate secret tokens.
// It extracts quoted strings and assignment values.
func extractTokens(line string) []string {
	var tokens []string

	// Extract quoted strings.
	quoteRe := regexp.MustCompile(`["']([^"']+)["']`)
	for _, m := range quoteRe.FindAllStringSubmatch(line, -1) {
		tokens = append(tokens, m[1])
	}

	// Extract assignment values (key=value without quotes).
	assignRe := regexp.MustCompile(`(?:=|:)\s*([A-Za-z0-9_\-/+=.]{20,})`)
	for _, m := range assignRe.FindAllStringSubmatch(line, -1) {
		tokens = append(tokens, m[1])
	}

	return tokens
}

// isFalsePositive checks if a line contains indicators of test/example data.
func isFalsePositive(line string) bool {
	lower := strings.ToLower(line)
	for _, indicator := range falsePositiveIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

// isTestFixturePath checks if a file path is a known test/fixture/example file.
func isTestFixturePath(path string) bool {
	lower := strings.ToLower(path)
	for _, suffix := range falsePositivePathSuffixes {
		if strings.HasSuffix(lower, suffix) || strings.Contains(lower, suffix) {
			return true
		}
	}
	return false
}

// sanitizeID converts a pattern name to a short uppercase ID fragment.
func sanitizeID(name string) string {
	r := strings.NewReplacer(" ", "-", "/", "-")
	return strings.ToUpper(r.Replace(name))
}
