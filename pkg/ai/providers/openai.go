// Package providers implements concrete LLM provider backends.
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/toyinlola/shipsafe/pkg/ai"
)

// OpenAIProvider implements ai.LLMProvider for any OpenAI-compatible API.
// This covers Ollama, vLLM, LocalAI, OpenRouter, and OpenAI itself.
type OpenAIProvider struct {
	config ai.ProviderConfig
	client *http.Client
}

// NewOpenAIProvider creates a provider for OpenAI-compatible endpoints.
func NewOpenAIProvider(cfg ai.ProviderConfig, timeout time.Duration) *OpenAIProvider {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	return &OpenAIProvider{
		config: cfg,
		client: &http.Client{Timeout: timeout},
	}
}

// chatRequest is the OpenAI chat completions request body.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the OpenAI chat completions response body.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Complete sends a prompt to the OpenAI-compatible endpoint and returns the response.
func (p *OpenAIProvider) Complete(ctx context.Context, prompt string, opts ai.CompletionOpts) (string, error) {
	messages := make([]chatMessage, 0, 2)
	if opts.SystemPrompt != "" {
		messages = append(messages, chatMessage{Role: "system", Content: opts.SystemPrompt})
	}
	messages = append(messages, chatMessage{Role: "user", Content: prompt})

	reqBody := chatRequest{
		Model:    p.config.Model,
		Messages: messages,
		Stream:   false,
	}
	if opts.MaxTokens > 0 {
		reqBody.MaxTokens = opts.MaxTokens
	}
	if opts.Temperature > 0 {
		t := opts.Temperature
		reqBody.Temperature = &t
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ai: marshaling request: %w", err)
	}

	url := p.config.Endpoint + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ai: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ai: sending request to %s: %w", url, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ai: reading response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("ai: rate limited by provider (HTTP 429)")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ai: provider returned HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("ai: decoding response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("ai: provider error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("ai: provider returned no choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// Available checks if the provider endpoint is reachable by sending a lightweight request.
func (p *OpenAIProvider) Available(ctx context.Context) bool {
	if p.config.Endpoint == "" || p.config.Model == "" {
		return false
	}

	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := p.config.Endpoint + "/models"
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, url, nil)
	if err != nil {
		slog.Debug("ai: availability check failed", "error", err)
		return false
	}
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		slog.Debug("ai: endpoint unreachable", "endpoint", p.config.Endpoint, "error", err)
		return false
	}
	defer resp.Body.Close() //nolint:errcheck

	return resp.StatusCode == http.StatusOK
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
