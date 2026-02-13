package analyzer

import (
	"context"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

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
			{Path: "pkg/handler/handler.go", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "package handler"}}}}},
			{Path: "pkg/handler/handler_test.go", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "package handler"}}}}},
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
			{Path: "pkg/handler/handler.go", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "package handler"}}}}},
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
			{Path: "pkg/service/service.go", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "package service"}}}}},
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
			{Path: "pkg/service/service.go", Status: interfaces.FileModified, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "x := 1"}}}}},
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
			{Path: "app/services/user.py", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "class User:"}}}}},
			{Path: "app/services/test_user.py", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "class TestUser:"}}}}},
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
			{Path: "app/services/user.py", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "class User:"}}}}},
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
			{Path: "src/utils/format.js", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "export function format() {}"}}}}},
			{Path: "src/utils/format.test.js", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "test('format', () => {})"}}}}},
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
			{Path: "src/utils/format.js", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "export function format() {}"}}}}},
			{Path: "src/utils/format.spec.js", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "describe('format', () => {})"}}}}},
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

func TestCoverageAnalyzer_TSFile_WithoutTest_Finding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "src/components/Button.tsx", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "export const Button = () => {}"}}}}},
		},
	}

	result, err := NewCoverageAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding when TSX file has no test")
	}
}

func TestCoverageAnalyzer_TSFile_WithSpecTS_NoFinding(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "src/services/api.ts", Status: interfaces.FileModified, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "export async function fetch() {}"}}}}},
			{Path: "src/services/api.spec.ts", Status: interfaces.FileModified, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "test('fetch', () => {})"}}}}},
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
	// If only test files are modified, no coverage finding should be raised.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/handler/handler_test.go", Status: interfaces.FileModified, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "func TestNew(t *testing.T) {}"}}}}},
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
	// Files with unrecognized extensions should be skipped.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "README.md", Status: interfaces.FileModified, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "# Readme"}}}}},
			{Path: "config.yaml", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "key: value"}}}}},
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
			{Path: "pkg/service.go", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "package pkg"}}}}},
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
			{Path: "pkg/a/a.go", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "package a"}}}}},
			{Path: "pkg/a/a_test.go", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "package a"}}}}},
			// Go file without test — finding.
			{Path: "pkg/b/b.go", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "package b"}}}}},
			// JS file without test — finding.
			{Path: "src/c.js", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "export const c = 1"}}}}},
			// Markdown — no finding (unknown type).
			{Path: "docs/README.md", Status: interfaces.FileModified, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "# Docs"}}}}},
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
			{Path: "app/models/user.rb", Status: interfaces.FileAdded, Hunks: []interfaces.Hunk{{AddedLines: []interfaces.Line{{Number: 1, Content: "class User"}}}}},
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
