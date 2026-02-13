package analyzer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// testFileMapping defines how source files map to test files per language.
type testFileMapping struct {
	// sourceExts are extensions that identify source files for this language.
	sourceExts []string
	// isTestFile returns true if the given path is a test file for this language.
	isTestFile func(path string) bool
	// testPatterns returns possible test file paths for a given source file path.
	testPatterns func(path string) []string
}

// Language-aware test file mappings.
var testFileMappings = []testFileMapping{
	{
		// Go: foo.go -> foo_test.go
		sourceExts: []string{".go"},
		isTestFile: func(path string) bool {
			return strings.HasSuffix(path, "_test.go")
		},
		testPatterns: func(path string) []string {
			base := strings.TrimSuffix(path, ".go")
			return []string{base + "_test.go"}
		},
	},
	{
		// Python: foo.py -> test_foo.py, foo_test.py, tests/test_foo.py
		sourceExts: []string{".py"},
		isTestFile: func(path string) bool {
			base := filepath.Base(path)
			return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py")
		},
		testPatterns: func(path string) []string {
			dir := filepath.Dir(path)
			base := filepath.Base(path)
			name := strings.TrimSuffix(base, ".py")
			return []string{
				filepath.Join(dir, "test_"+base),
				filepath.Join(dir, name+"_test.py"),
				filepath.Join(dir, "tests", "test_"+base),
				filepath.Join("tests", "test_"+base),
			}
		},
	},
	{
		// JavaScript: foo.js -> foo.test.js, foo.spec.js
		sourceExts: []string{".js", ".jsx"},
		isTestFile: func(path string) bool {
			return strings.HasSuffix(path, ".test.js") ||
				strings.HasSuffix(path, ".spec.js") ||
				strings.HasSuffix(path, ".test.jsx") ||
				strings.HasSuffix(path, ".spec.jsx")
		},
		testPatterns: func(path string) []string {
			ext := filepath.Ext(path)
			base := strings.TrimSuffix(path, ext)
			return []string{
				base + ".test" + ext,
				base + ".spec" + ext,
			}
		},
	},
	{
		// TypeScript: foo.ts -> foo.test.ts, foo.spec.ts
		sourceExts: []string{".ts", ".tsx"},
		isTestFile: func(path string) bool {
			return strings.HasSuffix(path, ".test.ts") ||
				strings.HasSuffix(path, ".spec.ts") ||
				strings.HasSuffix(path, ".test.tsx") ||
				strings.HasSuffix(path, ".spec.tsx")
		},
		testPatterns: func(path string) []string {
			ext := filepath.Ext(path)
			base := strings.TrimSuffix(path, ext)
			return []string{
				base + ".test" + ext,
				base + ".spec" + ext,
			}
		},
	},
	{
		// Ruby: foo.rb -> foo_test.rb, test_foo.rb, spec/foo_spec.rb
		sourceExts: []string{".rb"},
		isTestFile: func(path string) bool {
			base := filepath.Base(path)
			return strings.HasSuffix(base, "_test.rb") ||
				strings.HasSuffix(base, "_spec.rb") ||
				strings.HasPrefix(base, "test_")
		},
		testPatterns: func(path string) []string {
			dir := filepath.Dir(path)
			base := filepath.Base(path)
			name := strings.TrimSuffix(base, ".rb")
			return []string{
				filepath.Join(dir, name+"_test.rb"),
				filepath.Join(dir, name+"_spec.rb"),
				filepath.Join("spec", name+"_spec.rb"),
			}
		},
	},
	{
		// Rust: foo.rs -> tests within the same file (mod tests), or tests/foo.rs
		sourceExts: []string{".rs"},
		isTestFile: func(path string) bool {
			return strings.Contains(path, "/tests/") || strings.HasPrefix(path, "tests/")
		},
		testPatterns: func(path string) []string {
			base := filepath.Base(path)
			return []string{
				filepath.Join("tests", base),
			}
		},
	},
}

// CoverageAnalyzer checks whether new or modified source files have
// corresponding test files in the diff.
type CoverageAnalyzer struct{}

// NewCoverageAnalyzer creates a new test coverage heuristic analyzer.
func NewCoverageAnalyzer() *CoverageAnalyzer {
	return &CoverageAnalyzer{}
}

// Name returns the analyzer identifier.
func (c *CoverageAnalyzer) Name() string {
	return "coverage"
}

// Analyze checks that source files in the diff have corresponding test changes.
func (c *CoverageAnalyzer) Analyze(ctx context.Context, diff *interfaces.Diff) (*interfaces.AnalysisResult, error) {
	result := &interfaces.AnalysisResult{
		AnalyzerName: c.Name(),
	}

	// Build a set of all file paths in the diff for fast lookup.
	diffPaths := make(map[string]bool, len(diff.Files))
	for _, file := range diff.Files {
		diffPaths[file.Path] = true
	}

	for i := range diff.Files {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		file := &diff.Files[i]
		if file.IsBinary || file.Status == interfaces.FileDeleted {
			continue
		}

		// Skip if the file itself is a test file.
		if isTestFileForCoverage(file.Path) {
			continue
		}

		// Identify the language mapping for this file.
		mapping := findMapping(file.Path)
		if mapping == nil {
			continue
		}

		// Check if any expected test file is in the diff.
		testPatterns := mapping.testPatterns(file.Path)
		hasTest := false
		for _, pattern := range testPatterns {
			if diffPaths[pattern] {
				hasTest = true
				break
			}
		}

		if !hasTest {
			severity := interfaces.SeverityLow
			title := "Modified file has no test changes"
			if file.Status == interfaces.FileAdded {
				severity = interfaces.SeverityMedium
				title = "New file has no corresponding test file"
			}

			result.Findings = append(result.Findings, interfaces.Finding{
				ID:        fmt.Sprintf("COV-%s-%s", strings.ToUpper(string(file.Status)), sanitizePath(file.Path)),
				Category:  interfaces.CategoryCoverage,
				Severity:  severity,
				File:      file.Path,
				StartLine: 0,
				EndLine:   0,
				Title:     title,
				Description: fmt.Sprintf(
					"File %s was %s but no corresponding test file was found in the diff. Expected one of: %s",
					file.Path, file.Status, strings.Join(testPatterns, ", "),
				),
				Suggestion: "Add tests for the new or modified code to maintain test coverage.",
				Source:     "coverage",
				Confidence: 0.70,
			})
		}
	}

	return result, nil
}

// findMapping returns the test file mapping for a source file, or nil if no mapping matches.
func findMapping(path string) *testFileMapping {
	for i := range testFileMappings {
		mapping := &testFileMappings[i]
		// Skip if this file is already a test file.
		if mapping.isTestFile(path) {
			return nil
		}
		for _, ext := range mapping.sourceExts {
			if strings.HasSuffix(path, ext) {
				return mapping
			}
		}
	}
	return nil
}

// isTestFileForCoverage checks if a file is any kind of test file.
func isTestFileForCoverage(path string) bool {
	for _, mapping := range testFileMappings {
		if mapping.isTestFile(path) {
			return true
		}
	}
	return false
}

// sanitizePath converts a file path to a short ID-safe string.
func sanitizePath(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	r := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ".", "-")
	return strings.ToUpper(r.Replace(base))
}
