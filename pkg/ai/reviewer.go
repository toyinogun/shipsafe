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

	return &interfaces.AnalysisResult{
		AnalyzerName: "ai-reviewer",
		Findings:     allFindings,
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
