package ai

import (
	"fmt"
	"sort"
	"strings"

	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

const (
	// DefaultMaxTokenBudget is the default maximum token budget for context (estimated 4 chars per token).
	DefaultMaxTokenBudget = 4000
	charsPerToken         = 4
)

// securitySensitivePatterns are file path substrings that indicate security-sensitive files.
var securitySensitivePatterns = []string{
	"auth", "login", "password", "secret", "token", "crypt",
	"security", "permission", "session", "credential", "key",
	"oauth", "jwt", "cert", "ssl", "tls",
}

// BuildContext creates a concise context string from a Diff for LLM consumption.
// It respects the given maxTokenBudget (in tokens, estimated at 4 chars/token).
// If the diff is too large, security-sensitive files are prioritized.
func BuildContext(diff *interfaces.Diff, maxTokenBudget int) string {
	if maxTokenBudget <= 0 {
		maxTokenBudget = DefaultMaxTokenBudget
	}
	maxChars := maxTokenBudget * charsPerToken

	var b strings.Builder

	// Header: summary of the diff.
	if diff.PRTitle != "" {
		fmt.Fprintf(&b, "PR: %s\n", diff.PRTitle)
	}
	if diff.PRBody != "" {
		body := diff.PRBody
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		fmt.Fprintf(&b, "Description: %s\n", body)
	}

	// Summarise files and languages.
	languages := map[string]bool{}
	for _, f := range diff.Files {
		if f.Language != "" {
			languages[f.Language] = true
		}
	}
	if len(languages) > 0 {
		langs := make([]string, 0, len(languages))
		for l := range languages {
			langs = append(langs, l)
		}
		sort.Strings(langs)
		fmt.Fprintf(&b, "Languages: %s\n", strings.Join(langs, ", "))
	}
	fmt.Fprintf(&b, "Files changed: %d\n\n", len(diff.Files))

	// Sort files: security-sensitive first, then by total hunk content size (descending).
	files := make([]interfaces.FileDiff, len(diff.Files))
	copy(files, diff.Files)
	sort.SliceStable(files, func(i, j int) bool {
		iSec := isSecuritySensitive(files[i].Path)
		jSec := isSecuritySensitive(files[j].Path)
		if iSec != jSec {
			return iSec
		}
		return hunkSize(files[i]) > hunkSize(files[j])
	})

	// Build per-file context within budget.
	for _, f := range files {
		if f.IsBinary {
			continue
		}
		if b.Len() >= maxChars {
			fmt.Fprintf(&b, "\n... (remaining files truncated to fit token budget)\n")
			break
		}

		fileCtx := buildFileContext(f)
		if b.Len()+len(fileCtx) > maxChars {
			// Include a truncated version of this file.
			remaining := maxChars - b.Len()
			if remaining > 100 {
				b.WriteString(fileCtx[:remaining])
				b.WriteString("\n... (file truncated)\n")
			}
			break
		}
		b.WriteString(fileCtx)
	}

	return b.String()
}

// buildFileContext formats a single file's changes for LLM context.
func buildFileContext(f interfaces.FileDiff) string {
	var b strings.Builder

	fmt.Fprintf(&b, "--- File: %s [%s]", f.Path, f.Status)
	if f.Language != "" {
		fmt.Fprintf(&b, " (%s)", f.Language)
	}
	b.WriteString("\n")

	additions := 0
	deletions := 0
	for _, h := range f.Hunks {
		additions += len(h.AddedLines)
		deletions += len(h.RemovedLines)
	}
	fmt.Fprintf(&b, "+%d -%d lines\n", additions, deletions)

	for _, h := range f.Hunks {
		if h.Content != "" {
			fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", h.OldStart, h.OldLines, h.NewStart, h.NewLines)
			b.WriteString(h.Content)
			if !strings.HasSuffix(h.Content, "\n") {
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")
	return b.String()
}

// isSecuritySensitive checks if a file path suggests security-sensitive content.
func isSecuritySensitive(path string) bool {
	lower := strings.ToLower(path)
	for _, pat := range securitySensitivePatterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	return false
}

// hunkSize returns the total content length across all hunks in a file.
func hunkSize(f interfaces.FileDiff) int {
	total := 0
	for _, h := range f.Hunks {
		total += len(h.Content)
	}
	return total
}
