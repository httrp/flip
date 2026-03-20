package ai

import (
	"os"
	"testing"
)

func TestDefaultConfig_Values(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Provider != "ollama" {
		t.Errorf("expected default provider 'ollama', got %q", cfg.Provider)
	}
	if cfg.OllamaHost != "http://localhost:11434" {
		t.Errorf("unexpected default OllamaHost: %q", cfg.OllamaHost)
	}
	if cfg.OllamaModel != "llama3.2" {
		t.Errorf("unexpected default OllamaModel: %q", cfg.OllamaModel)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("unexpected default Temperature: %v", cfg.Temperature)
	}
	if cfg.MaxTokens != 2048 {
		t.Errorf("unexpected default MaxTokens: %d", cfg.MaxTokens)
	}
	if cfg.Timeout != 60 {
		t.Errorf("unexpected default Timeout: %d", cfg.Timeout)
	}
}

func TestLoadConfig_ProviderOverride(t *testing.T) {
	t.Setenv("FLIP_AI_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "sk-test")

	cfg := LoadConfig()
	if cfg.Provider != "openai" {
		t.Errorf("expected provider 'openai', got %q", cfg.Provider)
	}
}

func TestLoadConfig_TemperatureOverride(t *testing.T) {
	t.Setenv("FLIP_AI_TEMPERATURE", "0.2")

	cfg := LoadConfig()
	if cfg.Temperature != 0.2 {
		t.Errorf("expected Temperature 0.2, got %v", cfg.Temperature)
	}
}

func TestLoadConfig_MaxTokensOverride(t *testing.T) {
	t.Setenv("FLIP_AI_MAX_TOKENS", "512")

	cfg := LoadConfig()
	if cfg.MaxTokens != 512 {
		t.Errorf("expected MaxTokens 512, got %d", cfg.MaxTokens)
	}
}

func TestLoadConfig_InvalidTemperature_KeepsDefault(t *testing.T) {
	t.Setenv("FLIP_AI_TEMPERATURE", "not-a-number")

	cfg := LoadConfig()
	if cfg.Temperature != 0.7 {
		t.Errorf("invalid temperature should keep default 0.7, got %v", cfg.Temperature)
	}
}

func TestLoadConfig_OllamaModelOverride(t *testing.T) {
	t.Setenv("FLIP_OLLAMA_MODEL", "mistral")

	cfg := LoadConfig()
	if cfg.OllamaModel != "mistral" {
		t.Errorf("expected OllamaModel 'mistral', got %q", cfg.OllamaModel)
	}
}

func TestValidate_Ollama_Default(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("default Ollama config should be valid, got: %v", err)
	}
}

func TestValidate_Ollama_MissingHost(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OllamaHost = ""

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing OllamaHost")
	}
}

func TestValidate_OpenAI_MissingKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "openai"
	cfg.OpenAIKey = ""

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing OpenAI API key")
	}
}

func TestValidate_OpenAI_WithKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "openai"
	cfg.OpenAIKey = "sk-test-key"

	if err := cfg.Validate(); err != nil {
		t.Errorf("valid OpenAI config should pass, got: %v", err)
	}
}

func TestValidate_Anthropic_MissingKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "anthropic"
	cfg.AnthropicKey = ""

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing Anthropic API key")
	}
}

func TestValidate_Groq_MissingKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "groq"
	cfg.GroqKey = ""

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing Groq API key")
	}
}

func TestValidate_Azure_MissingEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "azure"
	cfg.AzureKey = "key-but-no-endpoint"
	cfg.AzureEndpoint = ""

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing Azure endpoint")
	}
}

func TestValidate_UnknownProvider(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "unknown-provider"

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestHasAPIKey_Ollama(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.HasAPIKey() {
		t.Error("Ollama should always report HasAPIKey = true")
	}
}

func TestHasAPIKey_OpenAI_Missing(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "openai"
	cfg.OpenAIKey = ""

	if cfg.HasAPIKey() {
		t.Error("OpenAI without key should return HasAPIKey = false")
	}
}

func TestHasAPIKey_OpenAI_Present(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "openai"
	cfg.OpenAIKey = "sk-anything"

	if !cfg.HasAPIKey() {
		t.Error("OpenAI with key should return HasAPIKey = true")
	}
}

func TestIsEnabled_DefaultTrue(t *testing.T) {
	os.Unsetenv("FLIP_FEATURE_AI")

	if !IsEnabled() {
		t.Error("AI should be enabled by default")
	}
}

func TestIsEnabled_DisabledByEnv(t *testing.T) {
	t.Setenv("FLIP_FEATURE_AI", "0")

	if IsEnabled() {
		t.Error("AI should be disabled when FLIP_FEATURE_AI=0")
	}
}

func TestIsEnabled_DisabledByFalse(t *testing.T) {
	t.Setenv("FLIP_FEATURE_AI", "false")

	if IsEnabled() {
		t.Error("AI should be disabled when FLIP_FEATURE_AI=false")
	}
}

func TestIsEnabled_ExplicitlyEnabled(t *testing.T) {
	t.Setenv("FLIP_FEATURE_AI", "1")

	if !IsEnabled() {
		t.Error("AI should be enabled when FLIP_FEATURE_AI=1")
	}
}
