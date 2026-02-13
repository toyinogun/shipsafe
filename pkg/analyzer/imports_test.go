package analyzer

import (
	"context"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

func TestImportsAnalyzer_Name(t *testing.T) {
	a := NewImportsAnalyzer()
	if a.Name() != "imports" {
		t.Errorf("expected name %q, got %q", "imports", a.Name())
	}
}

func TestImportsAnalyzer_ImplementsInterface(t *testing.T) {
	var _ Analyzer = NewImportsAnalyzer()
}

func TestImportsAnalyzer_GoMod_NewDependency(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "go.mod",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 5, Content: `	github.com/stretchr/testify v1.9.0`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for new Go dependency")
	}
	if !hasFindingWithID(result.Findings, "IMP-NEW") {
		t.Fatalf("expected IMP-NEW finding, got: %v", findingIDs(result.Findings))
	}
	assertAllFindingsHaveSeverity(t, result.Findings, "IMP-NEW", interfaces.SeverityLow)
}

func TestImportsAnalyzer_GoMod_MajorVersionBump(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "go.mod",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						RemovedLines: []interfaces.Line{
							{Number: 5, Content: `	github.com/go-chi/chi v4.1.2`},
						},
						AddedLines: []interfaces.Line{
							{Number: 5, Content: `	github.com/go-chi/chi v5.0.0`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasFindingWithID(result.Findings, "IMP-MAJOR") {
		t.Fatalf("expected IMP-MAJOR finding for major version bump, got: %v", findingIDs(result.Findings))
	}
	assertAllFindingsHaveSeverity(t, result.Findings, "IMP-MAJOR", interfaces.SeverityMedium)
}

func TestImportsAnalyzer_GoMod_DependencyRemoved(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "go.mod",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						RemovedLines: []interfaces.Line{
							{Number: 5, Content: `	github.com/old/dep v1.0.0`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasFindingWithID(result.Findings, "IMP-REMOVED") {
		t.Fatalf("expected IMP-REMOVED finding, got: %v", findingIDs(result.Findings))
	}
	assertAllFindingsHaveSeverity(t, result.Findings, "IMP-REMOVED", interfaces.SeverityInfo)
}

func TestImportsAnalyzer_PackageJSON_NewDependency(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "package.json",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 10, Content: `    "express": "^4.18.0"`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for new npm dependency")
	}
	f := result.Findings[0]
	if f.Category != interfaces.CategoryImport {
		t.Errorf("expected category %q, got %q", interfaces.CategoryImport, f.Category)
	}
}

func TestImportsAnalyzer_PackageJSON_MajorBump(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "package.json",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						RemovedLines: []interfaces.Line{
							{Number: 10, Content: `    "react": "^17.0.0"`},
						},
						AddedLines: []interfaces.Line{
							{Number: 10, Content: `    "react": "^18.0.0"`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasFindingWithID(result.Findings, "IMP-MAJOR") {
		t.Fatalf("expected IMP-MAJOR finding for npm major bump, got: %v", findingIDs(result.Findings))
	}
}

func TestImportsAnalyzer_RequirementsTxt_NewDependency(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "requirements.txt",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 3, Content: `flask>=2.0.0`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for new Python dependency")
	}
}

func TestImportsAnalyzer_CargoToml_NewDependency(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "Cargo.toml",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 8, Content: `serde = "1.0"`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for new Rust dependency")
	}
}

func TestImportsAnalyzer_Gemfile_NewDependency(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "Gemfile",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 5, Content: `gem 'rails', '~> 7.0'`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for new Ruby dependency")
	}
}

func TestImportsAnalyzer_PomXML_NewDependency(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "pom.xml",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 20, Content: `    <dependency>`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for new Maven dependency")
	}
}

func TestImportsAnalyzer_NonManifestFile_NoFindings(t *testing.T) {
	files := []string{
		"main.go",
		"src/index.ts",
		"README.md",
		"config.yaml",
	}

	for _, path := range files {
		t.Run(path, func(t *testing.T) {
			diff := diffWithAddedLines(path, `some content v1.0.0`)
			result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("expected no findings for non-manifest file %q, got %d", path, len(result.Findings))
			}
		})
	}
}

func TestImportsAnalyzer_NestedManifest_Detected(t *testing.T) {
	// go.mod in a subdirectory should still be detected.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "services/api/go.mod",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 5, Content: `	github.com/new/dep v1.0.0`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding for nested manifest file")
	}
}

func TestImportsAnalyzer_MinorVersionBump_NotMajor(t *testing.T) {
	// Minor/patch bumps should be flagged as new, not major.
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "go.mod",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						RemovedLines: []interfaces.Line{
							{Number: 5, Content: `	github.com/foo/bar v1.2.0`},
						},
						AddedLines: []interfaces.Line{
							{Number: 5, Content: `	github.com/foo/bar v1.3.0`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasFindingWithID(result.Findings, "IMP-MAJOR") {
		t.Fatal("minor version bump should not produce IMP-MAJOR finding")
	}
}

func TestImportsAnalyzer_SkipsBinaryFiles(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:     "go.mod",
				Status:   interfaces.FileModified,
				IsBinary: true,
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for binary files, got %d", len(result.Findings))
	}
}

func TestImportsAnalyzer_EmptyDiff(t *testing.T) {
	diff := &interfaces.Diff{}
	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for empty diff, got %d", len(result.Findings))
	}
}

func TestImportsAnalyzer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "go.mod",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 5, Content: `	github.com/new/dep v1.0.0`},
						},
					},
				},
			},
		},
	}

	_, err := NewImportsAnalyzer().Analyze(ctx, diff)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestImportsAnalyzer_CommentLines_Skipped(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "go.mod",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 5, Content: `// github.com/commented/dep v1.0.0`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for commented lines, got %d", len(result.Findings))
	}
}

func TestImportsAnalyzer_MetadataIncluded(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "go.mod",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 5, Content: `	github.com/new/dep v1.0.0`},
						},
					},
				},
			},
		},
	}

	result, err := NewImportsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected findings")
	}
	f := result.Findings[0]
	if f.Metadata == nil {
		t.Fatal("expected metadata on imports finding")
	}
	if _, ok := f.Metadata["language"]; !ok {
		t.Error("expected 'language' key in metadata")
	}
	if _, ok := f.Metadata["manifest"]; !ok {
		t.Error("expected 'manifest' key in metadata")
	}
}

func TestExtractDepName(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"go module", `	github.com/stretchr/testify v1.9.0`, "github.com/stretchr/testify"},
		{"npm package", `    "express": "^4.18.0"`, "express"},
		{"pip package", `flask>=2.0.0`, "flask"},
		{"ruby gem", `gem 'rails', '~> 7.0'`, "rails"},
		{"cargo crate", `serde = "1.0"`, "serde"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDepName(tt.line)
			if got != tt.want {
				t.Errorf("extractDepName(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}
