package interfaces

import "context"

// VCSProvider abstracts git platform operations (GitHub, Forgejo, GitLab).
// It handles fetching diffs, posting PR comments, and setting commit statuses.
type VCSProvider interface {
	// GetDiff retrieves the diff for a PR/MR by its reference (e.g., PR number, MR IID).
	GetDiff(ctx context.Context, prRef string) (*Diff, error)

	// PostComment posts the report as a PR/MR comment.
	PostComment(ctx context.Context, prRef string, body string) error

	// SetStatus sets the commit status check on a given SHA.
	SetStatus(ctx context.Context, sha string, status StatusState, description string) error
}

// DiffParser parses raw diff content into structured Diff objects.
// Used for both file-based diffs and piped input.
type DiffParser interface {
	// Parse converts raw unified diff bytes into a structured Diff.
	Parse(ctx context.Context, raw []byte) (*Diff, error)

	// ParseFile reads a diff file from disk and parses it.
	ParseFile(ctx context.Context, path string) (*Diff, error)
}

// Pipeline orchestrates the full analysis workflow.
// It coordinates the VCS layer, analyzers, AI reviewer, scorer, and reporter.
type Pipeline interface {
	// Run executes the full analysis pipeline and returns a report.
	Run(ctx context.Context, diff *Diff) (*Report, error)
}
