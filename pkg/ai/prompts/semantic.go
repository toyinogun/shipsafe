// Package prompts provides LLM prompt templates for AI-powered code review.
package prompts

import "fmt"

const semanticSystemPrompt = `You are a senior code reviewer performing a semantic analysis of a pull request.
Your job is to determine whether the code changes match the stated intent (PR title/description).
Focus on mismatches between what the PR says it does and what the code actually does.

You MUST respond with valid JSON only. No markdown, no commentary outside the JSON.

Response format:
{
  "findings": [
    {
      "file": "path/to/file.go",
      "line": 42,
      "severity": "medium",
      "title": "Short title of the issue",
      "description": "Detailed explanation of the mismatch",
      "suggestion": "How to fix it"
    }
  ]
}

Severity levels: "critical", "high", "medium", "low", "info"

If there are no issues, return: {"findings": []}
Do NOT invent issues that don't exist. Only report genuine mismatches.`

// SemanticPrompt builds the user prompt for semantic diff analysis.
func SemanticPrompt(diffContext string) string {
	return fmt.Sprintf(`Analyze the following code change for semantic correctness.
Does the implementation match the stated intent? Are there mismatches between what the PR description says and what the code actually does?

%s

Respond with JSON only.`, diffContext)
}

// SemanticSystemPrompt returns the system prompt for semantic analysis.
func SemanticSystemPrompt() string {
	return semanticSystemPrompt
}
