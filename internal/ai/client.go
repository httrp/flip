package ai

// client.go - AI Client Factory
//
// Provides a unified interface for creating and managing AI providers.

import (
	"context"
	"fmt"
)

// Client wraps an AI provider with additional functionality
type Client struct {
	provider Provider
	config   *Config
}

// NewClient creates a new AI client from configuration
func NewClient() (*Client, error) {
	cfg := LoadConfig()
	return NewClientWithConfig(cfg)
}

// NewClientWithConfig creates a new AI client with the given configuration
func NewClientWithConfig(cfg *Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	var provider Provider
	switch cfg.Provider {
	case "ollama":
		provider = NewOllamaProvider(cfg)
	case "openai":
		provider = NewOpenAIProvider(cfg)
	case "anthropic":
		provider = NewAnthropicProvider(cfg)
	case "azure":
		// Azure uses OpenAI-compatible API
		provider = NewAzureOpenAIProvider(cfg)
	case "groq":
		// Groq uses OpenAI-compatible API with custom LPU
		provider = NewGroqProvider(cfg)
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}

	return &Client{
		provider: provider,
		config:   cfg,
	}, nil
}

// Provider returns the underlying AI provider
func (c *Client) Provider() Provider {
	return c.provider
}

// Config returns the client configuration
func (c *Client) Config() *Config {
	return c.config
}

// IsAvailable checks if the AI provider is available
func (c *Client) IsAvailable(ctx context.Context) bool {
	return c.provider.IsAvailable(ctx)
}

// Complete sends a prompt and returns the full response
func (c *Client) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return c.provider.Complete(ctx, req)
}

// CompleteStream sends a prompt and streams the response
func (c *Client) CompleteStream(ctx context.Context, req *CompletionRequest) (Stream, error) {
	return c.provider.CompleteStream(ctx, req)
}

// SimpleComplete sends a simple text prompt and returns the response
func (c *Client) SimpleComplete(ctx context.Context, prompt string) (string, error) {
	resp, err := c.provider.Complete(ctx, &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: prompt},
		},
		Temperature: c.config.Temperature,
		MaxTokens:   c.config.MaxTokens,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// ChatComplete sends a chat with system prompt and returns the response
func (c *Client) ChatComplete(ctx context.Context, system, user string) (string, error) {
	resp, err := c.provider.Complete(ctx, &CompletionRequest{
		System: system,
		Messages: []Message{
			{Role: RoleUser, Content: user},
		},
		Temperature: c.config.Temperature,
		MaxTokens:   c.config.MaxTokens,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// ListModels returns available models from the provider
func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	return c.provider.ListModels(ctx)
}

// NewAzureOpenAIProvider creates an Azure OpenAI provider
// Azure uses the OpenAI API format with different authentication
func NewAzureOpenAIProvider(cfg *Config) *OpenAIProvider {
	// Build Azure endpoint URL
	baseURL := cfg.AzureEndpoint
	if cfg.AzureAPIVersion != "" {
		// Azure requires API version in URL
		baseURL = fmt.Sprintf("%s/openai/deployments/%s", cfg.AzureEndpoint, cfg.AzureDeployment)
	}

	return &OpenAIProvider{
		apiKey:  cfg.AzureKey,
		baseURL: baseURL,
		model:   cfg.AzureDeployment,
		client:  nil, // Will be initialized with default
	}
}

// GetDefaultProvider returns the default provider name based on what's available
func GetDefaultProvider() string {
	cfg := LoadConfig()

	// Check in order of preference
	if cfg.OpenAIKey != "" {
		return "openai"
	}
	if cfg.AnthropicKey != "" {
		return "anthropic"
	}
	if cfg.GroqKey != "" {
		return "groq"
	}
	if cfg.AzureKey != "" {
		return "azure"
	}

	// Default to Ollama (works without API key)
	return "ollama"
}

// QuickCheck performs a quick availability check for all providers
func QuickCheck(ctx context.Context) map[string]bool {
	results := make(map[string]bool)
	cfg := LoadConfig()

	// Check Ollama
	ollama := NewOllamaProvider(cfg)
	results["ollama"] = ollama.IsAvailable(ctx)

	// Check OpenAI if configured
	if cfg.OpenAIKey != "" {
		openai := NewOpenAIProvider(cfg)
		results["openai"] = openai.IsAvailable(ctx)
	}

	// Check Anthropic if configured
	if cfg.AnthropicKey != "" {
		anthropicProvider := NewAnthropicProvider(cfg)
		results["anthropic"] = anthropicProvider.IsAvailable(ctx)
	}

	// Check Groq if configured
	if cfg.GroqKey != "" {
		groqProvider := NewGroqProvider(cfg)
		results["groq"] = groqProvider.IsAvailable(ctx)
	}

	return results
}
