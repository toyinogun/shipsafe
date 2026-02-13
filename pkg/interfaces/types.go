// Package interfaces defines the shared types and contracts for all ShipSafe modules.
// This package has ZERO dependencies on any other pkg/ package.
// All cross-module communication goes through types and interfaces defined here.
package interfaces

import "time"

// Severity levels for findings
type Severity string

const (
	SeverityCritical Severity = "critical" // Must fix before merge
	SeverityHigh     Severity = "high"     // Strongly recommended fix
	SeverityMedium   Severity = "medium"   // Should fix
	SeverityLow      Severity = "low"      // Informational
	SeverityInfo     Severity = "info"     // FYI only
)

// Category classifies what type of issue was found
type Category string

const (
	CategoryComplexity Category = "complexity"
	CategoryCoverage   Category = "coverage"
	CategorySecurity   Category = "security"
	CategorySecrets    Category = "secrets"
	CategoryPattern    Category = "pattern"
	CategoryImport     Category = "import"
	CategoryLogic      Category = "logic"
	CategoryConvention Category = "convention"
)

// Finding represents a single issue found during analysis.
type Finding struct {
	ID          string         `json:"id"`
	Category    Category       `json:"category"`
	Severity    Severity       `json:"severity"`
	File        string         `json:"file"`
	StartLine   int            `json:"start_line"`
	EndLine     int            `json:"end_line"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Suggestion  string         `json:"suggestion,omitempty"`
	Source      string         `json:"source"`
	Confidence  float64        `json:"confidence"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Rating represents the overall trust rating.
type Rating string

const (
	RatingGreen  Rating = "GREEN"  // Safe to merge
	RatingYellow Rating = "YELLOW" // Review recommended
	RatingRed    Rating = "RED"    // Do not merge
)

// TrustScore is the final calculated trust score.
type TrustScore struct {
	Score        int              `json:"score"`         // 0-100
	Rating       Rating           `json:"rating"`        // GREEN/YELLOW/RED
	Breakdown    map[Category]int `json:"breakdown"`     // Per-category scores
	FindingCount map[Severity]int `json:"finding_count"` // Findings by severity
}

// Diff represents a parsed code diff.
type Diff struct {
	BaseSHA string     `json:"base_sha"`
	HeadSHA string     `json:"head_sha"`
	Files   []FileDiff `json:"files"`
	PRTitle string     `json:"pr_title,omitempty"`
	PRBody  string     `json:"pr_body,omitempty"`
	Author  string     `json:"author,omitempty"`
}

// FileStatus describes how a file was changed.
type FileStatus string

const (
	FileAdded    FileStatus = "added"
	FileModified FileStatus = "modified"
	FileDeleted  FileStatus = "deleted"
	FileRenamed  FileStatus = "renamed"
)

// FileDiff represents changes to a single file.
type FileDiff struct {
	Path     string     `json:"path"`
	OldPath  string     `json:"old_path,omitempty"`
	Status   FileStatus `json:"status"`
	Hunks    []Hunk     `json:"hunks"`
	Language string     `json:"language"`
	IsBinary bool       `json:"is_binary"`
}

// Hunk represents a contiguous block of changes within a file.
type Hunk struct {
	OldStart     int    `json:"old_start"`
	OldLines     int    `json:"old_lines"`
	NewStart     int    `json:"new_start"`
	NewLines     int    `json:"new_lines"`
	Content      string `json:"content"`
	AddedLines   []Line `json:"added_lines"`
	RemovedLines []Line `json:"removed_lines"`
}

// Line represents a single line of code with its line number.
type Line struct {
	Number  int    `json:"number"`
	Content string `json:"content"`
}

// AnalysisResult is what each analyzer returns.
type AnalysisResult struct {
	AnalyzerName string         `json:"analyzer_name"`
	Findings     []Finding      `json:"findings"`
	Duration     time.Duration  `json:"duration"`
	Error        error          `json:"-"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// DiffMetadata summarises the scope of a diff.
type DiffMetadata struct {
	FilesChanged int    `json:"files_changed"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	BaseSHA      string `json:"base_sha"`
	HeadSHA      string `json:"head_sha"`
}

// Report is the final output of a ShipSafe analysis run.
type Report struct {
	ID         string         `json:"id"`
	Timestamp  time.Time      `json:"timestamp"`
	TrustScore TrustScore     `json:"trust_score"`
	Findings   []Finding      `json:"findings"`
	Summary    string         `json:"summary"`
	DiffMeta   DiffMetadata   `json:"diff_metadata"`
	Duration   time.Duration  `json:"duration"`
	Config     map[string]any `json:"config,omitempty"`
}

// AIReviewOptions configures the AI review pass.
type AIReviewOptions struct {
	ContextFiles []string `json:"context_files,omitempty"`
	FocusAreas   []string `json:"focus_areas,omitempty"`
	MaxTokens    int      `json:"max_tokens,omitempty"`
}

// StatusState represents a VCS commit status.
type StatusState string

const (
	StatusPending StatusState = "pending"
	StatusSuccess StatusState = "success"
	StatusFailure StatusState = "failure"
	StatusError   StatusState = "error"
)
