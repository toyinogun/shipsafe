package vcs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

func TestDiffParser_Parse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, diff *interfaces.Diff)
	}{
		{
			name: "single file modification",
			input: `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -10,7 +10,8 @@ func main() {
 	fmt.Println("hello")
-	fmt.Println("old")
+	fmt.Println("new")
+	fmt.Println("extra")
 	fmt.Println("world")
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				if len(diff.Files) != 1 {
					t.Fatalf("expected 1 file, got %d", len(diff.Files))
				}
				f := diff.Files[0]
				assertEqual(t, "path", "main.go", f.Path)
				assertEqual(t, "status", interfaces.FileModified, f.Status)
				assertEqual(t, "language", "go", f.Language)
				assertFalse(t, "binary", f.IsBinary)

				if len(f.Hunks) != 1 {
					t.Fatalf("expected 1 hunk, got %d", len(f.Hunks))
				}
				h := f.Hunks[0]
				assertIntEqual(t, "old_start", 10, h.OldStart)
				assertIntEqual(t, "old_lines", 7, h.OldLines)
				assertIntEqual(t, "new_start", 10, h.NewStart)
				assertIntEqual(t, "new_lines", 8, h.NewLines)
				assertIntEqual(t, "added_count", 2, len(h.AddedLines))
				assertIntEqual(t, "removed_count", 1, len(h.RemovedLines))

				// Verify line numbers
				assertIntEqual(t, "removed_line_number", 11, h.RemovedLines[0].Number)
				assertEqual(t, "removed_content", "\tfmt.Println(\"old\")", h.RemovedLines[0].Content)
				assertIntEqual(t, "added_line_1_number", 11, h.AddedLines[0].Number)
				assertIntEqual(t, "added_line_2_number", 12, h.AddedLines[1].Number)
			},
		},
		{
			name: "multi-file diff",
			input: `diff --git a/foo.go b/foo.go
index 1111111..2222222 100644
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,4 @@
 package main
+// added comment

 func foo() {}
diff --git a/bar.py b/bar.py
index 3333333..4444444 100644
--- a/bar.py
+++ b/bar.py
@@ -5,6 +5,7 @@ def bar():
     pass

+# new line
 def baz():
     pass
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				if len(diff.Files) != 2 {
					t.Fatalf("expected 2 files, got %d", len(diff.Files))
				}
				assertEqual(t, "file1_path", "foo.go", diff.Files[0].Path)
				assertEqual(t, "file1_lang", "go", diff.Files[0].Language)
				assertEqual(t, "file2_path", "bar.py", diff.Files[1].Path)
				assertEqual(t, "file2_lang", "python", diff.Files[1].Language)

				assertIntEqual(t, "file1_added", 1, len(diff.Files[0].Hunks[0].AddedLines))
				assertIntEqual(t, "file2_added", 1, len(diff.Files[1].Hunks[0].AddedLines))
			},
		},
		{
			name: "new file",
			input: `diff --git a/new_file.go b/new_file.go
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/new_file.go
@@ -0,0 +1,5 @@
+package main
+
+func hello() {
+	fmt.Println("hi")
+}
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				if len(diff.Files) != 1 {
					t.Fatalf("expected 1 file, got %d", len(diff.Files))
				}
				f := diff.Files[0]
				assertEqual(t, "path", "new_file.go", f.Path)
				assertEqual(t, "status", interfaces.FileAdded, f.Status)
				assertIntEqual(t, "hunks", 1, len(f.Hunks))

				h := f.Hunks[0]
				assertIntEqual(t, "old_start", 0, h.OldStart)
				assertIntEqual(t, "old_lines", 0, h.OldLines)
				assertIntEqual(t, "new_start", 1, h.NewStart)
				assertIntEqual(t, "new_lines", 5, h.NewLines)
				assertIntEqual(t, "added_count", 5, len(h.AddedLines))
				assertIntEqual(t, "removed_count", 0, len(h.RemovedLines))

				// First added line should be at line 1
				assertIntEqual(t, "first_line_number", 1, h.AddedLines[0].Number)
				assertEqual(t, "first_line_content", "package main", h.AddedLines[0].Content)
			},
		},
		{
			name: "deleted file",
			input: `diff --git a/old_file.go b/old_file.go
deleted file mode 100644
index abc1234..0000000
--- a/old_file.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package main
-
-func old() {}
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				if len(diff.Files) != 1 {
					t.Fatalf("expected 1 file, got %d", len(diff.Files))
				}
				f := diff.Files[0]
				assertEqual(t, "path", "old_file.go", f.Path)
				assertEqual(t, "status", interfaces.FileDeleted, f.Status)

				h := f.Hunks[0]
				assertIntEqual(t, "added_count", 0, len(h.AddedLines))
				assertIntEqual(t, "removed_count", 3, len(h.RemovedLines))
				assertIntEqual(t, "first_removed_number", 1, h.RemovedLines[0].Number)
			},
		},
		{
			name: "renamed file",
			input: `diff --git a/old_name.go b/new_name.go
similarity index 100%
rename from old_name.go
rename to new_name.go
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				if len(diff.Files) != 1 {
					t.Fatalf("expected 1 file, got %d", len(diff.Files))
				}
				f := diff.Files[0]
				assertEqual(t, "path", "new_name.go", f.Path)
				assertEqual(t, "old_path", "old_name.go", f.OldPath)
				assertEqual(t, "status", interfaces.FileRenamed, f.Status)
				assertIntEqual(t, "hunks", 0, len(f.Hunks))
			},
		},
		{
			name: "renamed file with changes",
			input: `diff --git a/utils/helper.go b/pkg/helper.go
similarity index 85%
rename from utils/helper.go
rename to pkg/helper.go
--- a/utils/helper.go
+++ b/pkg/helper.go
@@ -1,4 +1,4 @@
-package utils
+package pkg

 func Helper() string {
 	return "help"
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				f := diff.Files[0]
				assertEqual(t, "path", "pkg/helper.go", f.Path)
				assertEqual(t, "old_path", "utils/helper.go", f.OldPath)
				assertEqual(t, "status", interfaces.FileRenamed, f.Status)
				assertIntEqual(t, "hunks", 1, len(f.Hunks))
				assertIntEqual(t, "added", 1, len(f.Hunks[0].AddedLines))
				assertIntEqual(t, "removed", 1, len(f.Hunks[0].RemovedLines))
			},
		},
		{
			name: "binary file",
			input: `diff --git a/image.png b/image.png
index abc1234..def5678 100644
Binary files a/image.png and b/image.png differ
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				if len(diff.Files) != 1 {
					t.Fatalf("expected 1 file, got %d", len(diff.Files))
				}
				f := diff.Files[0]
				assertEqual(t, "path", "image.png", f.Path)
				assertTrue(t, "binary", f.IsBinary)
				assertIntEqual(t, "hunks", 0, len(f.Hunks))
			},
		},
		{
			name: "new binary file",
			input: `diff --git a/logo.png b/logo.png
new file mode 100644
index 0000000..abc1234
Binary files /dev/null and b/logo.png differ
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				f := diff.Files[0]
				assertEqual(t, "path", "logo.png", f.Path)
				assertEqual(t, "status", interfaces.FileAdded, f.Status)
				assertTrue(t, "binary", f.IsBinary)
			},
		},
		{
			name: "multiple hunks in one file",
			input: `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -2,6 +2,7 @@ package main

 import "fmt"

+// first change
 func main() {
 	fmt.Println("hello")
 }
@@ -20,6 +21,7 @@ func other() {
 	x := 1
 	y := 2

+	// second change
 	fmt.Println(x + y)
 }
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				f := diff.Files[0]
				assertIntEqual(t, "hunks", 2, len(f.Hunks))

				assertIntEqual(t, "hunk1_old_start", 2, f.Hunks[0].OldStart)
				assertIntEqual(t, "hunk1_new_start", 2, f.Hunks[0].NewStart)
				assertIntEqual(t, "hunk1_added", 1, len(f.Hunks[0].AddedLines))
				assertIntEqual(t, "hunk1_added_number", 5, f.Hunks[0].AddedLines[0].Number)

				assertIntEqual(t, "hunk2_old_start", 20, f.Hunks[1].OldStart)
				assertIntEqual(t, "hunk2_new_start", 21, f.Hunks[1].NewStart)
				assertIntEqual(t, "hunk2_added", 1, len(f.Hunks[1].AddedLines))
				assertIntEqual(t, "hunk2_added_number", 24, f.Hunks[1].AddedLines[0].Number)
			},
		},
		{
			name: "no newline at end of file marker",
			input: `diff --git a/file.go b/file.go
index abc1234..def5678 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,3 @@
 package main

-var x = 1
\ No newline at end of file
+var x = 2
\ No newline at end of file
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				f := diff.Files[0]
				h := f.Hunks[0]
				assertIntEqual(t, "added", 1, len(h.AddedLines))
				assertIntEqual(t, "removed", 1, len(h.RemovedLines))
				assertEqual(t, "added_content", "var x = 2", h.AddedLines[0].Content)
				assertEqual(t, "removed_content", "var x = 1", h.RemovedLines[0].Content)
			},
		},
		{
			name: "single line hunk header without line count",
			input: `diff --git a/file.txt b/file.txt
index abc1234..def5678 100644
--- a/file.txt
+++ b/file.txt
@@ -1 +1 @@
-old
+new
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				h := diff.Files[0].Hunks[0]
				assertIntEqual(t, "old_lines", 1, h.OldLines)
				assertIntEqual(t, "new_lines", 1, h.NewLines)
				assertIntEqual(t, "added", 1, len(h.AddedLines))
				assertIntEqual(t, "removed", 1, len(h.RemovedLines))
			},
		},
		{
			name: "mixed file types",
			input: `diff --git a/Dockerfile b/Dockerfile
index 1111111..2222222 100644
--- a/Dockerfile
+++ b/Dockerfile
@@ -1,2 +1,3 @@
 FROM golang:1.23
+RUN apt-get update
 CMD ["./app"]
diff --git a/config.yaml b/config.yaml
index 3333333..4444444 100644
--- a/config.yaml
+++ b/config.yaml
@@ -1,2 +1,3 @@
 key: value
+new_key: new_value
 other: data
`,
			check: func(t *testing.T, diff *interfaces.Diff) {
				assertIntEqual(t, "files", 2, len(diff.Files))
				assertEqual(t, "file1_lang", "dockerfile", diff.Files[0].Language)
				assertEqual(t, "file2_lang", "yaml", diff.Files[1].Language)
			},
		},
	}

	parser := NewDiffParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := parser.Parse(context.Background(), []byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, diff)
		})
	}
}

func TestDiffParser_Parse_Errors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "empty input",
			input:   "",
			wantErr: ErrEmptyDiff,
		},
		{
			name:    "whitespace only",
			input:   "   \n\t\n  ",
			wantErr: ErrEmptyDiff,
		},
		{
			name:    "no diff headers",
			input:   "this is not a diff\njust some random text\n",
			wantErr: ErrInvalidDiff,
		},
	}

	parser := NewDiffParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(context.Background(), []byte(tt.input))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != nil && err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestDiffParser_Parse_ContextCancellation(t *testing.T) {
	input := `diff --git a/file.go b/file.go
index abc1234..def5678 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,3 @@
 package main
-var x = 1
+var x = 2
`

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	parser := NewDiffParser()
	_, err := parser.Parse(ctx, []byte(input))
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

func TestDiffParser_ParseFile(t *testing.T) {
	diffContent := `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main

+// new comment
 func main() {}
`

	t.Run("valid file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.diff")
		if err := os.WriteFile(path, []byte(diffContent), 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		parser := NewDiffParser()
		diff, err := parser.ParseFile(context.Background(), path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diff.Files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(diff.Files))
		}
		assertEqual(t, "path", "main.go", diff.Files[0].Path)
	})

	t.Run("nonexistent file", func(t *testing.T) {
		parser := NewDiffParser()
		_, err := parser.ParseFile(context.Background(), "/nonexistent/path/file.diff")
		if err == nil {
			t.Fatal("expected error for nonexistent file, got nil")
		}
	})
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"src/app.py", "python"},
		{"index.js", "javascript"},
		{"component.tsx", "typescriptreact"},
		{"lib/utils.rs", "rust"},
		{"Dockerfile", "dockerfile"},
		{"Makefile", "makefile"},
		{"deploy/config.yaml", "yaml"},
		{"deploy/config.yml", "yaml"},
		{".gitignore", "gitignore"},
		{"data.json", "json"},
		{"style.css", "css"},
		{"query.sql", "sql"},
		{"infra/main.tf", "terraform"},
		{"unknown.xyz", ""},
		{"no_extension", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := detectLanguage(tt.path)
			if got != tt.expected {
				t.Errorf("detectLanguage(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

// Test helpers

func assertEqual[T comparable](t *testing.T, field string, want, got T) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", field, got, want)
	}
}

func assertIntEqual(t *testing.T, field string, want, got int) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %d, want %d", field, got, want)
	}
}

func assertTrue(t *testing.T, field string, got bool) {
	t.Helper()
	if !got {
		t.Errorf("%s: expected true, got false", field)
	}
}

func assertFalse(t *testing.T, field string, got bool) {
	t.Helper()
	if got {
		t.Errorf("%s: expected false, got true", field)
	}
}
