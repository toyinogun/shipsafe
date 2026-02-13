package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/toyinlola/shipsafe/pkg/ai/prompts"
	"github.com/toyinlola/shipsafe/pkg/interfaces"
)

// Reviewer implements interfaces.AIReviewer using an LLM provider.
type Reviewer struct {
	provider       LLMProvider
	maxTokenBudget int
}

// Option configures a Reviewer.
type Option func(*Reviewer)

// WithMaxTokenBudget sets the maximum token budget for diff context sent to the LLM.
func WithMaxTokenBudget(budget int) Option {
	return func(r *Reviewer) {
		r.maxTokenBudget = budget
	}
}

// NewReviewer creates an AI reviewer backed by the given LLM provider.
func NewReviewer(provider LLMProvider, opts ...Option) *Reviewer {
	r := &Reviewer{
		provider:       provider,
		maxTokenBudget: DefaultMaxTokenBudget,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Review performs AI-powered analysis of the diff using three review passes:
// semantic, logic, and convention. Reviews run sequentially to respect rate limits.
func (r *Reviewer) Review(ctx context.Context, diff *interfaces.Diff, opts *interfaces.AIReviewOptions) (*interfaces.AnalysisResult, error) {
	start := time.Now()

	budget := r.maxTokenBudget
	if opts != nil && opts.MaxTokens > 0 {
		budget = opts.MaxTokens
	}
	diffContext := BuildContext(diff, budget)

	var allFindings []interfaces.Finding

	// Run review passes sequentially to respect rate limits.
	type reviewPass struct {
		name         string
		category     interfaces.Category
		systemPrompt string
		userPrompt   string
	}

	passes := []reviewPass{
		{
			name:         "semantic",
			category:     interfaces.CategoryLogic,
			systemPrompt: prompts.SemanticSystemPrompt(),
			userPrompt:   prompts.SemanticPrompt(diffContext),
		},
		{
			name:         "logic",
			category:     interfaces.CategoryLogic,
			systemPrompt: prompts.LogicSystemPrompt(),
			userPrompt:   prompts.LogicPrompt(diffContext),
		},
		{
			name:         "convention",
			category:     interfaces.CategoryConvention,
			systemPrompt: prompts.ConventionSystemPrompt(),
			userPrompt:   prompts.ConventionPrompt(diffContext),
		},
	}

	for _, pass := range passes {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		slog.Info("running AI review pass", "pass", pass.name)

		response, err := r.provider.Complete(ctx, pass.userPrompt, CompletionOpts{
			MaxTokens:    2048,
			Temperature:  0.1,
			SystemPrompt: pass.systemPrompt,
		})
		if err != nil {
			slog.Warn("AI review pass failed", "pass", pass.name, "error", err)
			continue
		}

		findings, confidence := parseFindings(response, pass.category)
		if confidence < 0.3 {
			slog.Warn("AI response poorly structured, skipping findings", "pass", pass.name, "confidence", confidence)
			continue
		}

		slog.Info("AI review pass complete", "pass", pass.name, "findings", len(findings), "confidence", confidence)
		allFindings = append(allFindings, findings...)
	}

	deduped := deduplicateFindings(allFindings)
	if removed := len(allFindings) - len(deduped); removed > 0 {
		slog.Info("deduplicated AI findings", "before", len(allFindings), "after", len(deduped), "removed", removed)
	}

	return &interfaces.AnalysisResult{
		AnalyzerName: "ai-reviewer",
		Findings:     deduped,
		Duration:     time.Since(start),
	}, nil
}

// Available returns true if the LLM provider is configured and reachable.
func (r *Reviewer) Available(ctx context.Context) bool {
	return r.provider.Available(ctx)
}

// llmFinding is the expected JSON structure from the LLM response.
type llmFinding struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// llmResponse is the expected top-level JSON structure from the LLM.
type llmResponse struct {
	Findings []llmFinding `json:"findings"`
}

// parseFindings extracts structured findings from the LLM response text.
// Returns the findings and a confidence score (0.0-1.0) indicating how well-structured the response was.
func parseFindings(response string, category interfaces.Category) ([]interfaces.Finding, float64) {
	response = strings.TrimSpace(response)

	// Strip markdown code fences if present.
	response = stripCodeFences(response)

	var llmResp llmResponse
	if err := json.Unmarshal([]byte(response), &llmResp); err != nil {
		slog.Debug("ai: failed to parse LLM response as JSON", "error", err, "response_prefix", truncateStr(response, 200))
		return nil, 0.0
	}

	if len(llmResp.Findings) == 0 {
		return nil, 1.0 // Valid JSON, no findings — high confidence.
	}

	findings := make([]interfaces.Finding, 0, len(llmResp.Findings))
	validCount := 0

	for i, lf := range llmResp.Findings {
		if lf.Title == "" || lf.Description == "" {
			continue
		}

		sev := mapSeverity(lf.Severity)
		findings = append(findings, interfaces.Finding{
			ID:          fmt.Sprintf("ai-%s-%d", string(category), i),
			Category:    category,
			Severity:    sev,
			File:        lf.File,
			StartLine:   lf.Line,
			EndLine:     lf.Line,
			Title:       lf.Title,
			Description: lf.Description,
			Suggestion:  lf.Suggestion,
			Source:       "ai-reviewer",
			Confidence:  0.7, // AI findings get moderate confidence by default.
		})
		validCount++
	}

	// Confidence based on ratio of valid findings to total.
	confidence := 1.0
	if len(llmResp.Findings) > 0 {
		confidence = float64(validCount) / float64(len(llmResp.Findings))
	}

	return findings, confidence
}

// mapSeverity converts a severity string from the LLM to the interfaces.Severity type.
func mapSeverity(s string) interfaces.Severity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical":
		return interfaces.SeverityCritical
	case "high":
		return interfaces.SeverityHigh
	case "medium":
		return interfaces.SeverityMedium
	case "low":
		return interfaces.SeverityLow
	case "info":
		return interfaces.SeverityInfo
	default:
		return interfaces.SeverityMedium
	}
}

// stripCodeFences removes markdown code fences (```json ... ```) from the response.
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence (```json or ```)
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
	}
	if strings.HasSuffix(s, "```") {
		s = s[:len(s)-3]
	}
	return strings.TrimSpace(s)
}

// truncateStr shortens a string for logging purposes.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// deduplicateFindings removes duplicate findings that were independently found
// by multiple review passes. Two findings are considered duplicates if they
// reference the same file, their line numbers are within 3 lines, and their
// descriptions refer to the same issue. When duplicates are found, the finding
// with higher severity is kept. If severities are equal, the earlier finding
// (from the earlier pass) is kept.
func deduplicateFindings(findings []interfaces.Finding) []interfaces.Finding {
	if len(findings) <= 1 {
		return findings
	}

	// Track which findings have been marked as duplicates.
	removed := make([]bool, len(findings))

	for i := 0; i < len(findings); i++ {
		if removed[i] {
			continue
		}
		for j := i + 1; j < len(findings); j++ {
			if removed[j] {
				continue
			}
			if !isDuplicate(findings[i], findings[j]) {
				continue
			}
			// Keep the one with higher severity; if equal, keep earlier (i).
			if severityRank(findings[j].Severity) > severityRank(findings[i].Severity) {
				removed[i] = true
				break // i is removed, no need to compare it further.
			}
			removed[j] = true
		}
	}

	result := make([]interfaces.Finding, 0, len(findings))
	for i, f := range findings {
		if !removed[i] {
			result = append(result, f)
		}
	}
	return result
}

// isDuplicate returns true if two findings refer to the same issue.
func isDuplicate(a, b interfaces.Finding) bool {
	if a.File != b.File {
		return false
	}

	lineDist := a.StartLine - b.StartLine
	if lineDist < 0 {
		lineDist = -lineDist
	}
	if lineDist > 3 {
		return false
	}

	return descriptionsSimilar(a.Description, b.Description)
}

// descriptionsSimilar returns true if two descriptions refer to the same issue.
// It uses two heuristics:
//  1. The first 4 significant words of both descriptions match exactly.
//  2. At least 3 of the first 4 significant words of one description appear
//     anywhere in the other description's significant words.
func descriptionsSimilar(a, b string) bool {
	wordsA := significantWords(a)
	wordsB := significantWords(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return false
	}

	// Check if first N significant words match exactly.
	n := 4
	if len(wordsA) < n {
		n = len(wordsA)
	}
	if len(wordsB) < n {
		n = len(wordsB)
	}
	if n > 0 {
		match := true
		for i := 0; i < n; i++ {
			if wordsA[i] != wordsB[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	// Check if the key words of one appear in the other's word set.
	if keyWordsOverlap(wordsA, wordsB) || keyWordsOverlap(wordsB, wordsA) {
		return true
	}

	return false
}

// keyWordsOverlap returns true if at least 3 of the first 4 significant words
// of src appear anywhere in dst.
func keyWordsOverlap(src, dst []string) bool {
	n := 4
	if len(src) < n {
		n = len(src)
	}
	if n < 3 {
		return false
	}

	dstSet := make(map[string]bool, len(dst))
	for _, w := range dst {
		dstSet[w] = true
	}

	matches := 0
	for i := 0; i < n; i++ {
		if dstSet[src[i]] {
			matches++
		}
	}
	return matches >= 3
}

// significantWords extracts lowercase words from a string, filtering out
// common stop words that don't contribute to meaning.
func significantWords(s string) []string {
	words := strings.Fields(strings.ToLower(s))
	result := make([]string, 0, len(words))
	for _, w := range words {
		// Strip common punctuation from edges.
		w = strings.Trim(w, ".,;:!?\"'()[]{}")
		if w == "" {
			continue
		}
		if stopWords[w] {
			continue
		}
		result = append(result, w)
	}
	return result
}

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true,
	"did": true, "will": true, "would": true, "could": true, "should": true,
	"may": true, "might": true, "shall": true, "can": true,
	"in": true, "on": true, "at": true, "to": true, "for": true,
	"of": true, "with": true, "by": true, "from": true, "as": true,
	"into": true, "through": true, "during": true, "before": true, "after": true,
	"and": true, "but": true, "or": true, "nor": true, "not": true,
	"so": true, "yet": true, "both": true, "either": true, "neither": true,
	"this": true, "that": true, "these": true, "those": true,
	"it": true, "its": true, "he": true, "she": true, "they": true,
	"we": true, "you": true, "i": true, "me": true,
}

// severityRank returns a numeric rank for severity (higher = more severe).
func severityRank(s interfaces.Severity) int {
	switch s {
	case interfaces.SeverityCritical:
		return 5
	case interfaces.SeverityHigh:
		return 4
	case interfaces.SeverityMedium:
		return 3
	case interfaces.SeverityLow:
		return 2
	case interfaces.SeverityInfo:
		return 1
	default:
		return 0
	}
}
