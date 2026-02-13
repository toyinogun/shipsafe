package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// mockProvider is a test double for LLMProvider.
type mockProvider struct {
	responses []string // responses to return in order
	callIndex int
	available bool
	err       error
}

func (m *mockProvider) Complete(_ context.Context, _ string, _ CompletionOpts) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.callIndex >= len(m.responses) {
		return `{"findings": []}`, nil
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp, nil
}

func (m *mockProvider) Available(_ context.Context) bool {
	return m.available
}

func TestReviewer_Review_WellFormattedResponse(t *testing.T) {
	provider := &mockProvider{
		available: true,
		responses: []string{
			// Semantic pass
			`{"findings": [{"file": "pkg/auth/handler.go", "line": 42, "severity": "medium", "title": "PR says login but code does logout", "description": "The PR title mentions adding login but the code implements a logout handler.", "suggestion": "Rename the handler or update the PR description."}]}`,
			// Logic pass
			`{"findings": [{"file": "pkg/auth/handler.go", "line": 55, "severity": "high", "title": "Nil pointer dereference", "description": "User object is used without nil check after database lookup.", "suggestion": "Add a nil check before accessing user fields."}]}`,
			// Convention pass
			`{"findings": []}`,
		},
	}

	reviewer := NewReviewer(provider)
	diff := &interfaces.Diff{
		PRTitle: "Add login handler",
		Files: []interfaces.FileDiff{
			{
				Path:     "pkg/auth/handler.go",
				Status:   interfaces.FileModified,
				Language: "go",
				Hunks: []interfaces.Hunk{
					{
						NewStart: 40,
						NewLines: 20,
						Content:  "+func HandleLogout(w http.ResponseWriter, r *http.Request) {\n",
						AddedLines: []interfaces.Line{
							{Number: 42, Content: "func HandleLogout(w http.ResponseWriter, r *http.Request) {"},
						},
					},
				},
			},
		},
	}

	result, err := reviewer.Review(context.Background(), diff, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.AnalyzerName != "ai-reviewer" {
		t.Errorf("expected analyzer name 'ai-reviewer', got %q", result.AnalyzerName)
	}

	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(result.Findings))
	}

	// Check first finding (semantic).
	f0 := result.Findings[0]
	if f0.Category != interfaces.CategoryLogic {
		t.Errorf("expected category 'logic', got %q", f0.Category)
	}
	if f0.Severity != interfaces.SeverityMedium {
		t.Errorf("expected severity 'medium', got %q", f0.Severity)
	}
	if f0.Source != "ai-reviewer" {
		t.Errorf("expected source 'ai-reviewer', got %q", f0.Source)
	}
	if f0.File != "pkg/auth/handler.go" {
		t.Errorf("expected file 'pkg/auth/handler.go', got %q", f0.File)
	}

	// Check second finding (logic).
	f1 := result.Findings[1]
	if f1.Severity != interfaces.SeverityHigh {
		t.Errorf("expected severity 'high', got %q", f1.Severity)
	}
}

func TestReviewer_Review_MalformedResponse(t *testing.T) {
	provider := &mockProvider{
		available: true,
		responses: []string{
			"This is not JSON at all, just some random text about the code.",
			"Also not JSON.",
			"Still not JSON.",
		},
	}

	reviewer := NewReviewer(provider)
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "main.go", Status: interfaces.FileModified, Language: "go"},
		},
	}

	result, err := reviewer.Review(context.Background(), diff, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings for malformed responses, got %d", len(result.Findings))
	}
}

func TestReviewer_Review_MarkdownCodeFences(t *testing.T) {
	provider := &mockProvider{
		available: true,
		responses: []string{
			"```json\n{\"findings\": [{\"file\": \"main.go\", \"line\": 10, \"severity\": \"low\", \"title\": \"Test finding\", \"description\": \"A test finding wrapped in code fences.\", \"suggestion\": \"Fix it.\"}]}\n```",
			`{"findings": []}`,
			`{"findings": []}`,
		},
	}

	reviewer := NewReviewer(provider)
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "main.go", Status: interfaces.FileModified, Language: "go"},
		},
	}

	result, err := reviewer.Review(context.Background(), diff, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding from code-fenced response, got %d", len(result.Findings))
	}

	if result.Findings[0].Title != "Test finding" {
		t.Errorf("expected title 'Test finding', got %q", result.Findings[0].Title)
	}
}

func TestReviewer_Review_ProviderError(t *testing.T) {
	provider := &mockProvider{
		available: true,
		err:       fmt.Errorf("connection refused"),
	}

	reviewer := NewReviewer(provider)
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "main.go", Status: interfaces.FileModified, Language: "go"},
		},
	}

	result, err := reviewer.Review(context.Background(), diff, nil)
	if err != nil {
		t.Fatalf("unexpected error (should degrade gracefully): %v", err)
	}

	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings when provider errors, got %d", len(result.Findings))
	}
}

func TestReviewer_Available_NotConfigured(t *testing.T) {
	provider := &mockProvider{available: false}
	reviewer := NewReviewer(provider)

	if reviewer.Available(context.Background()) {
		t.Error("expected Available() to return false for unconfigured provider")
	}
}

func TestReviewer_Available_Configured(t *testing.T) {
	provider := &mockProvider{available: true}
	reviewer := NewReviewer(provider)

	if !reviewer.Available(context.Background()) {
		t.Error("expected Available() to return true for configured provider")
	}
}

func TestReviewer_Review_EmptyFindings(t *testing.T) {
	provider := &mockProvider{
		available: true,
		responses: []string{
			`{"findings": []}`,
			`{"findings": []}`,
			`{"findings": []}`,
		},
	}

	reviewer := NewReviewer(provider)
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "clean.go", Status: interfaces.FileModified, Language: "go"},
		},
	}

	result, err := reviewer.Review(context.Background(), diff, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings for clean code, got %d", len(result.Findings))
	}
}

func TestReviewer_Review_PartiallyValidFindings(t *testing.T) {
	provider := &mockProvider{
		available: true,
		responses: []string{
			// One valid finding, one missing required fields.
			`{"findings": [
				{"file": "main.go", "line": 1, "severity": "high", "title": "Real issue", "description": "A real issue."},
				{"file": "main.go", "line": 2, "severity": "low", "title": "", "description": ""}
			]}`,
			`{"findings": []}`,
			`{"findings": []}`,
		},
	}

	reviewer := NewReviewer(provider)
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "main.go", Status: interfaces.FileModified, Language: "go"},
		},
	}

	result, err := reviewer.Review(context.Background(), diff, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 valid finding (skipping incomplete one), got %d", len(result.Findings))
	}
}

func TestReviewer_Review_ContextCancellation(t *testing.T) {
	provider := &mockProvider{available: true}

	reviewer := NewReviewer(provider)
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "main.go", Status: interfaces.FileModified, Language: "go"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := reviewer.Review(ctx, diff, nil)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestBuildContext_BasicDiff(t *testing.T) {
	diff := &interfaces.Diff{
		PRTitle: "Add authentication",
		PRBody:  "This PR adds JWT-based auth.",
		Files: []interfaces.FileDiff{
			{
				Path:     "pkg/auth/handler.go",
				Status:   interfaces.FileAdded,
				Language: "go",
				Hunks: []interfaces.Hunk{
					{
						NewStart: 1,
						NewLines: 5,
						Content:  "+package auth\n+\n+func Login() {}\n",
						AddedLines: []interfaces.Line{
							{Number: 1, Content: "package auth"},
							{Number: 3, Content: "func Login() {}"},
						},
					},
				},
			},
		},
	}

	ctx := BuildContext(diff, DefaultMaxTokenBudget)

	if !strings.Contains(ctx, "Add authentication") {
		t.Error("expected context to contain PR title")
	}
	if !strings.Contains(ctx, "JWT-based auth") {
		t.Error("expected context to contain PR body")
	}
	if !strings.Contains(ctx, "pkg/auth/handler.go") {
		t.Error("expected context to contain file path")
	}
	if !strings.Contains(ctx, "go") {
		t.Error("expected context to contain language")
	}
}

func TestBuildContext_TruncationOnLargeDiff(t *testing.T) {
	// Create a diff that exceeds the token budget.
	var files []interfaces.FileDiff
	for i := 0; i < 50; i++ {
		content := strings.Repeat(fmt.Sprintf("+line %d of file %d\n", i, i), 100)
		files = append(files, interfaces.FileDiff{
			Path:     fmt.Sprintf("pkg/module%d/file%d.go", i, i),
			Status:   interfaces.FileModified,
			Language: "go",
			Hunks: []interfaces.Hunk{
				{NewStart: 1, NewLines: 100, Content: content},
			},
		})
	}

	diff := &interfaces.Diff{Files: files}
	budget := 500 // Very small budget.
	ctx := BuildContext(diff, budget)

	maxChars := budget * charsPerToken
	if len(ctx) > maxChars+200 { // Allow some slack for the truncation message.
		t.Errorf("context length %d exceeds budget of ~%d chars", len(ctx), maxChars)
	}
}

func TestBuildContext_SecurityFilePriority(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:     "pkg/utils/helpers.go",
				Status:   interfaces.FileModified,
				Language: "go",
				Hunks: []interfaces.Hunk{
					{Content: "+func helper() {}"},
				},
			},
			{
				Path:     "pkg/auth/login.go",
				Status:   interfaces.FileModified,
				Language: "go",
				Hunks: []interfaces.Hunk{
					{Content: "+func Login() {}"},
				},
			},
		},
	}

	ctx := BuildContext(diff, DefaultMaxTokenBudget)

	// Auth file should appear before utils file.
	authIdx := strings.Index(ctx, "pkg/auth/login.go")
	utilsIdx := strings.Index(ctx, "pkg/utils/helpers.go")

	if authIdx == -1 || utilsIdx == -1 {
		t.Fatal("expected both files in context")
	}

	if authIdx > utilsIdx {
		t.Error("expected security-sensitive file (auth) to appear before utility file")
	}
}

func TestBuildContext_BinaryFilesSkipped(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "image.png", Status: interfaces.FileAdded, IsBinary: true},
			{Path: "main.go", Status: interfaces.FileModified, Language: "go", Hunks: []interfaces.Hunk{{Content: "+package main"}}},
		},
	}

	ctx := BuildContext(diff, DefaultMaxTokenBudget)

	if strings.Contains(ctx, "image.png") {
		t.Error("expected binary files to be skipped in context")
	}
	if !strings.Contains(ctx, "main.go") {
		t.Error("expected non-binary file in context")
	}
}

func TestParseFindings_ValidJSON(t *testing.T) {
	response := `{"findings": [{"file": "main.go", "line": 10, "severity": "high", "title": "Bug", "description": "A bug.", "suggestion": "Fix it."}]}`

	findings, confidence := parseFindings(response, interfaces.CategoryLogic)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if confidence < 0.9 {
		t.Errorf("expected high confidence, got %f", confidence)
	}
	if findings[0].Source != "ai-reviewer" {
		t.Errorf("expected source 'ai-reviewer', got %q", findings[0].Source)
	}
}

func TestParseFindings_InvalidJSON(t *testing.T) {
	response := "This is just plain text, not JSON at all."

	findings, confidence := parseFindings(response, interfaces.CategoryLogic)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for invalid JSON, got %d", len(findings))
	}
	if confidence != 0.0 {
		t.Errorf("expected 0 confidence for invalid JSON, got %f", confidence)
	}
}

func TestParseFindings_EmptyFindings(t *testing.T) {
	response := `{"findings": []}`

	findings, confidence := parseFindings(response, interfaces.CategoryConvention)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
	if confidence != 1.0 {
		t.Errorf("expected 1.0 confidence for valid empty response, got %f", confidence)
	}
}

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected interfaces.Severity
	}{
		{"critical", interfaces.SeverityCritical},
		{"CRITICAL", interfaces.SeverityCritical},
		{"high", interfaces.SeverityHigh},
		{"medium", interfaces.SeverityMedium},
		{"low", interfaces.SeverityLow},
		{"info", interfaces.SeverityInfo},
		{"unknown", interfaces.SeverityMedium},
		{"", interfaces.SeverityMedium},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapSeverity(tt.input)
			if got != tt.expected {
				t.Errorf("mapSeverity(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no fences",
			input:    `{"findings": []}`,
			expected: `{"findings": []}`,
		},
		{
			name:     "json fences",
			input:    "```json\n{\"findings\": []}\n```",
			expected: `{"findings": []}`,
		},
		{
			name:     "plain fences",
			input:    "```\n{\"findings\": []}\n```",
			expected: `{"findings": []}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCodeFences(tt.input)
			if got != tt.expected {
				t.Errorf("stripCodeFences() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWithMaxTokenBudget(t *testing.T) {
	provider := &mockProvider{available: true}
	reviewer := NewReviewer(provider, WithMaxTokenBudget(8000))

	if reviewer.maxTokenBudget != 8000 {
		t.Errorf("expected maxTokenBudget 8000, got %d", reviewer.maxTokenBudget)
	}
}
