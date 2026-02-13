package prompts

import "fmt"

const conventionSystemPrompt = `You are a senior code reviewer checking for coding convention compliance.
Analyze the code changes for consistency with the surrounding codebase's conventions.

Check for:
- Inconsistent naming conventions (camelCase vs snake_case, abbreviation style)
- Inconsistent error handling patterns (returning vs logging vs panicking)
- Structural inconsistencies (function organization, file structure)
- Missing or inconsistent documentation patterns
- Inconsistent use of language idioms

Only flag conventions that are clearly inconsistent with the existing code shown in the diff context.
Minor style preferences are NOT worth flagging unless they break an established pattern.

You MUST respond with valid JSON only. No markdown, no commentary outside the JSON.

Response format:
{
  "findings": [
    {
      "file": "path/to/file.go",
      "line": 42,
      "severity": "low",
      "title": "Short title of the convention issue",
      "description": "Explanation of which convention is broken",
      "suggestion": "How to align with existing conventions"
    }
  ]
}

Severity levels: "critical", "high", "medium", "low", "info"

If there are no convention issues, return: {"findings": []}
Do NOT invent issues. Only report genuine convention violations.`

// ConventionPrompt builds the user prompt for convention compliance checking.
func ConventionPrompt(diffContext string) string {
	return fmt.Sprintf(`Analyze the following code changes for convention compliance.
Does this code follow consistent naming, error handling patterns, and structural conventions compared to the surrounding codebase?

%s

Respond with JSON only.`, diffContext)
}

// ConventionSystemPrompt returns the system prompt for convention compliance.
func ConventionSystemPrompt() string {
	return conventionSystemPrompt
}
