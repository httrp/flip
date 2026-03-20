package ai

// config.go - AI Configuration
//
// Handles loading AI configuration from environment variables and config files.
// Priority: Environment variables > Config file > Defaults

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Config holds the AI configuration
type Config struct {
	// Provider specifies which AI provider to use
	// Options: "ollama" (default), "openai", "anthropic", "groq", "mistral", "azure"
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

	// Mistral settings
	MistralKey   string
	MistralModel string
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
		MistralModel:   "mistral-large-latest",
	}
}

// LoadConfig loads AI configuration from environment variables
func LoadConfig() *Config {
	cfg := DefaultConfig()

	// Load from config file first (lowest priority after defaults)
	loadConfigFile(cfg)

	// Main provider selection (env vars override config file)
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

	// Mistral settings
	if v := os.Getenv("MISTRAL_API_KEY"); v != "" {
		cfg.MistralKey = v
	}
	if v := os.Getenv("FLIP_MISTRAL_MODEL"); v != "" {
		cfg.MistralModel = v
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
	case "mistral":
		cfg.APIKey = cfg.MistralKey
		cfg.Model = cfg.MistralModel
		cfg.BaseURL = "https://api.mistral.ai/v1"
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
	case "mistral":
		return c.MistralKey != ""
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
	case "mistral":
		if c.MistralKey == "" {
			return NewProviderError("mistral", ErrCodeConfiguration, "MISTRAL_API_KEY not set", nil)
		}
	default:
		return NewProviderError(c.Provider, ErrCodeConfiguration, "unknown provider", nil)
	}
	return nil
}

// getFlipConfigDir returns the flip configuration directory (same logic as platform.GetFlipConfigDir
// but duplicated here to avoid circular imports).
func getFlipConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "flip"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "flip"), nil
	case "linux":
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(home, ".config")
		}
		return filepath.Join(configDir, "flip"), nil
	default:
		return filepath.Join(home, ".flip"), nil
	}
}

// ConfigFilePath returns the path to the AI config file.
func ConfigFilePath() string {
	dir, err := getFlipConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "ai.env")
}

// loadConfigFile loads key=value pairs from the AI config file.
// Only sets values that are not already set by environment variables.
func loadConfigFile(cfg *Config) {
	path := ConfigFilePath()
	if path == "" {
		return
	}

	f, err := os.Open(path)
	if err != nil {
		return // file doesn't exist, that's OK
	}
	defer f.Close()

	fileVars := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Remove surrounding quotes if present
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		fileVars[key] = val
	}

	// Apply file values only where env is not already set
	applyIfEmpty := func(envKey, fileVal string, setter func(string)) {
		if os.Getenv(envKey) == "" && fileVal != "" {
			setter(fileVal)
		}
	}

	applyIfEmpty("FLIP_AI_PROVIDER", fileVars["FLIP_AI_PROVIDER"], func(v string) { cfg.Provider = strings.ToLower(v) })
	applyIfEmpty("OLLAMA_HOST", fileVars["OLLAMA_HOST"], func(v string) { cfg.OllamaHost = v })
	applyIfEmpty("FLIP_OLLAMA_MODEL", fileVars["FLIP_OLLAMA_MODEL"], func(v string) { cfg.OllamaModel = v })
	applyIfEmpty("OPENAI_API_KEY", fileVars["OPENAI_API_KEY"], func(v string) { cfg.OpenAIKey = v })
	applyIfEmpty("FLIP_OPENAI_MODEL", fileVars["FLIP_OPENAI_MODEL"], func(v string) { cfg.OpenAIModel = v })
	applyIfEmpty("OPENAI_ORG_ID", fileVars["OPENAI_ORG_ID"], func(v string) { cfg.OpenAIOrgID = v })
	applyIfEmpty("OPENAI_BASE_URL", fileVars["OPENAI_BASE_URL"], func(v string) { cfg.OpenAIBaseURL = v })
	applyIfEmpty("ANTHROPIC_API_KEY", fileVars["ANTHROPIC_API_KEY"], func(v string) { cfg.AnthropicKey = v })
	applyIfEmpty("FLIP_ANTHROPIC_MODEL", fileVars["FLIP_ANTHROPIC_MODEL"], func(v string) { cfg.AnthropicModel = v })
	applyIfEmpty("AZURE_OPENAI_ENDPOINT", fileVars["AZURE_OPENAI_ENDPOINT"], func(v string) { cfg.AzureEndpoint = v })
	applyIfEmpty("AZURE_OPENAI_KEY", fileVars["AZURE_OPENAI_KEY"], func(v string) { cfg.AzureKey = v })
	applyIfEmpty("AZURE_OPENAI_DEPLOYMENT", fileVars["AZURE_OPENAI_DEPLOYMENT"], func(v string) { cfg.AzureDeployment = v })
	applyIfEmpty("AZURE_OPENAI_API_VERSION", fileVars["AZURE_OPENAI_API_VERSION"], func(v string) { cfg.AzureAPIVersion = v })
	applyIfEmpty("GROQ_API_KEY", fileVars["GROQ_API_KEY"], func(v string) { cfg.GroqKey = v })
	applyIfEmpty("FLIP_GROQ_MODEL", fileVars["FLIP_GROQ_MODEL"], func(v string) { cfg.GroqModel = v })
	applyIfEmpty("MISTRAL_API_KEY", fileVars["MISTRAL_API_KEY"], func(v string) { cfg.MistralKey = v })
	applyIfEmpty("FLIP_MISTRAL_MODEL", fileVars["FLIP_MISTRAL_MODEL"], func(v string) { cfg.MistralModel = v })
}

// SaveConfig writes the current AI configuration to the config file.
// This persists API keys and provider settings so they survive across sessions.
func SaveConfig(cfg *Config) error {
	path := ConfigFilePath()
	if path == "" {
		return fmt.Errorf("could not determine config directory")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	var lines []string
	lines = append(lines, "# Flip AI Configuration")
	lines = append(lines, "# Managed by 'flip ai setup'. Environment variables override these values.")
	lines = append(lines, "")

	// Always write the active provider
	if cfg.Provider != "" {
		lines = append(lines, fmt.Sprintf("FLIP_AI_PROVIDER=%s", cfg.Provider))
	}
	lines = append(lines, "")

	// Ollama
	if cfg.OllamaHost != "" && cfg.OllamaHost != "http://localhost:11434" {
		lines = append(lines, fmt.Sprintf("OLLAMA_HOST=%s", cfg.OllamaHost))
	}
	if cfg.OllamaModel != "" && cfg.OllamaModel != "llama3.2" {
		lines = append(lines, fmt.Sprintf("FLIP_OLLAMA_MODEL=%s", cfg.OllamaModel))
	}

	// OpenAI
	if cfg.OpenAIKey != "" {
		lines = append(lines, fmt.Sprintf("OPENAI_API_KEY=%s", cfg.OpenAIKey))
	}
	if cfg.OpenAIModel != "" && cfg.OpenAIModel != "gpt-4o-mini" {
		lines = append(lines, fmt.Sprintf("FLIP_OPENAI_MODEL=%s", cfg.OpenAIModel))
	}

	// Anthropic
	if cfg.AnthropicKey != "" {
		lines = append(lines, fmt.Sprintf("ANTHROPIC_API_KEY=%s", cfg.AnthropicKey))
	}
	if cfg.AnthropicModel != "" && cfg.AnthropicModel != "claude-3-haiku-20240307" {
		lines = append(lines, fmt.Sprintf("FLIP_ANTHROPIC_MODEL=%s", cfg.AnthropicModel))
	}

	// Groq
	if cfg.GroqKey != "" {
		lines = append(lines, fmt.Sprintf("GROQ_API_KEY=%s", cfg.GroqKey))
	}
	if cfg.GroqModel != "" && cfg.GroqModel != "llama-3.3-70b-versatile" {
		lines = append(lines, fmt.Sprintf("FLIP_GROQ_MODEL=%s", cfg.GroqModel))
	}

	// Mistral
	if cfg.MistralKey != "" {
		lines = append(lines, fmt.Sprintf("MISTRAL_API_KEY=%s", cfg.MistralKey))
	}
	if cfg.MistralModel != "" && cfg.MistralModel != "mistral-large-latest" {
		lines = append(lines, fmt.Sprintf("FLIP_MISTRAL_MODEL=%s", cfg.MistralModel))
	}

	// Azure
	if cfg.AzureEndpoint != "" {
		lines = append(lines, fmt.Sprintf("AZURE_OPENAI_ENDPOINT=%s", cfg.AzureEndpoint))
	}
	if cfg.AzureKey != "" {
		lines = append(lines, fmt.Sprintf("AZURE_OPENAI_KEY=%s", cfg.AzureKey))
	}
	if cfg.AzureDeployment != "" {
		lines = append(lines, fmt.Sprintf("AZURE_OPENAI_DEPLOYMENT=%s", cfg.AzureDeployment))
	}
	if cfg.AzureAPIVersion != "" {
		lines = append(lines, fmt.Sprintf("AZURE_OPENAI_API_VERSION=%s", cfg.AzureAPIVersion))
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0600)
}
