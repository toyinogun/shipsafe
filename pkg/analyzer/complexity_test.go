package analyzer

import (
	"context"
	"strings"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

func TestComplexityAnalyzer_Name(t *testing.T) {
	a := NewComplexityAnalyzer()
	if a.Name() != "complexity" {
		t.Errorf("expected name %q, got %q", "complexity", a.Name())
	}
}

func TestComplexityAnalyzer_ImplementsInterface(t *testing.T) {
	var _ Analyzer = NewComplexityAnalyzer()
}

func TestComplexityAnalyzer_WithThresholdOption(t *testing.T) {
	a := NewComplexityAnalyzer(WithComplexityThreshold(10))
	if a.threshold != 10 {
		t.Errorf("expected threshold 10, got %d", a.threshold)
	}
}

func TestComplexityAnalyzer_HighComplexityFunction_ReturnsFinding(t *testing.T) {
	// Build a function with many decision points (complexity > 15).
	lines := []string{
		`func processData(input []string) error {`,
		`    if len(input) == 0 {`,
		`        return nil`,
		`    }`,
		`    for _, item := range input {`,
		`        if item == "" {`,
		`            continue`,
		`        }`,
		`        if strings.HasPrefix(item, "a") {`,
		`            if len(item) > 5 {`,
		`                for _, c := range item {`,
		`                    if c == 'x' || c == 'y' {`,
		`                        break`,
		`                    }`,
		`                }`,
		`            }`,
		`        } else if strings.HasPrefix(item, "b") {`,
		`            switch item {`,
		`            case "ba":`,
		`            case "bb":`,
		`            case "bc":`,
		`            case "bd":`,
		`            }`,
		`        } else if item == "c" && len(item) > 0 {`,
		`            while true {`,
		`                if done {`,
		`                    break`,
		`                }`,
		`            }`,
		`        }`,
		`    }`,
		`    return nil`,
		`}`,
	}

	diff := diffWithAddedLines("processor.go", lines...)
	result, err := NewComplexityAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least one complexity finding for high-complexity function")
	}

	f := result.Findings[0]
	if f.Category != interfaces.CategoryComplexity {
		t.Errorf("expected category %q, got %q", interfaces.CategoryComplexity, f.Category)
	}
	if !strings.Contains(f.Title, "processData") {
		t.Errorf("expected title to contain function name, got %q", f.Title)
	}
}

func TestComplexityAnalyzer_VeryHighComplexity_ReturnsHighSeverity(t *testing.T) {
	// Build a function with complexity > 20.
	// Decision points: 8 if + 2 for + 5 case + 1 while + 2 || + 2 && + 1 elif = 21
	// Total complexity: 1 (base) + 21 = 22
	lines := []string{
		`func megaFunction(x int) int {`,
		`    if x > 0 {`,
		`        if x > 1 {`,
		`            if x > 2 {`,
		`                if x > 3 {`,
		`                    if x > 4 {`,
		`                        if x > 5 {`,
		`                            for i := 0; i < x; i++ {`,
		`                                if i%2 == 0 || i%3 == 0 {`,
		`                                    switch i {`,
		`                                    case 1:`,
		`                                    case 2:`,
		`                                    case 3:`,
		`                                    case 4:`,
		`                                    case 5:`,
		`                                    }`,
		`                                }`,
		`                                while x > 0 && i < 100 {`,
		`                                    if done || abort {`,
		`                                    }`,
		`                                }`,
		`                                for j := 0; j < i; j++ {`,
		`                                    if j > 0 && j < 50 {`,
		`                                    }`,
		`                                }`,
		`                            }`,
		`                        }`,
		`                    }`,
		`                }`,
		`            }`,
		`        }`,
		`    }`,
		`    return x`,
		`}`,
	}

	diff := diffWithAddedLines("mega.go", lines...)
	result, err := NewComplexityAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected findings for very high complexity function")
	}

	f := result.Findings[0]
	if f.Severity != interfaces.SeverityHigh {
		t.Errorf("expected HIGH severity for complexity > 20, got %q", f.Severity)
	}
}

func TestComplexityAnalyzer_LowComplexityFunction_NoFinding(t *testing.T) {
	lines := []string{
		`func add(a, b int) int {`,
		`    return a + b`,
		`}`,
	}

	diff := diffWithAddedLines("math.go", lines...)
	result, err := NewComplexityAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for simple function, got %d", len(result.Findings))
	}
}

func TestComplexityAnalyzer_MultipleFunctions_OnlyFlagsComplex(t *testing.T) {
	lines := []string{
		`func simple(x int) int {`,
		`    return x + 1`,
		`}`,
		`func complex(input []string) error {`,
		`    if len(input) == 0 {`,
		`        return nil`,
		`    }`,
		`    for _, item := range input {`,
		`        if item == "" {`,
		`            continue`,
		`        }`,
		`        if strings.HasPrefix(item, "a") {`,
		`            if len(item) > 5 {`,
		`                for _, c := range item {`,
		`                    if c == 'x' || c == 'y' {`,
		`                        break`,
		`                    }`,
		`                }`,
		`            }`,
		`        } else if strings.HasPrefix(item, "b") {`,
		`            switch item {`,
		`            case "ba":`,
		`            case "bb":`,
		`            case "bc":`,
		`            case "bd":`,
		`            }`,
		`        } else if item == "c" && len(item) > 0 {`,
		`            while true {`,
		`                if done {`,
		`                    break`,
		`                }`,
		`            }`,
		`        }`,
		`    }`,
		`    return nil`,
		`}`,
	}

	diff := diffWithAddedLines("mixed.go", lines...)
	result, err := NewComplexityAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the complex function should be flagged.
	for _, f := range result.Findings {
		if strings.Contains(f.Title, "simple") {
			t.Errorf("simple function should not be flagged, but got: %s", f.Title)
		}
	}
}

func TestComplexityAnalyzer_PythonFunction(t *testing.T) {
	// Python function with many decision points. Note: Python uses 'or'/'and'
	// but our regex matches && and || operators. We use if/for/while/elif/except.
	lines := []string{
		`def handle_request(request):`,
		`    if request.method == "GET":`,
		`        if request.user.is_authenticated:`,
		`            if request.user.is_admin:`,
		`                for item in request.items:`,
		`                    if item.status == "active":`,
		`                        if item.priority > 5:`,
		`                            for tag in item.tags:`,
		`                                if tag == "critical":`,
		`                                    if item.assigned:`,
		`                                        while item.retries < 3:`,
		`                                            if item.can_retry:`,
		`                                                pass`,
		`                                            elif item.force:`,
		`                                                pass`,
		`                                            elif item.skip:`,
		`                                                pass`,
		`                                            except:`,
		`                                                pass`,
		`                    elif item.status == "pending":`,
		`                        for sub in item.children:`,
		`                            if sub.ready:`,
		`                                pass`,
	}

	diff := diffWithAddedLines("views.py", lines...)
	result, err := NewComplexityAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected complexity finding for complex Python function")
	}
}

func TestComplexityAnalyzer_JavaScriptFunction(t *testing.T) {
	lines := []string{
		`function processEvents(events) {`,
		`    if (!events) return;`,
		`    for (const event of events) {`,
		`        if (event.type === "click") {`,
		`            if (event.target && event.target.id) {`,
		`                switch (event.target.id) {`,
		`                case "submit":`,
		`                case "cancel":`,
		`                case "reset":`,
		`                case "delete":`,
		`                }`,
		`            }`,
		`        } else if (event.type === "keydown") {`,
		`            if (event.key === "Enter" || event.key === "Escape") {`,
		`                for (let i = 0; i < handlers.length; i++) {`,
		`                    if (handlers[i].active && handlers[i].matches(event)) {`,
		`                        break;`,
		`                    }`,
		`                }`,
		`            }`,
		`        }`,
		`    }`,
		`}`,
	}

	diff := diffWithAddedLines("events.js", lines...)
	result, err := NewComplexityAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected complexity finding for complex JavaScript function")
	}
}

func TestComplexityAnalyzer_CustomThreshold(t *testing.T) {
	// With threshold=3, even a moderately complex function should be flagged.
	lines := []string{
		`func moderate(x int) int {`,
		`    if x > 0 {`,
		`        for i := 0; i < x; i++ {`,
		`            if i%2 == 0 {`,
		`                if i > 10 {`,
		`                    return i`,
		`                }`,
		`            }`,
		`        }`,
		`    }`,
		`    return 0`,
		`}`,
	}

	diff := diffWithAddedLines("math.go", lines...)
	result, err := NewComplexityAnalyzer(WithComplexityThreshold(3)).Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected finding with low threshold")
	}
}

func TestComplexityAnalyzer_CommentsNotCounted(t *testing.T) {
	lines := []string{
		`func example(x int) int {`,
		`    // if x > 0 then do something`,
		`    // for each item in the list`,
		`    // while condition is true`,
		`    /* case 1: */`,
		`    # elif this or that`,
		`    return x`,
		`}`,
	}

	diff := diffWithAddedLines("example.go", lines...)
	result, err := NewComplexityAnalyzer(WithComplexityThreshold(1)).Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("comments should not contribute to complexity, got %d findings", len(result.Findings))
	}
}

func TestComplexityAnalyzer_SkipsDeletedFiles(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "old.go",
				Status: interfaces.FileDeleted,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 1, Content: `func complex(x int) { if x > 0 { for i := range x { if i > 0 { if i > 1 { if i > 2 { if i > 3 { if i > 4 { if i > 5 { if i > 6 { if i > 7 { if i > 8 { if i > 9 { } } } } } } } } } } } } }`},
						},
					},
				},
			},
		},
	}

	result, err := NewComplexityAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for deleted files, got %d", len(result.Findings))
	}
}

func TestComplexityAnalyzer_EmptyDiff(t *testing.T) {
	diff := &interfaces.Diff{}
	result, err := NewComplexityAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for empty diff, got %d", len(result.Findings))
	}
}

func TestComplexityAnalyzer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	diff := diffWithAddedLines("main.go", `func main() {}`)
	_, err := NewComplexityAnalyzer().Analyze(ctx, diff)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestCountComplexity(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  int
	}{
		{
			name:  "empty function",
			lines: []string{`func foo() {`, `}`},
			want:  1, // base complexity
		},
		{
			name:  "single if",
			lines: []string{`if x > 0 {`, `}`},
			want:  2, // 1 base + 1 if
		},
		{
			name:  "if with logical operators",
			lines: []string{`if x > 0 && y < 10 || z == 5 {`},
			want:  4, // 1 base + 1 if + 1 && + 1 ||
		},
		{
			name:  "for loop",
			lines: []string{`for i := 0; i < n; i++ {`},
			want:  2, // 1 base + 1 for
		},
		{
			name:  "switch with cases",
			lines: []string{`switch x {`, `case 1:`, `case 2:`, `case 3:`},
			want:  4, // 1 base + 3 case
		},
		{
			name:  "ternary",
			lines: []string{`result := x > 0 ? "positive" : "negative"`},
			want:  2, // 1 base + 1 ternary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countComplexity(tt.lines)
			if got != tt.want {
				t.Errorf("countComplexity() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExtractFuncName(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantName string
		wantOK   bool
	}{
		{"go func", `func processData(input string) error {`, "processData", true},
		{"go method", `func (s *Server) Start(ctx context.Context) error {`, "Start", true},
		{"js function", `function handleClick(event) {`, "handleClick", true},
		{"async function", `async function fetchData() {`, "fetchData", true},
		{"python def", `def calculate_total(items):`, "calculate_total", true},
		{"rust fn", `fn process_data(input: &str) -> Result<()> {`, "process_data", true},
		{"pub rust fn", `pub fn new() -> Self {`, "new", true},
		{"java method", `    public static void processData(String input) {`, "processData", true},
		{"not a function", `    x := 42`, "", false},
		{"variable", `    name := "hello"`, "", false},
		{"if statement", `    if len(input) == 0 {`, "", false},
		{"for loop", `    for _, item := range input {`, "", false},
		{"while loop", `    while true {`, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, ok := extractFuncName(tt.line)
			if ok != tt.wantOK {
				t.Errorf("extractFuncName(%q) ok = %v, want %v", tt.line, ok, tt.wantOK)
			}
			if ok && name != tt.wantName {
				t.Errorf("extractFuncName(%q) name = %q, want %q", tt.line, name, tt.wantName)
			}
		})
	}
}

func TestComplexityAnalyzer_TestFile_HigherThreshold(t *testing.T) {
	// Build a function with complexity ~18 (above default 15, below test threshold 25).
	lines := []string{
		`func TestProcessData_ComplexFixtures(t *testing.T) {`,
		`    if len(input) == 0 {`,
		`        return`,
		`    }`,
		`    for _, item := range input {`,
		`        if item == "" {`,
		`            continue`,
		`        }`,
		`        if strings.HasPrefix(item, "a") {`,
		`            if len(item) > 5 {`,
		`                for _, c := range item {`,
		`                    if c == 'x' || c == 'y' {`,
		`                        break`,
		`                    }`,
		`                }`,
		`            }`,
		`        } else if strings.HasPrefix(item, "b") {`,
		`            switch item {`,
		`            case "ba":`,
		`            case "bb":`,
		`            case "bc":`,
		`            case "bd":`,
		`            }`,
		`        } else if item == "c" && len(item) > 0 {`,
		`            if done {`,
		`                break`,
		`            }`,
		`        }`,
		`    }`,
		`}`,
	}

	testPaths := []string{
		"pkg/analyzer/secrets_test.go",
		"src/utils.test.js",
		"tests/test_handler.py",
		"src/utils.spec.ts",
	}

	for _, path := range testPaths {
		t.Run(path, func(t *testing.T) {
			diff := diffWithAddedLines(path, lines...)
			result, err := NewComplexityAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("test file %q should have boosted threshold, got %d findings: %v",
					path, len(result.Findings), result.Findings[0].Title)
			}
		})
	}
}

func TestComplexityAnalyzer_TestFile_StillFlagsExtremeComplexity(t *testing.T) {
	// Build a function with complexity well above the boosted threshold (25).
	// Need > 25 decision points + base 1 = 26+ total.
	lines := []string{
		`func TestMega_ExtremeComplexity(t *testing.T) {`,
		`    if x > 0 {`,
		`        if x > 1 {`,
		`            if x > 2 {`,
		`                if x > 3 {`,
		`                    if x > 4 {`,
		`                        if x > 5 {`,
		`                            if x > 6 {`,
		`                                if x > 7 {`,
		`                                    for i := 0; i < x; i++ {`,
		`                                        if i%2 == 0 || i%3 == 0 {`,
		`                                            switch i {`,
		`                                            case 1:`,
		`                                            case 2:`,
		`                                            case 3:`,
		`                                            case 4:`,
		`                                            case 5:`,
		`                                            }`,
		`                                        }`,
		`                                        while x > 0 && i < 100 {`,
		`                                            if done || abort {`,
		`                                            }`,
		`                                        }`,
		`                                        for j := 0; j < i; j++ {`,
		`                                            if j > 0 && j < 50 {`,
		`                                            }`,
		`                                        }`,
		`                                        if extra || flag {`,
		`                                        }`,
		`                                    }`,
		`                                }`,
		`                            }`,
		`                        }`,
		`                    }`,
		`                }`,
		`            }`,
		`        }`,
		`    }`,
		`}`,
	}

	diff := diffWithAddedLines("mega_test.go", lines...)
	result, err := NewComplexityAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("extremely complex test function should still be flagged")
	}
}

func TestComplexityAnalyzer_NonTestFile_OriginalThreshold(t *testing.T) {
	// Ensure non-test files still use the original threshold.
	lines := []string{
		`func processData(input []string) error {`,
		`    if len(input) == 0 {`,
		`        return nil`,
		`    }`,
		`    for _, item := range input {`,
		`        if item == "" {`,
		`            continue`,
		`        }`,
		`        if strings.HasPrefix(item, "a") {`,
		`            if len(item) > 5 {`,
		`                for _, c := range item {`,
		`                    if c == 'x' || c == 'y' {`,
		`                        break`,
		`                    }`,
		`                }`,
		`            }`,
		`        } else if strings.HasPrefix(item, "b") {`,
		`            switch item {`,
		`            case "ba":`,
		`            case "bb":`,
		`            case "bc":`,
		`            case "bd":`,
		`            }`,
		`        } else if item == "c" && len(item) > 0 {`,
		`            while true {`,
		`                if done {`,
		`                    break`,
		`                }`,
		`            }`,
		`        }`,
		`    }`,
		`    return nil`,
		`}`,
	}

	diff := diffWithAddedLines("processor.go", lines...)
	result, err := NewComplexityAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("non-test file should use original threshold and flag this function")
	}
}

func TestComplexityAnalyzer_MetadataIncluded(t *testing.T) {
	// Use a low threshold to guarantee a finding for metadata inspection.
	lines := []string{
		`func handler(x int) int {`,
		`    if x > 0 {`,
		`        if x > 1 {`,
		`            for i := 0; i < x; i++ {`,
		`                if i > 10 {`,
		`                    return i`,
		`                }`,
		`            }`,
		`        }`,
		`    }`,
		`    return 0`,
		`}`,
	}

	diff := diffWithAddedLines("processor.go", lines...)
	result, err := NewComplexityAnalyzer(WithComplexityThreshold(3)).Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected findings")
	}

	f := result.Findings[0]
	if f.Metadata == nil {
		t.Fatal("expected metadata on complexity finding")
	}
	if _, ok := f.Metadata["complexity"]; !ok {
		t.Error("expected 'complexity' key in metadata")
	}
	if _, ok := f.Metadata["threshold"]; !ok {
		t.Error("expected 'threshold' key in metadata")
	}
}
