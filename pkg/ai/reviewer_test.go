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

func TestDeduplicateFindings_ExactDuplicateKeepsHigherSeverity(t *testing.T) {
	findings := []interfaces.Finding{
		{File: "main.go", StartLine: 10, Severity: interfaces.SeverityMedium, Description: "null pointer dereference on user.profile.email"},
		{File: "main.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "null pointer dereference on user.profile.email"},
	}

	result := deduplicateFindings(findings)
	if len(result) != 1 {
		t.Fatalf("expected 1 finding after dedup, got %d", len(result))
	}
	if result[0].Severity != interfaces.SeverityHigh {
		t.Errorf("expected high severity to be kept, got %q", result[0].Severity)
	}
}

func TestDeduplicateFindings_NearbyLines(t *testing.T) {
	findings := []interfaces.Finding{
		{File: "auth.go", StartLine: 42, Severity: interfaces.SeverityHigh, Description: "missing return for else branch"},
		{File: "auth.go", StartLine: 44, Severity: interfaces.SeverityMedium, Description: "missing return for else branch"},
	}

	result := deduplicateFindings(findings)
	if len(result) != 1 {
		t.Fatalf("expected 1 finding after dedup (lines within threshold), got %d", len(result))
	}
	if result[0].Severity != interfaces.SeverityHigh {
		t.Errorf("expected high severity to be kept, got %q", result[0].Severity)
	}
}

func TestDeduplicateFindings_Lines5ApartIsDuplicate(t *testing.T) {
	findings := []interfaces.Finding{
		{File: "auth.go", StartLine: 40, Severity: interfaces.SeverityMedium, Description: "missing nil check before accessing user field"},
		{File: "auth.go", StartLine: 45, Severity: interfaces.SeverityHigh, Description: "missing nil check before accessing user field"},
	}

	result := deduplicateFindings(findings)
	if len(result) != 1 {
		t.Fatalf("expected 1 finding after dedup (lines 5 apart, within threshold of %d), got %d", LineProximityThreshold, len(result))
	}
	if result[0].Severity != interfaces.SeverityHigh {
		t.Errorf("expected high severity to be kept, got %q", result[0].Severity)
	}
}

func TestDeduplicateFindings_Lines6ApartIsDuplicate(t *testing.T) {
	findings := []interfaces.Finding{
		{File: "auth.go", StartLine: 40, Severity: interfaces.SeverityMedium, Description: "hardcoded secret detected in configuration"},
		{File: "auth.go", StartLine: 46, Severity: interfaces.SeverityHigh, Description: "hardcoded secret detected in configuration"},
	}

	result := deduplicateFindings(findings)
	if len(result) != 1 {
		t.Fatalf("expected 1 finding after dedup (lines exactly %d apart), got %d", LineProximityThreshold, len(result))
	}
}

func TestDeduplicateFindings_Lines7ApartNotDuplicate(t *testing.T) {
	findings := []interfaces.Finding{
		{File: "auth.go", StartLine: 40, Severity: interfaces.SeverityMedium, Description: "hardcoded secret detected in configuration"},
		{File: "auth.go", StartLine: 47, Severity: interfaces.SeverityHigh, Description: "hardcoded secret detected in configuration"},
	}

	result := deduplicateFindings(findings)
	if len(result) != 2 {
		t.Fatalf("expected 2 findings (lines 7 apart, beyond threshold of %d), got %d", LineProximityThreshold, len(result))
	}
}

func TestDeduplicateFindings_LinesTooFarApart(t *testing.T) {
	findings := []interfaces.Finding{
		{File: "auth.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "null pointer dereference on user object"},
		{File: "auth.go", StartLine: 50, Severity: interfaces.SeverityHigh, Description: "null pointer dereference on user object"},
	}

	result := deduplicateFindings(findings)
	if len(result) != 2 {
		t.Fatalf("expected 2 findings (different locations), got %d", len(result))
	}
}

func TestDeduplicateFindings_DifferentFiles(t *testing.T) {
	findings := []interfaces.Finding{
		{File: "auth.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "null pointer dereference on user"},
		{File: "handler.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "null pointer dereference on user"},
	}

	result := deduplicateFindings(findings)
	if len(result) != 2 {
		t.Fatalf("expected 2 findings (different files), got %d", len(result))
	}
}

func TestDeduplicateFindings_DifferentIssues(t *testing.T) {
	findings := []interfaces.Finding{
		{File: "main.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "null pointer dereference on user.profile.email"},
		{File: "main.go", StartLine: 11, Severity: interfaces.SeverityMedium, Description: "unused variable declared but never read"},
	}

	result := deduplicateFindings(findings)
	if len(result) != 2 {
		t.Fatalf("expected 2 findings (different issues), got %d", len(result))
	}
}

func TestDeduplicateFindings_SameSeverityKeepsEarlier(t *testing.T) {
	findings := []interfaces.Finding{
		{File: "main.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "null pointer dereference on user.profile", Source: "semantic"},
		{File: "main.go", StartLine: 10, Severity: interfaces.SeverityHigh, Description: "null pointer dereference on user.profile", Source: "logic"},
	}

	result := deduplicateFindings(findings)
	if len(result) != 1 {
		t.Fatalf("expected 1 finding after dedup, got %d", len(result))
	}
	if result[0].Source != "semantic" {
		t.Errorf("expected earlier (semantic) finding to be kept, got source %q", result[0].Source)
	}
}

func TestDeduplicateFindings_SimilarDescriptionsContainsKeyPhrase(t *testing.T) {
	findings := []interfaces.Finding{
		{File: "main.go", StartLine: 10, Severity: interfaces.SeverityMedium, Description: "null pointer dereference on user.profile.email when accessing nested field"},
		{File: "main.go", StartLine: 11, Severity: interfaces.SeverityHigh, Description: "Potential null pointer dereference on user.profile.email — add a nil check before accessing the field"},
	}

	result := deduplicateFindings(findings)
	if len(result) != 1 {
		t.Fatalf("expected 1 finding after dedup (similar descriptions), got %d", len(result))
	}
	if result[0].Severity != interfaces.SeverityHigh {
		t.Errorf("expected high severity to be kept, got %q", result[0].Severity)
	}
}

func TestDeduplicateFindings_EmptyAndSingle(t *testing.T) {
	// Empty input.
	result := deduplicateFindings(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 findings for nil input, got %d", len(result))
	}

	// Single finding.
	single := []interfaces.Finding{
		{File: "main.go", StartLine: 1, Severity: interfaces.SeverityLow, Description: "something"},
	}
	result = deduplicateFindings(single)
	if len(result) != 1 {
		t.Errorf("expected 1 finding for single input, got %d", len(result))
	}
}

func TestDeduplicateFindings_RealisticScenario(t *testing.T) {
	// Simulates the real bug: semantic and logic passes both find the same 4 issues.
	findings := []interfaces.Finding{
		// Semantic pass findings.
		{File: "pkg/auth/handler.go", StartLine: 42, Severity: interfaces.SeverityMedium, Description: "null pointer dereference on user.profile.email when accessing without nil check"},
		{File: "pkg/auth/handler.go", StartLine: 55, Severity: interfaces.SeverityMedium, Description: "missing return for else branch in validation function"},
		{File: "pkg/api/router.go", StartLine: 20, Severity: interfaces.SeverityHigh, Description: "SQL injection risk in query parameter handling"},
		{File: "pkg/api/router.go", StartLine: 88, Severity: interfaces.SeverityLow, Description: "unused error return value from database Close"},
		// Logic pass findings — same issues, slightly different wording.
		{File: "pkg/auth/handler.go", StartLine: 43, Severity: interfaces.SeverityHigh, Description: "null pointer dereference on user.profile.email — the user object may be nil after lookup"},
		{File: "pkg/auth/handler.go", StartLine: 55, Severity: interfaces.SeverityMedium, Description: "missing return for else branch causes function to fall through"},
		{File: "pkg/api/router.go", StartLine: 21, Severity: interfaces.SeverityHigh, Description: "SQL injection risk in query parameter — unsanitized input used in query"},
		{File: "pkg/api/router.go", StartLine: 88, Severity: interfaces.SeverityMedium, Description: "unused error return value from database Close call"},
	}

	result := deduplicateFindings(findings)
	if len(result) != 4 {
		t.Fatalf("expected 4 unique findings from 8 duplicates, got %d", len(result))
	}

	// Verify the higher-severity versions were kept.
	severityByFile := map[string]interfaces.Severity{}
	for _, f := range result {
		key := fmt.Sprintf("%s:%d", f.File, f.StartLine)
		severityByFile[key] = f.Severity
	}

	// null pointer: semantic=medium, logic=high → should keep high at line 43.
	if s, ok := severityByFile["pkg/auth/handler.go:43"]; !ok || s != interfaces.SeverityHigh {
		// Also acceptable if the kept finding is at line 42 with high severity.
		if s2, ok2 := severityByFile["pkg/auth/handler.go:42"]; !ok2 || s2 != interfaces.SeverityHigh {
			t.Errorf("expected high severity null pointer finding to be kept")
		}
	}
}

func TestSignificantWords(t *testing.T) {
	words := significantWords("The user object is nil after the database lookup")
	expected := []string{"user", "object", "nil", "database", "lookup"}
	if len(words) != len(expected) {
		t.Fatalf("expected %d words, got %d: %v", len(expected), len(words), words)
	}
	for i, w := range expected {
		if words[i] != w {
			t.Errorf("word[%d] = %q, want %q", i, words[i], w)
		}
	}
}

func TestDescriptionsSimilar(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected bool
	}{
		{
			name:     "identical",
			a:        "null pointer dereference on user.profile",
			b:        "null pointer dereference on user.profile",
			expected: true,
		},
		{
			name:     "same key words different wording",
			a:        "null pointer dereference on user.profile.email when accessing nested field",
			b:        "potential null pointer dereference on user.profile.email — add nil check",
			expected: true,
		},
		{
			name:     "completely different",
			a:        "null pointer dereference on user object",
			b:        "unused variable declared but never read",
			expected: false,
		},
		{
			name:     "empty strings",
			a:        "",
			b:        "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := descriptionsSimilar(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("descriptionsSimilar(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestSeverityRank(t *testing.T) {
	if severityRank(interfaces.SeverityCritical) <= severityRank(interfaces.SeverityHigh) {
		t.Error("critical should rank higher than high")
	}
	if severityRank(interfaces.SeverityHigh) <= severityRank(interfaces.SeverityMedium) {
		t.Error("high should rank higher than medium")
	}
	if severityRank(interfaces.SeverityMedium) <= severityRank(interfaces.SeverityLow) {
		t.Error("medium should rank higher than low")
	}
	if severityRank(interfaces.SeverityLow) <= severityRank(interfaces.SeverityInfo) {
		t.Error("low should rank higher than info")
	}
}

func TestReviewer_Review_DeduplicatesAcrossPasses(t *testing.T) {
	provider := &mockProvider{
		available: true,
		responses: []string{
			// Semantic pass — finds 2 issues.
			`{"findings": [
				{"file": "pkg/auth/handler.go", "line": 42, "severity": "medium", "title": "Nil pointer", "description": "null pointer dereference on user.profile.email"},
				{"file": "pkg/auth/handler.go", "line": 55, "severity": "medium", "title": "Missing return", "description": "missing return for else branch"}
			]}`,
			// Logic pass — finds same 2 issues with different severity.
			`{"findings": [
				{"file": "pkg/auth/handler.go", "line": 43, "severity": "high", "title": "Nil deref", "description": "null pointer dereference on user.profile.email may cause panic"},
				{"file": "pkg/auth/handler.go", "line": 55, "severity": "high", "title": "No return", "description": "missing return for else branch in validation"}
			]}`,
			// Convention pass — no findings.
			`{"findings": []}`,
		},
	}

	reviewer := NewReviewer(provider)
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{Path: "pkg/auth/handler.go", Status: interfaces.FileModified, Language: "go"},
		},
	}

	result, err := reviewer.Review(context.Background(), diff, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 deduplicated findings, got %d", len(result.Findings))
	}

	// Both should be high severity (kept from logic pass).
	for _, f := range result.Findings {
		if f.Severity != interfaces.SeverityHigh {
			t.Errorf("expected high severity after dedup, got %q for %q", f.Severity, f.Description)
		}
	}
}
