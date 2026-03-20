package ai

// validate.go - API Key Validation
//
// Provides lightweight key validation for all providers.
// Uses the cheapest possible API call to verify that a key is accepted.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ValidateAPIKey checks whether the configured API key is accepted by the provider.
// Uses lightweight API calls that consume minimal or zero quota.
func ValidateAPIKey(ctx context.Context, cfg *Config) *KeyStatus {
	provider := cfg.Provider

	switch provider {
	case "ollama":
		return &KeyStatus{Provider: provider, Valid: true, Message: "no API key required"}
	case "openai":
		return validateOpenAIKey(ctx, cfg.OpenAIKey, cfg.OpenAIBaseURL, "openai")
	case "anthropic":
		return validateAnthropicKey(ctx, cfg.AnthropicKey)
	case "groq":
		return validateOpenAIKey(ctx, cfg.GroqKey, "https://api.groq.com/openai/v1", "groq")
	case "mistral":
		return validateOpenAIKey(ctx, cfg.MistralKey, "https://api.mistral.ai/v1", "mistral")
	case "azure":
		return validateAzureKey(ctx, cfg)
	default:
		return &KeyStatus{Provider: provider, Valid: false, Message: "unknown provider"}
	}
}

// validateOpenAIKey validates a key against an OpenAI-compatible /models endpoint.
// Works for OpenAI, Groq, and Mistral.
func validateOpenAIKey(ctx context.Context, apiKey string, baseURL string, provider string) *KeyStatus {
	if apiKey == "" {
		return &KeyStatus{Provider: provider, Valid: false, Message: "no API key configured"}
	}

	if baseURL == "" {
		switch provider {
		case "openai":
			baseURL = "https://api.openai.com/v1"
		case "groq":
			baseURL = "https://api.groq.com/openai/v1"
		case "mistral":
			baseURL = "https://api.mistral.ai/v1"
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
	if err != nil {
		return &KeyStatus{Provider: provider, Valid: false, Message: fmt.Sprintf("request error: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return &KeyStatus{Provider: provider, Valid: true, Message: "key set, but provider not reachable"}
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return &KeyStatus{Provider: provider, Valid: true, Message: "API key valid ✓"}
	case http.StatusUnauthorized, http.StatusForbidden:
		return &KeyStatus{Provider: provider, Valid: false, Message: "API key rejected (invalid or revoked)"}
	case http.StatusTooManyRequests:
		return &KeyStatus{Provider: provider, Valid: true, RateLimited: true, Message: "API key valid (rate-limited)"}
	default:
		return &KeyStatus{Provider: provider, Valid: true, Message: fmt.Sprintf("key set (HTTP %d during check)", resp.StatusCode)}
	}
}

// validateAnthropicKey validates an Anthropic key by sending a minimal messages request.
// Anthropic doesn't have a /models endpoint, so we use a tiny completion request
// with max_tokens=1 to minimize cost.
func validateAnthropicKey(ctx context.Context, apiKey string) *KeyStatus {
	if apiKey == "" {
		return &KeyStatus{Provider: "anthropic", Valid: false, Message: "no API key configured"}
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Send a minimal request that will be rejected quickly (invalid model) or succeed with 1 token
	body := map[string]interface{}{
		"model":      "claude-3-haiku-20240307",
		"max_tokens": 1,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return &KeyStatus{Provider: "anthropic", Valid: false, Message: "internal error"}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return &KeyStatus{Provider: "anthropic", Valid: false, Message: fmt.Sprintf("request error: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return &KeyStatus{Provider: "anthropic", Valid: true, Message: "key set, but provider not reachable"}
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return &KeyStatus{Provider: "anthropic", Valid: true, Message: "API key valid ✓"}
	case http.StatusUnauthorized, http.StatusForbidden:
		return &KeyStatus{Provider: "anthropic", Valid: false, Message: "API key rejected (invalid or revoked)"}
	case http.StatusTooManyRequests:
		return &KeyStatus{Provider: "anthropic", Valid: true, RateLimited: true, Message: "API key valid (rate-limited)"}
	default:
		// Any other response (including 400) means the key was accepted
		return &KeyStatus{Provider: "anthropic", Valid: true, Message: "API key valid ✓"}
	}
}

// validateAzureKey validates an Azure OpenAI key.
func validateAzureKey(ctx context.Context, cfg *Config) *KeyStatus {
	if cfg.AzureKey == "" {
		return &KeyStatus{Provider: "azure", Valid: false, Message: "no API key configured"}
	}
	if cfg.AzureEndpoint == "" {
		return &KeyStatus{Provider: "azure", Valid: false, Message: "no endpoint configured"}
	}

	client := &http.Client{Timeout: 10 * time.Second}

	url := fmt.Sprintf("%s/openai/models?api-version=%s", cfg.AzureEndpoint, cfg.AzureAPIVersion)
	if cfg.AzureAPIVersion == "" {
		url = fmt.Sprintf("%s/openai/models?api-version=2024-02-01", cfg.AzureEndpoint)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &KeyStatus{Provider: "azure", Valid: false, Message: fmt.Sprintf("request error: %v", err)}
	}
	req.Header.Set("api-key", cfg.AzureKey)

	resp, err := client.Do(req)
	if err != nil {
		return &KeyStatus{Provider: "azure", Valid: true, Message: "key set, but endpoint not reachable"}
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return &KeyStatus{Provider: "azure", Valid: true, Message: "API key valid ✓"}
	case http.StatusUnauthorized, http.StatusForbidden:
		return &KeyStatus{Provider: "azure", Valid: false, Message: "API key rejected"}
	default:
		return &KeyStatus{Provider: "azure", Valid: true, Message: fmt.Sprintf("key set (HTTP %d during check)", resp.StatusCode)}
	}
}

// MaskAPIKey returns a masked version of an API key for display.
// Shows first 4 and last 4 characters: "sk-ab...wxyz"
func MaskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
