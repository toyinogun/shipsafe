// Package vcs provides version control integration for ShipSafe.
// It handles diff parsing, git operations, and VCS platform API clients.
package vcs

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

var (
	ErrEmptyDiff   = errors.New("vcs: empty diff input")
	ErrInvalidDiff = errors.New("vcs: invalid diff format")
)

var (
	diffHeaderRegex = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
	binaryFileRegex = regexp.MustCompile(`^Binary files .+ and .+ differ$`)
)

// extToLanguage maps file extensions to language identifiers.
var extToLanguage = map[string]string{
	".go":     "go",
	".py":     "python",
	".js":     "javascript",
	".ts":     "typescript",
	".tsx":    "typescriptreact",
	".jsx":    "javascriptreact",
	".java":   "java",
	".rs":     "rust",
	".rb":     "ruby",
	".c":      "c",
	".cpp":    "cpp",
	".cc":     "cpp",
	".h":      "c",
	".hpp":    "cpp",
	".cs":     "csharp",
	".swift":  "swift",
	".kt":     "kotlin",
	".php":    "php",
	".sh":     "shell",
	".bash":   "shell",
	".zsh":    "shell",
	".yml":    "yaml",
	".yaml":   "yaml",
	".json":   "json",
	".xml":    "xml",
	".html":   "html",
	".css":    "css",
	".scss":   "scss",
	".sql":    "sql",
	".md":     "markdown",
	".toml":   "toml",
	".ini":    "ini",
	".tf":     "terraform",
	".proto":  "protobuf",
	".lua":    "lua",
	".ex":     "elixir",
	".exs":    "elixir",
	".erl":    "erlang",
	".hs":     "haskell",
	".scala":  "scala",
	".pl":     "perl",
	".pm":     "perl",
	".r":      "r",
	".R":      "r",
}

// nameToLanguage maps special filenames to language identifiers.
var nameToLanguage = map[string]string{
	"Dockerfile":   "dockerfile",
	"Makefile":     "makefile",
	"Jenkinsfile":  "groovy",
	"Vagrantfile":  "ruby",
	"Gemfile":      "ruby",
	"Rakefile":     "ruby",
	".gitignore":   "gitignore",
	".dockerignore": "dockerignore",
}

// detectLanguage returns the language name for a file path based on extension or filename.
func detectLanguage(path string) string {
	base := filepath.Base(path)
	if lang, ok := nameToLanguage[base]; ok {
		return lang
	}
	ext := filepath.Ext(path)
	if lang, ok := extToLanguage[ext]; ok {
		return lang
	}
	return ""
}

// diffParser implements the interfaces.DiffParser interface.
type diffParser struct{}

// NewDiffParser creates a new DiffParser that handles unified diff format.
func NewDiffParser() interfaces.DiffParser {
	return &diffParser{}
}

func (p *diffParser) Parse(ctx context.Context, raw []byte) (*interfaces.Diff, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, ErrEmptyDiff
	}

	diff := &interfaces.Diff{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	var current *fileState
	var files []interfaces.FileDiff

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("vcs: parsing cancelled: %w", ctx.Err())
		default:
		}

		line := scanner.Text()

		// Start of a new file diff
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			if current != nil {
				files = append(files, current.toFileDiff())
			}
			current = &fileState{
				gitOldPath: matches[1],
				gitNewPath: matches[2],
			}
			continue
		}

		if current == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "--- "):
			current.minusHeader = strings.TrimPrefix(line, "--- ")
		case strings.HasPrefix(line, "+++ "):
			current.plusHeader = strings.TrimPrefix(line, "+++ ")
		case strings.HasPrefix(line, "rename from "):
			current.renameFrom = strings.TrimPrefix(line, "rename from ")
		case strings.HasPrefix(line, "rename to "):
			current.renameTo = strings.TrimPrefix(line, "rename to ")
		case strings.HasPrefix(line, "new file mode"):
			current.newFile = true
		case strings.HasPrefix(line, "deleted file mode"):
			current.deletedFile = true
		case binaryFileRegex.MatchString(line):
			current.binary = true
		case hunkHeaderRegex.MatchString(line):
			current.beginHunk(line)
		case strings.HasPrefix(line, "\\"):
			// "\ No newline at end of file" — skip
		case current.hasActiveHunk():
			current.appendLine(line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("vcs: reading diff: %w", err)
	}

	if current != nil {
		files = append(files, current.toFileDiff())
	}

	if len(files) == 0 {
		return nil, ErrInvalidDiff
	}

	diff.Files = files
	return diff, nil
}

func (p *diffParser) ParseFile(ctx context.Context, path string) (*interfaces.Diff, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("vcs: reading diff file %s: %w", path, err)
	}
	return p.Parse(ctx, data)
}

// fileState holds mutable state while parsing a single file's diff.
type fileState struct {
	gitOldPath  string
	gitNewPath  string
	minusHeader string
	plusHeader   string
	renameFrom  string
	renameTo    string
	newFile     bool
	deletedFile bool
	binary      bool
	hunks       []hunkState
}

// hunkState holds mutable state while parsing a single hunk.
type hunkState struct {
	oldStart int
	oldLines int
	newStart int
	newLines int
	lines    []string
}

func (f *fileState) beginHunk(line string) {
	matches := hunkHeaderRegex.FindStringSubmatch(line)
	if matches == nil {
		return
	}

	hs := hunkState{}
	hs.oldStart, _ = strconv.Atoi(matches[1])
	if matches[2] != "" {
		hs.oldLines, _ = strconv.Atoi(matches[2])
	} else {
		hs.oldLines = 1
	}
	hs.newStart, _ = strconv.Atoi(matches[3])
	if matches[4] != "" {
		hs.newLines, _ = strconv.Atoi(matches[4])
	} else {
		hs.newLines = 1
	}

	f.hunks = append(f.hunks, hs)
}

func (f *fileState) hasActiveHunk() bool {
	return len(f.hunks) > 0
}

func (f *fileState) appendLine(line string) {
	if len(f.hunks) == 0 {
		return
	}
	f.hunks[len(f.hunks)-1].lines = append(f.hunks[len(f.hunks)-1].lines, line)
}

func (f *fileState) status() interfaces.FileStatus {
	if f.renameFrom != "" || f.renameTo != "" {
		return interfaces.FileRenamed
	}
	if f.newFile || f.minusHeader == "/dev/null" {
		return interfaces.FileAdded
	}
	if f.deletedFile || f.plusHeader == "/dev/null" {
		return interfaces.FileDeleted
	}
	return interfaces.FileModified
}

func (f *fileState) toFileDiff() interfaces.FileDiff {
	st := f.status()
	fd := interfaces.FileDiff{
		Status:   st,
		IsBinary: f.binary,
	}

	switch st {
	case interfaces.FileAdded:
		fd.Path = f.gitNewPath
	case interfaces.FileDeleted:
		fd.Path = f.gitOldPath
	case interfaces.FileRenamed:
		if f.renameTo != "" {
			fd.Path = f.renameTo
			fd.OldPath = f.renameFrom
		} else {
			fd.Path = f.gitNewPath
			fd.OldPath = f.gitOldPath
		}
	default:
		fd.Path = f.gitNewPath
	}

	fd.Language = detectLanguage(fd.Path)

	for _, hs := range f.hunks {
		fd.Hunks = append(fd.Hunks, hs.toHunk())
	}

	return fd
}

func (hs *hunkState) toHunk() interfaces.Hunk {
	hunk := interfaces.Hunk{
		OldStart: hs.oldStart,
		OldLines: hs.oldLines,
		NewStart: hs.newStart,
		NewLines: hs.newLines,
		Content:  strings.Join(hs.lines, "\n"),
	}

	oldLine := hs.oldStart
	newLine := hs.newStart

	for _, line := range hs.lines {
		if len(line) == 0 {
			// Blank context line (some git versions omit the leading space)
			oldLine++
			newLine++
			continue
		}

		switch line[0] {
		case '+':
			hunk.AddedLines = append(hunk.AddedLines, interfaces.Line{
				Number:  newLine,
				Content: line[1:],
			})
			newLine++
		case '-':
			hunk.RemovedLines = append(hunk.RemovedLines, interfaces.Line{
				Number:  oldLine,
				Content: line[1:],
			})
			oldLine++
		default: // context line (starts with space)
			oldLine++
			newLine++
		}
	}

	return hunk
}
