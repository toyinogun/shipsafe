package analyzer

import (
	"context"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// helper to build a Diff with a single file containing added lines.
func diffWithAddedLines(path string, lines ...string) *interfaces.Diff {
	added := make([]interfaces.Line, len(lines))
	for i, l := range lines {
		added[i] = interfaces.Line{Number: i + 1, Content: l}
	}
	return &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   path,
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						NewStart:   1,
						NewLines:   len(lines),
						AddedLines: added,
					},
				},
			},
		},
	}
}

func TestSecretsAnalyzer_Name(t *testing.T) {
	a := NewSecretsAnalyzer()
	if a.Name() != "secrets" {
		t.Errorf("expected name %q, got %q", "secrets", a.Name())
	}
}

func TestSecretsAnalyzer_AWSAccessKey(t *testing.T) {
	diff := diffWithAddedLines("config.go",
		`awsKey := "AKIAIOSFODNN7WKRB3PQ"`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least one finding for AWS access key")
	}
	assertHasFindingWithTitle(t, result.Findings, "AWS Access Key ID")
}

func TestSecretsAnalyzer_AWSSecretKey(t *testing.T) {
	diff := diffWithAddedLines("config.go",
		`aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYKZ6NR4TWAB`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least one finding for AWS secret key")
	}
	assertHasSeverity(t, result.Findings, interfaces.SeverityCritical)
}

func TestSecretsAnalyzer_PrivateKey(t *testing.T) {
	diff := diffWithAddedLines("deploy.sh",
		`-----BEGIN RSA PRIVATE KEY-----`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least one finding for private key")
	}
	assertHasSeverity(t, result.Findings, interfaces.SeverityCritical)
}

func TestSecretsAnalyzer_SSHPrivateKey(t *testing.T) {
	diff := diffWithAddedLines("id_rsa",
		`-----BEGIN OPENSSH PRIVATE KEY-----`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least one finding for SSH private key")
	}
	assertHasFindingWithTitle(t, result.Findings, "RSA/SSH Private Key")
}

func TestSecretsAnalyzer_GenericAPIKey(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"api_key equals", `api_key = "sk_live_1234567890abcdef"`},
		{"apikey equals", `apikey="abcdef1234567890ghij"`},
		{"api-key colon", `api-key: abcdefghijklmnopqrst`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines("config.yaml", tt.line)
			result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) == 0 {
				t.Fatalf("expected finding for %q", tt.line)
			}
		})
	}
}

func TestSecretsAnalyzer_BearerToken(t *testing.T) {
	diff := diffWithAddedLines("main.go",
		`Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkw`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least one finding for bearer token")
	}
	assertHasFindingWithTitle(t, result.Findings, "Bearer Token")
}

func TestSecretsAnalyzer_ConnectionStrings(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"postgres", `dsn := "postgres://user:pass@host:5432/dbname"`},
		{"mysql", `dsn := "mysql://root:secret@localhost/mydb"`},
		{"mongodb", `uri := "mongodb://admin:s3cretval@cluster.internal.io/db"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines("db.go", tt.line)
			result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) == 0 {
				t.Fatalf("expected finding for connection string: %s", tt.line)
			}
		})
	}
}

func TestSecretsAnalyzer_PasswordAssignment(t *testing.T) {
	diff := diffWithAddedLines("config.go",
		`password = "SuperS3cretP@ssword!"`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least one finding for password")
	}
}

func TestSecretsAnalyzer_GitHubToken(t *testing.T) {
	diff := diffWithAddedLines("ci.go",
		`token := "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef1234"`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least one finding for GitHub token")
	}
}

func TestSecretsAnalyzer_NoFindingsOnCleanCode(t *testing.T) {
	diff := diffWithAddedLines("main.go",
		`func main() {`,
		`    fmt.Println("hello world")`,
		`    x := 42`,
		`}`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings, got %d: %+v", len(result.Findings), result.Findings)
	}
}

func TestSecretsAnalyzer_SkipsRemovedLines(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "config.go",
				Status: interfaces.FileModified,
				Hunks: []interfaces.Hunk{
					{
						// Secret in removed lines only — should NOT flag.
						RemovedLines: []interfaces.Line{
							{Number: 1, Content: `password = "SuperS3cretP@ssword!"`},
						},
						AddedLines: []interfaces.Line{
							{Number: 1, Content: `password = os.Getenv("DB_PASSWORD")`},
						},
					},
				},
			},
		},
	}

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for removed lines, got %d", len(result.Findings))
	}
}

func TestSecretsAnalyzer_SkipsDeletedFiles(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:   "old-config.go",
				Status: interfaces.FileDeleted,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 1, Content: `password = "secret123456"`},
						},
					},
				},
			},
		},
	}

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for deleted files, got %d", len(result.Findings))
	}
}

func TestSecretsAnalyzer_SkipsBinaryFiles(t *testing.T) {
	diff := &interfaces.Diff{
		Files: []interfaces.FileDiff{
			{
				Path:     "image.png",
				Status:   interfaces.FileAdded,
				IsBinary: true,
				Hunks: []interfaces.Hunk{
					{
						AddedLines: []interfaces.Line{
							{Number: 1, Content: `AKIAIOSFODNN7EXAMPLE`},
						},
					},
				},
			},
		},
	}

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for binary files, got %d", len(result.Findings))
	}
}

func TestSecretsAnalyzer_SkipsTestFixtures(t *testing.T) {
	paths := []string{
		"pkg/analyzer/secrets_test.go",
		"tests/fixtures/secrets.go",
		"testdata/config.yaml",
		"config.example.yml",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			diff := diffWithAddedLines(path,
				`password = "SuperS3cretP@ssword!"`,
			)
			result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("expected no findings for test fixture path %q, got %d", path, len(result.Findings))
			}
		})
	}
}

func TestSecretsAnalyzer_FalsePositivePlaceholders(t *testing.T) {
	lines := []string{
		`api_key = "your-api-key-here-placeholder"`,
		`password = "changeme"`,
		`secret = "EXAMPLE_SECRET_KEY_VALUE_12345"`,
		`token = "replace_me_with_real_token_value"`,
		`dsn := "postgres://user:${DB_PASSWORD}@host/db"`,
		`key := "{{.APIKey}}"`,
		`api_key = "dummy_key_for_testing_only"`,
	}

	for _, line := range lines {
		t.Run(line, func(t *testing.T) {
			diff := diffWithAddedLines("config.go", line)
			result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("expected false positive to be skipped for %q, got %d findings", line, len(result.Findings))
			}
		})
	}
}

func TestSecretsAnalyzer_HighEntropyString(t *testing.T) {
	// A high-entropy random string that doesn't match known patterns.
	diff := diffWithAddedLines("auth.go",
		`verificationCode := "aB3$kL9mN2pQ7rS4tU6vW8xY0z1cD5eF"`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected entropy finding for high-entropy string")
	}
	found := false
	for _, f := range result.Findings {
		if f.Title == "High-entropy string detected" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a high-entropy finding")
	}
}

func TestSecretsAnalyzer_LowEntropyNotFlagged(t *testing.T) {
	// A long but low-entropy string should NOT trigger.
	diff := diffWithAddedLines("main.go",
		`msg := "aaaaaaaaaaaaaaaaaaaaaaaaa"`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, f := range result.Findings {
		if f.Title == "High-entropy string detected" {
			t.Fatal("low-entropy string should not trigger entropy check")
		}
	}
}

func TestSecretsAnalyzer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	diff := diffWithAddedLines("config.go",
		`password = "SuperS3cretP@ssword!"`,
	)

	_, err := NewSecretsAnalyzer().Analyze(ctx, diff)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestSecretsAnalyzer_MultipleFindings(t *testing.T) {
	diff := diffWithAddedLines("config.go",
		`aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYKZ6NR4TWAB`,
		`password = "SuperS3cretP@ssword!"`,
		`dsn := "postgres://admin:secret@db.internal.io:5432/production"`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) < 3 {
		t.Fatalf("expected at least 3 findings, got %d", len(result.Findings))
	}
}

func TestSecretsAnalyzer_EmptyDiff(t *testing.T) {
	diff := &interfaces.Diff{}

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for empty diff, got %d", len(result.Findings))
	}
}

func TestSecretsAnalyzer_SkipsGoSumFiles(t *testing.T) {
	diff := diffWithAddedLines("go.sum",
		`github.com/stretchr/testify v1.9.0 h1:HtqpIVDClZ4nwg75+f6Lvsy/wHu+3BoSGCbBAcpTsTg=`,
		`github.com/stretchr/testify v1.9.0/go.mod h1:r2ic/lqez/lEtzL7wO/rwa5dbSLXVDPFyf8C91i36aY=`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for go.sum, got %d", len(result.Findings))
	}
}

func TestSecretsAnalyzer_SkipsLockFiles(t *testing.T) {
	lockFiles := []string{
		"yarn.lock",
		"Cargo.lock",
		"Gemfile.lock",
		"package-lock.json",
	}

	for _, path := range lockFiles {
		t.Run(path, func(t *testing.T) {
			diff := diffWithAddedLines(path,
				`password = "SuperS3cretP@ssword!"`,
			)
			result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("expected no findings for lock file %q, got %d", path, len(result.Findings))
			}
		})
	}
}

func TestSecretsAnalyzer_SkipsDiffFiles(t *testing.T) {
	diff := diffWithAddedLines("tests/fixtures/diffs/secrets-leak.diff",
		`password = "SuperS3cretP@ssword!"`,
		`aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYKZ6NR4TWAB`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings for .diff file, got %d", len(result.Findings))
	}
}

func TestSecretsAnalyzer_EntropySkipsChecksumLines(t *testing.T) {
	checksumLines := []string{
		`h1:HtqpIVDClZ4nwg75+f6Lvsy/wHu+3BoSGCbBAcpTsTg=`,
		`sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824`,
		`sha512:cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce`,
		`sha1:a94a8fe5ccb19ba61c4c0873d391e987982fbbd3aabbccdd`,
	}

	for _, line := range checksumLines {
		t.Run(line[:6], func(t *testing.T) {
			diff := diffWithAddedLines("config.go", line)
			result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, f := range result.Findings {
				if f.Title == "High-entropy string detected" {
					t.Fatalf("checksum line should not trigger entropy check: %q", line)
				}
			}
		})
	}
}

func TestSecretsAnalyzer_ImplementsInterface(t *testing.T) {
	var _ Analyzer = NewSecretsAnalyzer()
}

func TestSecretsAnalyzer_TailwindClassNames_NotFlagged(t *testing.T) {
	tests := []struct {
		name string
		path string
		line string
	}{
		{
			"className JSX attribute",
			"components/Button.tsx",
			`<div className="flex items-center justify-between px-4 py-2 bg-gradient-to-r from-blue-500 to-purple-600 rounded-lg shadow-md">`,
		},
		{
			"class HTML attribute",
			"index.html",
			`<div class="flex items-center justify-between px-4 py-2 bg-gradient-to-r from-blue-500 to-purple-600">`,
		},
		{
			"cn() Tailwind merge utility",
			"components/Card.tsx",
			`<div className={cn("flex items-center gap-2 rounded-md border bg-popover p-4 text-popover-foreground shadow-md")}>`,
		},
		{
			"src attribute",
			"components/Avatar.tsx",
			`<img src="/images/avatars/user-profile-default-xl-2048x2048.png" />`,
		},
		{
			"href attribute",
			"components/Nav.tsx",
			`<a href="https://example.com/very/long/path/to/some/resource/that/is/high/entropy">`,
		},
		{
			"alt attribute",
			"components/Image.tsx",
			`<img alt="A beautiful landscape with mountains and rivers and trees and clouds" />`,
		},
		{
			"placeholder attribute",
			"components/Input.tsx",
			`<input placeholder="Enter your full name including middle initial and suffix" />`,
		},
		{
			"className with long Tailwind classes",
			"app/layout.tsx",
			`className="antialiased font-sans text-foreground bg-background min-h-screen flex flex-col overflow-x-hidden"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines(tt.path, tt.line)
			result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("expected no findings for Tailwind/styling line %q, got %d: %+v",
					tt.line, len(result.Findings), result.Findings)
			}
		})
	}
}

func TestSecretsAnalyzer_JSXAttributeValue_NotFlagged(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			"data attribute with long value",
			`<div data-testid="dashboard-navigation-sidebar-container-wrapper">`,
		},
		{
			"aria-label with long value",
			`<button aria-label="Close the navigation sidebar and return to main content area">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines("components/Nav.tsx", tt.line)
			result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, f := range result.Findings {
				if f.Title == "High-entropy string detected" {
					t.Fatalf("JSX attribute value should not trigger entropy: %q", tt.line)
				}
			}
		})
	}
}

func TestSecretsAnalyzer_FrontendFile_HigherEntropyThreshold(t *testing.T) {
	// A string with entropy ~5.0 should NOT flag in a .tsx file but WOULD in a .go file.
	mediumEntropyString := `tokenValue := "aB3kL9mN2pQ7rS4tU6vW8"` // entropy ~4.5-5.0

	// Should flag in .go file
	diff := diffWithAddedLines("auth.go", mediumEntropyString)
	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goFlagged := false
	for _, f := range result.Findings {
		if f.Title == "High-entropy string detected" {
			goFlagged = true
			break
		}
	}

	// Should NOT flag in .tsx file (higher threshold)
	diff = diffWithAddedLines("components/Auth.tsx", mediumEntropyString)
	result, err = NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tsxFlagged := false
	for _, f := range result.Findings {
		if f.Title == "High-entropy string detected" {
			tsxFlagged = true
			break
		}
	}

	if goFlagged && tsxFlagged {
		t.Fatal("medium-entropy string should not be flagged in .tsx but was")
	}
}

func TestSecretsAnalyzer_TextContentWithSpaces_NotFlagged(t *testing.T) {
	// Prose strings with 3+ spaces (testimonials, descriptions) should NOT
	// trigger entropy findings — real secrets never contain spaces.
	tests := []struct {
		name string
		path string
		line string
	}{
		{
			"testimonial quote in tsx",
			"components/Testimonials.tsx",
			`<p>"Our company has seen a 50% increase in productivity since implementing this solution across all departments"</p>`,
		},
		{
			"service description in tsx",
			"components/Services.tsx",
			`description="We provide comprehensive cloud infrastructure solutions for businesses looking to optimize their workflow"`,
		},
		{
			"long prose in jsx",
			"components/About.jsx",
			`const bio = "Jane is a senior software engineer with over 15 years of experience building distributed systems at scale"`,
		},
		{
			"multi-word content in html",
			"index.html",
			`<meta content="ShipSafe is a self-hosted AI code verification gateway for engineering teams who need data-sovereign quality assurance">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffWithAddedLines(tt.path, tt.line)
			result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, f := range result.Findings {
				if f.Title == "High-entropy string detected" {
					t.Fatalf("text content with spaces should not trigger entropy: %q", tt.line)
				}
			}
		})
	}
}

func TestSecretsAnalyzer_RealSecretInTSX_StillFlagged(t *testing.T) {
	// A real AWS key should still be detected in .tsx files.
	diff := diffWithAddedLines("components/Config.tsx",
		`const key = "AKIAIOSFODNN7WKRB3PQ"`,
	)

	result, err := NewSecretsAnalyzer().Analyze(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("real AWS key should still be detected in .tsx file")
	}
}

func TestShannonEntropy(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLow float64
		wantHi  float64
	}{
		{"empty string", "", 0, 0},
		{"single char", "aaaa", 0, 0.01},
		{"low entropy", "aabb", 0.9, 1.1},
		{"high entropy", "aB3$kL9mN2pQ7rS4", 3.5, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shannonEntropy(tt.input)
			if got < tt.wantLow || got > tt.wantHi {
				t.Errorf("shannonEntropy(%q) = %f, want between %f and %f", tt.input, got, tt.wantLow, tt.wantHi)
			}
		})
	}
}

// --- test helpers ---

func assertHasFindingWithTitle(t *testing.T, findings []interfaces.Finding, substr string) {
	t.Helper()
	for _, f := range findings {
		if contains(f.Title, substr) {
			return
		}
	}
	titles := make([]string, len(findings))
	for i, f := range findings {
		titles[i] = f.Title
	}
	t.Errorf("no finding with title containing %q, got: %v", substr, titles)
}

func assertHasSeverity(t *testing.T, findings []interfaces.Finding, severity interfaces.Severity) {
	t.Helper()
	for _, f := range findings {
		if f.Severity == severity {
			return
		}
	}
	t.Errorf("no finding with severity %q", severity)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
