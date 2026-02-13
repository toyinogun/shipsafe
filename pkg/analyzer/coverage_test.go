package analyzer

import (
	"context"
	"fmt"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// makeAddedLines generates n added lines for use in test diffs.
// This ensures files exceed the minLinesForCoverage threshold.
func makeAddedLines(n int) []interfaces.Line {
	lines := make([]interfaces.Line, n)
	for i := range lines {
		lines[i] = interfaces.Line{Number: i + 1, Content: fmt.Sprintf("line %d", i+1)}
	}
	return lines
}

// makeLargeHunk creates a hunk with enough added lines to exceed the coverage threshold.
func makeLargeHunk() []interfaces.Hunk {
	return []interfaces.Hunk{{AddedLines: makeAddedLines(25)}}
}

// makeSmallHunk creates a hunk with fewer lines than the coverage threshold.
func makeSmallHunk() []interfaces.Hunk {
	return []interfaces.Hunk{{AddedLines: makeAddedLines(5)}}
}

// makeLargeHunkWithLines creates a hunk with specific content lines, padded
// to exceed the minLinesForCoverage threshold.
func makeLargeHunkWithLines(contentLines ...string) []interfaces.Hunk {
	lines := make([]interfaces.Line, 0, 25)
	for i, content := range contentLines {
		lines = append(lines, interfaces.Line{Number: i + 1, Content: content})
	}
	for i := len(contentLines); i < 25; i++ {
		lines = append(lines, interfaces.Line{Number: i + 1, Content: fmt.Sprintf("line %d", i+1)})
	}
	return []interfaces.Hunk{{AddedLines: lines}}
}

func TestCoverageAnalyzer_Name(t *testing.T) {
	a := NewCoverageAnalyzer()
	if a.Name() != "coverage" {
		t.Errorf("expected name %q, got %q", "coverage", a.Name())
	}
}

func TestCoverageAnalyzer_ImplementsInterface(t *testing.T) {
	var _ Analyzer = NewCoverageAnalyzer()
}

func TestCoverageAnalyzer_GoFile_WithTest_NoFinding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/handler/handler.go", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
			{Path: "pkg/handler/handler_test.go", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings when test file exists, got %d: %v", len(result.Findings), findingIDs(result.Findings))
	}
}

func TestCoverageAnalyzer_GoFile_WithoutTest_Finding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/handler/handler.go", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding when Go file has no test")
	}
	f := result.Findings[0]
	if f.Category != interfaces.CategoryCoverage {
		t.Errorf("expected category %q, got %q", interfaces.CategoryCoverage, f.Category)
	}
}

func TestCoverageAnalyzer_NewFile_MediumSeverity(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/service/service.go", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for new file without tests")
	}
	if result.Findings[0].Severity != interfaces.SeverityMedium {
		t.Errorf("expected MEDIUM severity for new file, got %q", result.Findings[0].Severity)
	}
}

func TestCoverageAnalyzer_ModifiedFile_LowSeverity(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/service/service.go", Status: interfaces.FileModified, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for modified file without test changes")
	}
	if result.Findings[0].Severity != interfaces.SeverityLow {
		t.Errorf("expected LOW severity for modified file, got %q", result.Findings[0].Severity)
	}
}

func TestCoverageAnalyzer_PythonFile_WithTest_NoFinding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "app/services/user.py", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
			{Path: "app/services/test_user.py", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings when Python test file exists, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_PythonFile_WithoutTest_Finding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "app/services/user.py", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding when Python file has no test")
	}
}

func TestCoverageAnalyzer_JSFile_WithTestJS_NoFinding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "src/utils/format.js", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
			{Path: "src/utils/format.test.js", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings when JS test file exists, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_JSFile_WithSpecJS_NoFinding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "src/utils/format.js", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
			{Path: "src/utils/format.spec.js", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings when JS spec file exists, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_TSXFile_WithHooks_WithoutTest_Finding(t *testing.T) {
	// A .tsx file with hooks should be flagged for missing tests.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "src/components/BookingForm.tsx", Status: interfaces.FileAdded,
				Hunks: makeLargeHunkWithLines(
					`import { useState, useEffect } from "react"`,
					`const [value, setValue] = useState("")`,
					`useEffect(() => { fetch("/api/data") }, [])`,
				)},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding when TSX file with hooks has no test")
	}
}

func TestCoverageAnalyzer_TSFile_WithSpecTS_NoFinding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "src/services/api.ts", Status: interfaces.FileModified, Hunks: makeLargeHunk()},
			{Path: "src/services/api.spec.ts", Status: interfaces.FileModified, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings when TS spec file exists, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_TestFileOnly_NoFinding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/handler/handler_test.go", Status: interfaces.FileModified, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for test-only changes, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_SkipsDeletedFiles(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/old.go", Status: interfaces.FileDeleted},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for deleted files, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_SkipsBinaryFiles(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "image.png", Status: interfaces.FileAdded, IsBinary: true},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for binary files, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_UnknownExtension_NoFinding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "README.md", Status: interfaces.FileModified, Hunks: makeLargeHunk()},
			{Path: "config.yaml", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for unknown file types, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_EmptyDiff(t *testing.T) {
	diff := &interfaces.Diff{}
	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for empty diff, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/service.go", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	_, err := NewCoverageAnalyzer().Analyze(ctx, diff)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestCoverageAnalyzer_MultipleFiles_MixedResults(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			// Go file with test — no finding.
			{Path: "pkg/a/a.go", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
			{Path: "pkg/a/a_test.go", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
			// Go file without test — finding.
			{Path: "pkg/b/b.go", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
			// JS file without test — finding.
			{Path: "src/c.js", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
			// Markdown — no finding (unknown type).
			{Path: "docs/README.md", Status: interfaces.FileModified, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 findings (b.go and c.js), got %d: %v", len(result.Findings), findingIDs(result.Findings))
	}
}

func TestCoverageAnalyzer_RubyFile_WithoutTest_Finding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "app/models/user.rb", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for Ruby file without test")
	}
}

// --- New tests: Frontend exempt file patterns ---

func TestCoverageAnalyzer_NextJSLayoutPage_NoFinding(t *testing.T) {
	exemptPaths := []string{
		"app/layout.tsx",
		"app/layout.jsx",
		"app/page.tsx",
		"app/page.jsx",
		"app/dashboard/layout.tsx",
		"app/settings/page.tsx",
	}

	for _, path := range exemptPaths {
		t.Run(path, func(t *testing.T) {
			diff := &interfaces.Diff{
				Files: []interfaces.FileDiff{
					{Path: path, Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
				},
			}
			result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("expected no findings for exempt file %q, got %d", path, len(result.Findings))
			}
		})
	}
}

func TestCoverageAnalyzer_ConfigFiles_NoFinding(t *testing.T) {
	configPaths := []string{
		"next.config.js",
		"next.config.mjs",
		"next.config.ts",
		"tailwind.config.js",
		"tailwind.config.ts",
		"postcss.config.js",
		"postcss.config.mjs",
		"tsconfig.json",
		"tsconfig.app.json",
		"eslint.config.js",
		"eslint.config.mjs",
		"prettier.config.js",
	}

	for _, path := range configPaths {
		t.Run(path, func(t *testing.T) {
			diff := &interfaces.Diff{
				Files: []interfaces.FileDiff{
					{Path: path, Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
				},
			}
			result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("expected no findings for config file %q, got %d", path, len(result.Findings))
			}
		})
	}
}

func TestCoverageAnalyzer_TypeDefinitionFiles_NoFinding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "src/types/global.d.ts", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
			{Path: "src/env.d.ts", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for .d.ts files, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_StyleFiles_NoFinding(t *testing.T) {
	stylePaths := []string{
		"src/styles/globals.css",
		"src/styles/theme.scss",
		"app/globals.css",
	}

	for _, path := range stylePaths {
		t.Run(path, func(t *testing.T) {
			diff := &interfaces.Diff{
				Files: []interfaces.FileDiff{
					{Path: path, Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
				},
			}
			result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("expected no findings for style file %q, got %d", path, len(result.Findings))
			}
		})
	}
}

func TestCoverageAnalyzer_ConstantsAndAnimations_NoFinding(t *testing.T) {
	exemptPaths := []string{
		"src/lib/constants.ts",
		"src/lib/constants.tsx",
		"src/config/animations.ts",
		"src/config/animations.tsx",
	}

	for _, path := range exemptPaths {
		t.Run(path, func(t *testing.T) {
			diff := &interfaces.Diff{
				Files: []interfaces.FileDiff{
					{Path: path, Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
				},
			}
			result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("expected no findings for exempt file %q, got %d", path, len(result.Findings))
			}
		})
	}
}

func TestCoverageAnalyzer_SmallFile_NoFinding(t *testing.T) {
	// Files under 20 lines should not be flagged even without tests.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/util/helper.go", Status: interfaces.FileAdded, Hunks: makeSmallHunk()},
			{Path: "src/utils/index.ts", Status: interfaces.FileAdded, Hunks: makeSmallHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for small files under threshold, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_ExactlyAtThreshold_NoFinding(t *testing.T) {
	// A file with exactly 19 lines (< 20) should not be flagged.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/util/helper.go", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: makeAddedLines(19)}}},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for file with 19 lines, got %d", len(result.Findings))
	}
}

func TestCoverageAnalyzer_AtThreshold_Finding(t *testing.T) {
	// A file with exactly 20 lines (>= 20) should be flagged if no tests.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/util/helper.go", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: makeAddedLines(20)}}},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for file at 20-line threshold without tests")
	}
}

func TestCoverageAnalyzer_TSXWithLogic_StillFlagged(t *testing.T) {
	// A .tsx file with logic patterns should still be flagged.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "src/components/Dashboard.tsx", Status: interfaces.FileAdded,
				Hunks: makeLargeHunkWithLines(
					`import { useReducer, useCallback } from "react"`,
					`const [state, dispatch] = useReducer(reducer, initial)`,
				)},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for TSX file with logic patterns but no tests")
	}
}

func TestCoverageAnalyzer_PresentationalComponent_NoFinding(t *testing.T) {
	// Pure presentational .tsx/.jsx components without hooks/logic
	// should NOT be flagged for missing tests.
	presentationalFiles := []struct {
		name string
		path string
	}{
		{"footer component", "src/components/Footer.tsx"},
		{"header component", "src/components/Header.tsx"},
		{"hero section", "src/components/HeroSection.tsx"},
		{"jsx card", "src/components/Card.jsx"},
	}

	for _, tt := range presentationalFiles {
		t.Run(tt.name, func(t *testing.T) {
			diff := &interfaces.Diff{
				Files: []interfaces.FileDiff{
					{Path: tt.path, Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
				},
			}
			result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("expected no findings for presentational component %q, got %d",
					tt.path, len(result.Findings))
			}
		})
	}
}

func TestCoverageAnalyzer_TSXWithAsyncFetch_Finding(t *testing.T) {
	// A .tsx file with fetch() should be flagged.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "src/components/DataLoader.tsx", Status: interfaces.FileAdded,
				Hunks: makeLargeHunkWithLines(
					`const data = await fetch("/api/users")`,
				)},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for TSX file with fetch() but no tests")
	}
}

func TestCoverageAnalyzer_GoFile_NotAffectedByLogicCheck(t *testing.T) {
	// Non-React files (.go, .py, .js, .ts) should still be flagged
	// regardless of content — the logic pattern check only applies to .tsx/.jsx.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/service/service.go", Status: interfaces.FileAdded, Hunks: makeLargeHunk()},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for .go file without tests (logic check should not apply)")
	}
}
