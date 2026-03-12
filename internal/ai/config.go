package ai

// config.go - AI Configuration
//
// Handles loading AI configuration from environment variables and config files.
// Priority: Environment variables > Config file > Defaults

import (
	"os"
	"strconv"
	"strings"
)

// Config holds the AI configuration
type Config struct {
	// Provider specifies which AI provider to use
	// Options: "ollama" (default), "openai", "anthropic", "azure"
	Provider string

	// Model specifies the default model to use (provider-specific)
	Model string

	// APIKey is the API key for the provider (not needed for Ollama)
	APIKey string

	// BaseURL is the base URL for the API (for self-hosted or proxies)
	BaseURL string

	// Temperature is the default temperature for completions
	Temperature float64

	// MaxTokens is the default max tokens for completions
	MaxTokens int

	// Timeout in seconds for API calls
	Timeout int

	// Ollama-specific settings
	OllamaHost  string
	OllamaModel string

	// OpenAI-specific settings
	OpenAIKey     string
	OpenAIModel   string
	OpenAIOrgID   string
	OpenAIBaseURL string

	// Anthropic-specific settings
	AnthropicKey   string
	AnthropicModel string

	// Azure OpenAI settings
	AzureEndpoint   string
	AzureKey        string
	AzureDeployment string
	AzureAPIVersion string

	// Groq settings (fast inference)
	GroqKey   string
	GroqModel string
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Provider:       "ollama",
		Temperature:    0.7,
		MaxTokens:      2048,
		Timeout:        60,
		OllamaHost:     "http://localhost:11434",
		OllamaModel:    "llama3.2",
		OpenAIModel:    "gpt-4o-mini",
		AnthropicModel: "claude-3-haiku-20240307",
		GroqModel:      "llama-3.3-70b-versatile",
	}
}

// LoadConfig loads AI configuration from environment variables
func LoadConfig() *Config {
	cfg := DefaultConfig()

	// Main provider selection
	if v := os.Getenv("FLIP_AI_PROVIDER"); v != "" {
		cfg.Provider = strings.ToLower(v)
	}

	// Temperature
	if v := os.Getenv("FLIP_AI_TEMPERATURE"); v != "" {
		if t, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Temperature = t
		}
	}

	// Max tokens
	if v := os.Getenv("FLIP_AI_MAX_TOKENS"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			cfg.MaxTokens = t
		}
	}

	// Timeout
	if v := os.Getenv("FLIP_AI_TIMEOUT"); v != "" {
		if t, err := strconv.Atoi(v); err == nil {
			cfg.Timeout = t
		}
	}

	// Ollama settings
	if v := os.Getenv("OLLAMA_HOST"); v != "" {
		cfg.OllamaHost = v
	}
	if v := os.Getenv("FLIP_OLLAMA_MODEL"); v != "" {
		cfg.OllamaModel = v
	}

	// OpenAI settings
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.OpenAIKey = v
	}
	if v := os.Getenv("FLIP_OPENAI_MODEL"); v != "" {
		cfg.OpenAIModel = v
	}
	if v := os.Getenv("OPENAI_ORG_ID"); v != "" {
		cfg.OpenAIOrgID = v
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		cfg.OpenAIBaseURL = v
	}

	// Anthropic settings
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.AnthropicKey = v
	}
	if v := os.Getenv("FLIP_ANTHROPIC_MODEL"); v != "" {
		cfg.AnthropicModel = v
	}

	// Azure OpenAI settings
	if v := os.Getenv("AZURE_OPENAI_ENDPOINT"); v != "" {
		cfg.AzureEndpoint = v
	}
	if v := os.Getenv("AZURE_OPENAI_KEY"); v != "" {
		cfg.AzureKey = v
	}
	if v := os.Getenv("AZURE_OPENAI_DEPLOYMENT"); v != "" {
		cfg.AzureDeployment = v
	}
	if v := os.Getenv("AZURE_OPENAI_API_VERSION"); v != "" {
		cfg.AzureAPIVersion = v
	}

	// Groq settings
	if v := os.Getenv("GROQ_API_KEY"); v != "" {
		cfg.GroqKey = v
	}
	if v := os.Getenv("FLIP_GROQ_MODEL"); v != "" {
		cfg.GroqModel = v
	}

	// Set APIKey and Model based on provider
	switch cfg.Provider {
	case "openai":
		cfg.APIKey = cfg.OpenAIKey
		cfg.Model = cfg.OpenAIModel
		cfg.BaseURL = cfg.OpenAIBaseURL
	case "anthropic":
		cfg.APIKey = cfg.AnthropicKey
		cfg.Model = cfg.AnthropicModel
	case "azure":
		cfg.APIKey = cfg.AzureKey
		cfg.Model = cfg.AzureDeployment
		cfg.BaseURL = cfg.AzureEndpoint
	case "groq":
		cfg.APIKey = cfg.GroqKey
		cfg.Model = cfg.GroqModel
		cfg.BaseURL = "https://api.groq.com/openai/v1"
	default: // ollama
		cfg.Model = cfg.OllamaModel
		cfg.BaseURL = cfg.OllamaHost
	}

	return cfg
}

// IsEnabled returns true if AI features are enabled
// AI is enabled by default with Ollama (zero-config)
// Use FLIP_FEATURE_AI=0 to disable
func IsEnabled() bool {
	if v := os.Getenv("FLIP_FEATURE_AI"); v != "" {
		return v != "0" && strings.ToLower(v) != "false"
	}
	return true // Enabled by default (assumes Ollama)
}

// HasAPIKey returns true if the configured provider has an API key
func (c *Config) HasAPIKey() bool {
	switch c.Provider {
	case "ollama":
		return true // Ollama doesn't need an API key
	case "openai":
		return c.OpenAIKey != ""
	case "anthropic":
		return c.AnthropicKey != ""
	case "azure":
		return c.AzureKey != ""
	case "groq":
		return c.GroqKey != ""
	default:
		return false
	}
}

// Validate checks if the configuration is valid for the selected provider
func (c *Config) Validate() error {
	switch c.Provider {
	case "ollama":
		if c.OllamaHost == "" {
			return NewProviderError("ollama", ErrCodeConfiguration, "OLLAMA_HOST not set", nil)
		}
	case "openai":
		if c.OpenAIKey == "" {
			return NewProviderError("openai", ErrCodeConfiguration, "OPENAI_API_KEY not set", nil)
		}
	case "anthropic":
		if c.AnthropicKey == "" {
			return NewProviderError("anthropic", ErrCodeConfiguration, "ANTHROPIC_API_KEY not set", nil)
		}
	case "azure":
		if c.AzureEndpoint == "" || c.AzureKey == "" {
			return NewProviderError("azure", ErrCodeConfiguration, "Azure OpenAI endpoint or key not set", nil)
		}
	case "groq":
		if c.GroqKey == "" {
			return NewProviderError("groq", ErrCodeConfiguration, "GROQ_API_KEY not set", nil)
		}
	default:
		return NewProviderError(c.Provider, ErrCodeConfiguration, "unknown provider", nil)
	}
	return nil
}
