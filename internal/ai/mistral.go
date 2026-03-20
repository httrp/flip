package ai

// mistral.go - Mistral Provider Implementation
//
// Mistral provides high-quality models (open-weights and proprietary).
// Uses OpenAI-compatible API, so we extend OpenAIProvider.
//
// Models: mistral-large-latest, pixtral-large-latest, ministral-8b-latest, mistral-small-latest

import (
	"context"
	"net/http"
	"time"
)

// MistralProvider implements the Provider interface for Mistral
// Mistral uses an OpenAI-compatible API
type MistralProvider struct {
	*OpenAIProvider
}

// NewMistralProvider creates a new Mistral provider
func NewMistralProvider(cfg *Config) *MistralProvider {
	baseURL := "https://api.mistral.ai/v1"

	model := cfg.MistralModel
	if model == "" {
		model = "mistral-large-latest"
	}

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &MistralProvider{
		OpenAIProvider: &OpenAIProvider{
			apiKey:  cfg.MistralKey,
			baseURL: baseURL,
			model:   model,
			timeout: timeout,
			client: &http.Client{
				Timeout: timeout,
			},
		},
	}
}

func (p *MistralProvider) Name() string {
	return "mistral"
}

// ListModels returns the available Mistral models without API calls.
func (p *MistralProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	return MistralModels(), nil
}

// MistralModels returns the list of available Mistral models
func MistralModels() []ModelInfo {
	return []ModelInfo{
		{ID: "mistral-large-latest", Name: "Mistral Large", Description: "Top-tier reasoning, high capability", ContextSize: 131000},
		{ID: "pixtral-large-latest", Name: "Pixtral Large", Description: "Multimodal frontier model", ContextSize: 131000},
		{ID: "ministral-8b-latest", Name: "Ministral 8B", Description: "Powerful edge model", ContextSize: 32000},
		{ID: "mistral-small-latest", Name: "Mistral Small", Description: "Cost-effective, low latency", ContextSize: 32000},
		{ID: "open-mistral-nemo", Name: "Mistral Nemo", Description: "Open-weights 12B model", ContextSize: 128000},
		{ID: "codestral-latest", Name: "Codestral", Description: "Code generation model", ContextSize: 32000},
	}
}
