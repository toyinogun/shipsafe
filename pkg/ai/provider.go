// Package ai provides LLM-powered code review capabilities for ShipSafe.
// It supports any OpenAI-compatible API endpoint (Ollama, vLLM, LocalAI, OpenRouter, OpenAI)
// as well as Anthropic's Claude API.
package ai

import "context"

// ProviderType identifies the LLM provider protocol.
type ProviderType string

const (
	ProviderOpenAICompatible ProviderType = "openai-compatible"
	ProviderAnthropic        ProviderType = "anthropic"
)

// ProviderConfig holds the configuration for connecting to an LLM provider.
type ProviderConfig struct {
	Endpoint string       `json:"endpoint" yaml:"endpoint"` // Base URL (e.g., "http://ollama:11434/v1")
	Model    string       `json:"model" yaml:"model"`       // Model name (e.g., "codellama:13b")
	APIKey   string       `json:"-" yaml:"-"`               // API key (never serialized)
	Type     ProviderType `json:"type" yaml:"type"`         // Provider protocol type
}

// CompletionOpts configures a single LLM completion request.
type CompletionOpts struct {
	MaxTokens    int     `json:"max_tokens,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
	SystemPrompt string  `json:"system_prompt,omitempty"`
}

// LLMProvider abstracts communication with an LLM backend.
type LLMProvider interface {
	// Complete sends a prompt to the LLM and returns the response text.
	Complete(ctx context.Context, prompt string, opts CompletionOpts) (string, error)

	// Available checks whether the provider endpoint is configured and reachable.
	Available(ctx context.Context) bool
}
