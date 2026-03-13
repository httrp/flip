package ai

// groq.go - Groq Provider Implementation
//
// Groq provides extremely fast inference using custom LPU hardware.
// Uses OpenAI-compatible API, so we extend OpenAIProvider.
// Free tier available with generous rate limits.
//
// Models: llama-3.3-70b-versatile, llama-3.1-8b-instant, mixtral-8x7b-32768

import (
	"context"
	"net/http"
	"time"
)

// GroqProvider implements the Provider interface for Groq
// Groq uses an OpenAI-compatible API
type GroqProvider struct {
	*OpenAIProvider
}

// NewGroqProvider creates a new Groq provider
func NewGroqProvider(cfg *Config) *GroqProvider {
	baseURL := "https://api.groq.com/openai/v1"

	model := cfg.GroqModel
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &GroqProvider{
		OpenAIProvider: &OpenAIProvider{
			apiKey:  cfg.GroqKey,
			baseURL: baseURL,
			model:   model,
			timeout: timeout,
			client: &http.Client{
				Timeout: timeout,
			},
		},
	}
}

func (p *GroqProvider) Name() string {
	return "groq"
}

// ListModels returns the available Groq models without API calls.
func (p *GroqProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	return GroqModels(), nil
}

// GroqModels returns the list of available Groq models
func GroqModels() []ModelInfo {
	return []ModelInfo{
		{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B Versatile", Description: "Best overall performance", ContextSize: 128000},
		{ID: "llama-3.1-70b-versatile", Name: "Llama 3.1 70B Versatile", Description: "High quality, versatile", ContextSize: 128000},
		{ID: "llama-3.1-8b-instant", Name: "Llama 3.1 8B Instant", Description: "Very fast, good quality", ContextSize: 128000},
		{ID: "llama-3.2-90b-vision-preview", Name: "Llama 3.2 90B Vision", Description: "Vision capabilities", ContextSize: 128000},
		{ID: "mixtral-8x7b-32768", Name: "Mixtral 8x7B", Description: "MoE model, 32K context", ContextSize: 32768},
		{ID: "gemma2-9b-it", Name: "Gemma 2 9B", Description: "Google Gemma 2", ContextSize: 8192},
	}
}
