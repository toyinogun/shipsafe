package analyzer

import (
	"context"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

func TestPatternsAnalyzer_Name(t *testing.T) {
	a := NewPatternsAnalyzer()
	if a.Name() != "patterns" {
		t.Errorf("expected name %q, got %q", "patterns", a.Name())
	}
}

func TestPatternsAnalyzer_ImplementsInterface(t *testing.T) {
	var _ Analyzer = NewPatternsAnalyzer()
}

func TestPatternsAnalyzer_SQLConcatenation_Detected(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"select plus", `query := "SELECT * FROM users WHERE id=" + userID`},
		{"insert plus", `q := "INSERT INTO logs VALUES(" + val + ")"`},
		{"update plus", `db.Exec("UPDATE users SET name=" + name)`},
		{"delete plus", `stmt := "DELETE FROM sessions WHERE id=" + id`},
		{"sprintf select", `q := fmt.Sprintf("SELECT * FROM users WHERE id=%d", id)`},
		{"percent-s in query", `query := "SELECT * FROM users WHERE name='%s'"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines("db.go", tt.line)
			result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !hasFindingWithID(result.Findings, "PAT-SQL-CONCAT") {
				t.Fatalf("expected SQL concat finding for %q, got %d findings: %v", tt.line, len(result.Findings), findingIDs(result.Findings))
			}
			assertAllFindingsHaveSeverity(t, result.Findings, "PAT-SQL-CONCAT", interfaces.SeverityMedium)
		})
	}
}

func TestPatternsAnalyzer_SQLConcatenation_NotDetected(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"parameterized", `db.Query("SELECT * FROM users WHERE id = $1", id)`},
		{"orm", `users := db.Where("name = ?", name).Find(&users)`},
		{"string concat no sql", `msg := "hello " + name`},
		{"comment", `// SELECT * FROM users WHERE id=" + id`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines("db.go", tt.line)
			result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hasFindingWithID(result.Findings, "PAT-SQL-CONCAT") {
				t.Fatalf("unexpected SQL concat finding for safe code %q", tt.line)
			}
		})
	}
}

func TestPatternsAnalyzer_EmptyCatch_Detected(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"js empty catch", `} catch (e) {}`},
		{"java empty catch", `catch (Exception e) {}`},
		{"python bare except", `except:`},
		{"python typed except", `except ValueError:`},
		{"go-style empty catch", `catch {}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines("handler.go", tt.line)
			result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !hasFindingWithID(result.Findings, "PAT-EMPTY-CATCH") {
				t.Fatalf("expected empty catch finding for %q", tt.line)
			}
		})
	}
}

func TestPatternsAnalyzer_EmptyCatch_NotDetected(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"catch with body", `} catch (e) { log(e); }`},
		{"except with pass", `except ValueError as e:`},
		{"normal code", `x := 42`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines("handler.go", tt.line)
			result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hasFindingWithID(result.Findings, "PAT-EMPTY-CATCH") {
				t.Fatalf("unexpected empty catch finding for %q", tt.line)
			}
		})
	}
}

func TestPatternsAnalyzer_DebugPrint_Detected(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"console.log", `console.log("debug value:", x)`},
		{"console.debug", `console.debug(data)`},
		{"fmt.Println", `fmt.Println("debugging:", err)`},
		{"fmt.Printf", `fmt.Printf("value: %v\n", x)`},
		{"fmt.Print", `fmt.Print(result)`},
		{"print()", `print("debug")`},
		{"println()", `println("debug value")`},
		{"System.out.println", `System.out.println("debug")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines("service.go", tt.line)
			result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !hasFindingWithID(result.Findings, "PAT-DEBUG-PRINT") {
				t.Fatalf("expected debug print finding for %q, got: %v", tt.line, findingIDs(result.Findings))
			}
			assertAllFindingsHaveSeverity(t, result.Findings, "PAT-DEBUG-PRINT", interfaces.SeverityLow)
		})
	}
}

func TestPatternsAnalyzer_DebugPrint_SkippedInTestFiles(t *testing.T) {
	testPaths := []string{
		"pkg/handler/handler_test.go",
		"src/utils.test.js",
		"src/utils.spec.ts",
		"tests/test_helpers.py",
	}

	for _, path := range testPaths {
		t.Run(path, func(t *testing.T) {
			diff := diffWithAddedLines(path, `fmt.Println("debugging test")`)
			result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hasFindingWithID(result.Findings, "PAT-DEBUG-PRINT") {
				t.Fatalf("debug prints in test files should not be flagged: %s", path)
			}
		})
	}
}

func TestPatternsAnalyzer_DebugPrint_NotDetectedInComments(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"go comment", `// fmt.Println("debugging")`},
		{"python comment", `# print("debugging")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines("service.go", tt.line)
			result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hasFindingWithID(result.Findings, "PAT-DEBUG-PRINT") {
				t.Fatalf("commented-out debug print should not be flagged: %q", tt.line)
			}
		})
	}
}

func TestPatternsAnalyzer_TODO_Detected(t *testing.T) {
	tests := []struct {
		name string
		line string
		tag  string
	}{
		{"TODO", `// TODO: refactor this later`, "TODO"},
		{"FIXME", `# FIXME: handle edge case`, "FIXME"},
		{"HACK", `/* HACK: workaround for issue #123 */`, "HACK"},
		{"XXX", `// XXX: this is fragile`, "XXX"},
		{"lowercase todo", `// todo: clean up`, "TODO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines("service.go", tt.line)
			result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !hasFindingWithID(result.Findings, "PAT-TODO") {
				t.Fatalf("expected TODO finding for %q", tt.line)
			}
			assertAllFindingsHaveSeverity(t, result.Findings, "PAT-TODO", interfaces.SeverityInfo)
		})
	}
}

func TestPatternsAnalyzer_TODO_NotDetected(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"normal code", `x := 42`},
		{"normal comment", `// This function handles authentication`},
		{"string literal", `msg := "hello world"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines("service.go", tt.line)
			result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hasFindingWithID(result.Findings, "PAT-TODO") {
				t.Fatalf("unexpected TODO finding for %q", tt.line)
			}
		})
	}
}

func TestPatternsAnalyzer_MultiplePatterns_SingleLine(t *testing.T) {
	// A line with both a debug print and a TODO should produce two findings.
	diff := diffWithAddedLines("service.go",
		`fmt.Println("TODO: fix this later")`,
	)
	result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) < 2 {
		t.Fatalf("expected at least 2 findings (debug print + TODO), got %d: %v", len(result.Findings), findingIDs(result.Findings))
	}
}

func TestPatternsAnalyzer_SkipsDeletedFiles(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "old.go",
				Status: interfaces.FileDeleted,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 1, Content: `fmt.Println("debug")`},
						},
					},
				},
			},
		},
	}

	result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for deleted files, got %d", len(result.Findings))
	}
}

func TestPatternsAnalyzer_SkipsBinaryFiles(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:     "image.png",
				Status:   interfaces.FileAdded,
				IsBinary: true,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 1, Content: `console.log("debug")`},
						},
					},
				},
			},
		},
	}

	result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for binary files, got %d", len(result.Findings))
	}
}

func TestPatternsAnalyzer_EmptyDiff(t *testing.T) {
	diff := &interfaces.Diff{}
	result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for empty diff, got %d", len(result.Findings))
	}
}

func TestPatternsAnalyzer_NoFindingsOnCleanCode(t *testing.T) {
	diff := diffWithAddedLines("main.go",
		`func main() {`,
		`    slog.Info("starting server", "port", 8080)`,
		`    db.Query("SELECT * FROM users WHERE id = $1", id)`,
		`    if err != nil {`,
		`        return fmt.Errorf("query failed: %w", err)`,
		`    }`,
		`}`,
	)

	result, err := NewPatternsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings on clean code, got %d: %v", len(result.Findings), findingIDs(result.Findings))
	}
}

func TestPatternsAnalyzer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	diff := diffWithAddedLines("service.go", `fmt.Println("debug")`)
	_, err := NewPatternsAnalyzer().Analyze(ctx, diff)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

// --- test helpers ---

func hasFindingWithID(findings []interfaces.Finding, prefix string) bool {
	for _, f := range findings {
		if len(f.ID) >= len(prefix) && f.ID[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func findingIDs(findings []interfaces.Finding) []string {
	ids := make([]string, len(findings))
	for i, f := range findings {
		ids[i] = f.ID
	}
	return ids
}

func assertAllFindingsHaveSeverity(t *testing.T, findings []interfaces.Finding, idPrefix string, severity interfaces.Severity) {
	t.Helper()
	for _, f := range findings {
		if len(f.ID) >= len(idPrefix) && f.ID[:len(idPrefix)] == idPrefix {
			if f.Severity != severity {
				t.Errorf("finding %s has severity %q, expected %q", f.ID, f.Severity, severity)
			}
		}
	}
}
