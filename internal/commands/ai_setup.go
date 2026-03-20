package commands

// ai_setup.go - AI setup, configuration, and key management commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/httrp/flip/internal/ai"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// newAISetupCommand prepares local AI setup (Ollama + default model)
func newAISetupCommand() *cobra.Command {
	var (
		provider      string
		model         string
		apiKey        string
		host          string
		installOllama bool
		skipCheck     bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up AI provider and default model",
		Long: `Set up AI providers and default models from flip.

This command can:
1) Configure provider + default model instructions for all providers
2) Validate provider availability/configuration
3) For Ollama: optionally install Ollama and pull model

Examples:
  flip ai setup
  flip ai setup --provider openai --model gpt-4o-mini
  flip ai setup --provider anthropic --model claude-3-5-haiku-20241022
  flip ai setup --provider groq --model llama-3.3-70b-versatile
  flip ai setup --provider mistral --model mistral-large-latest
  flip ai setup --provider azure --model my-deployment
  flip ai setup --model llama3.2
  flip ai setup --install-ollama
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAISetup(provider, model, apiKey, host, installOllama, skipCheck)
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "AI provider (ollama, openai, anthropic, groq, mistral, azure)")
	cmd.Flags().StringVar(&model, "model", "", "Model to install (default: configured AI model)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key override for cloud providers (not persisted)")
	cmd.Flags().StringVar(&host, "host", "", "Ollama host override (default: OLLAMA_HOST or http://localhost:11434)")
	cmd.Flags().BoolVar(&installOllama, "install-ollama", false, "Install Ollama if not found in PATH")
	cmd.Flags().BoolVar(&skipCheck, "skip-check", false, "Skip provider connectivity/config check")

	return cmd
}

func runAISetup(provider, model, apiKey, host string, installOllama bool, skipCheck bool) error {
	cfg := ai.LoadConfig()

	targetProvider := strings.ToLower(strings.TrimSpace(provider))
	if targetProvider == "" {
		targetProvider = strings.ToLower(strings.TrimSpace(cfg.Provider))
	}
	if targetProvider == "" {
		targetProvider = "ollama"
	}

	if !isSupportedAIProvider(targetProvider) {
		return fmt.Errorf("unsupported provider: %s (supported: ollama, openai, anthropic, groq, mistral, azure)", targetProvider)
	}

	targetModel := strings.TrimSpace(model)
	if targetModel == "" {
		targetModel = defaultModelForProvider(cfg, targetProvider)
	}

	if strings.TrimSpace(host) != "" {
		cfg.OllamaHost = strings.TrimSpace(host)
	}

	if strings.TrimSpace(apiKey) != "" {
		setProviderAPIKey(cfg, targetProvider, strings.TrimSpace(apiKey))
	}

	fmt.Println()
	fmt.Println("🤖 AI Setup")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Provider: %s\n", targetProvider)
	fmt.Printf("Model:    %s\n", targetModel)
	if targetProvider == "ollama" {
		fmt.Printf("Host:     %s\n", cfg.OllamaHost)
	}

	setupCfg := buildSetupConfig(cfg, targetProvider, targetModel)

	if targetProvider == "ollama" {
		if err := ensureOllamaModelInstalled(targetModel, installOllama); err != nil {
			return err
		}
	}

	if !skipCheck {
		if err := checkAIProviderAvailability(setupCfg); err != nil {
			return err
		}
	}

	// Persist configuration to config file
	if err := ai.SaveConfig(setupCfg); err != nil {
		fmt.Printf("⚠️  Could not save config: %v\n", err)
		fmt.Println("   You can set the environment variables manually:")
		printAISetupEnvInstructions(setupCfg)
	} else {
		fmt.Printf("\n✅ Configuration saved to %s\n", ai.ConfigFilePath())
	}

	// Validate API key if this is a cloud provider
	if targetProvider != "ollama" {
		fmt.Println("\n🔑 Validating API key...")
		valCtx, valCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer valCancel()
		keyStatus := ai.ValidateAPIKey(valCtx, setupCfg)
		if keyStatus.Valid {
			fmt.Printf("   %s\n", keyStatus.Message)
		} else {
			fmt.Printf("   ⚠️  %s\n", keyStatus.Message)
			fmt.Println("   Check your API key and try again with: flip ai setup --provider", targetProvider)
		}
	}

	fmt.Println("✅ AI setup complete.")
	return nil
}

func ensureOllamaModelInstalled(targetModel string, installOllama bool) error {
	if strings.TrimSpace(targetModel) == "" {
		targetModel = "llama3.2"
	}

	if _, err := exec.LookPath("ollama"); err != nil {
		if !installOllama {
			return fmt.Errorf("Ollama not found. Install it manually or run `flip ai setup --install-ollama`")
		}

		if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
			return fmt.Errorf("automatic Ollama installation is only supported on Linux/macOS")
		}

		fmt.Println("\n📦 Installing Ollama...")
		installCmd := exec.Command("sh", "-c", "curl -fsSL https://ollama.com/install.sh | sh")
		installCmd.Stdin = os.Stdin
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install Ollama: %w", err)
		}
	}

	fmt.Println("\n🔍 Checking Ollama service...")
	listCmd := exec.Command("ollama", "list")
	listOutput, err := listCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ollama is installed but not ready: %w\nOutput: %s", err, strings.TrimSpace(string(listOutput)))
	}

	if ollamaListContainsModel(string(listOutput), targetModel) {
		fmt.Printf("✅ Model already installed: %s\n", targetModel)
		return nil
	}

	fmt.Printf("\n⬇️  Pulling model: %s\n", targetModel)
	pullCmd := exec.Command("ollama", "pull", targetModel)
	pullCmd.Stdin = os.Stdin
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull model '%s': %w", targetModel, err)
	}

	verifyCmd := exec.Command("ollama", "list")
	verifyOutput, err := verifyCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("model pull finished, but verification failed: %w", err)
	}

	if !ollamaListContainsModel(string(verifyOutput), targetModel) {
		return fmt.Errorf("model '%s' not found after pull; run `ollama list` to verify", targetModel)
	}

	fmt.Printf("✅ Installed model: %s\n", targetModel)
	return nil
}

func checkAIProviderAvailability(cfg *ai.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ai.NewClientWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	if client.IsAvailable(ctx) {
		fmt.Println("Status:   ✅ Available")
		return nil
	}

	fmt.Println("Status:   ⚠️  Not reachable right now")
	fmt.Println("          Config is set, but provider check failed.")
	return nil
}

func printAISetupEnvInstructions(cfg *ai.Config) {
	fmt.Println()
	fmt.Println("Environment setup:")
	fmt.Printf("  export FLIP_AI_PROVIDER=%s\n", cfg.Provider)

	switch cfg.Provider {
	case "ollama":
		if cfg.OllamaHost != "" {
			fmt.Printf("  export OLLAMA_HOST=%s\n", cfg.OllamaHost)
		}
		if cfg.OllamaModel != "" {
			fmt.Printf("  export FLIP_OLLAMA_MODEL=%s\n", cfg.OllamaModel)
		}
	case "openai":
		fmt.Println("  export OPENAI_API_KEY=<YOUR_OPENAI_KEY>")
		if cfg.OpenAIModel != "" {
			fmt.Printf("  export FLIP_OPENAI_MODEL=%s\n", cfg.OpenAIModel)
		}
	case "anthropic":
		fmt.Println("  export ANTHROPIC_API_KEY=<YOUR_ANTHROPIC_KEY>")
		if cfg.AnthropicModel != "" {
			fmt.Printf("  export FLIP_ANTHROPIC_MODEL=%s\n", cfg.AnthropicModel)
		}
	case "groq":
		fmt.Println("  export GROQ_API_KEY=<YOUR_GROQ_KEY>")
		if cfg.GroqModel != "" {
			fmt.Printf("  export FLIP_GROQ_MODEL=%s\n", cfg.GroqModel)
		}
	case "mistral":
		fmt.Println("  export MISTRAL_API_KEY=<YOUR_MISTRAL_KEY>")
		if cfg.MistralModel != "" {
			fmt.Printf("  export FLIP_MISTRAL_MODEL=%s\n", cfg.MistralModel)
		}
	case "azure":
		fmt.Println("  export AZURE_OPENAI_ENDPOINT=<YOUR_AZURE_ENDPOINT>")
		fmt.Println("  export AZURE_OPENAI_KEY=<YOUR_AZURE_KEY>")
		if cfg.AzureDeployment != "" {
			fmt.Printf("  export AZURE_OPENAI_DEPLOYMENT=%s\n", cfg.AzureDeployment)
		}
		if cfg.AzureAPIVersion != "" {
			fmt.Printf("  export AZURE_OPENAI_API_VERSION=%s\n", cfg.AzureAPIVersion)
		}
	}

	fmt.Println()
	fmt.Println("Tip: Add these exports to your shell profile (~/.bashrc or ~/.zshrc).")
}

func buildSetupConfig(base *ai.Config, provider, model string) *ai.Config {
	cfg := *base
	cfg.Provider = provider

	switch provider {
	case "ollama":
		cfg.OllamaModel = model
	case "openai":
		cfg.OpenAIModel = model
	case "anthropic":
		cfg.AnthropicModel = model
	case "groq":
		cfg.GroqModel = model
	case "mistral":
		cfg.MistralModel = model
	case "azure":
		cfg.AzureDeployment = model
	}

	// Recompute derived fields
	switch provider {
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
	default:
		cfg.Model = cfg.OllamaModel
		cfg.BaseURL = cfg.OllamaHost
	}

	return &cfg
}

func setProviderAPIKey(cfg *ai.Config, provider, key string) {
	switch provider {
	case "openai":
		cfg.OpenAIKey = key
	case "anthropic":
		cfg.AnthropicKey = key
	case "groq":
		cfg.GroqKey = key
	case "mistral":
		cfg.MistralKey = key
	case "azure":
		cfg.AzureKey = key
	}
}

// newAIRenewKeyCommand allows interactive renewal of API keys
func newAIRenewKeyCommand() *cobra.Command {
	var provider string

	cmd := &cobra.Command{
		Use:   "renew-key",
		Short: "Update or renew an API key for a provider",
		Long: `Interactively update the API key for an AI provider.

Lets you select a provider, enter a new key, validates it immediately,
and saves it to the config file.

Examples:
  flip ai renew-key
  flip ai renew-key --provider mistral`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAIRenewKey(provider)
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "Provider to renew key for (skip selection)")

	return cmd
}

func runAIRenewKey(targetProvider string) error {
	cfg := ai.LoadConfig()

	// Cloud providers that use API keys
	type keyProvider struct {
		name   string
		hasKey bool
		masked string
	}

	cloudProviders := []keyProvider{
		{name: "openai", hasKey: cfg.OpenAIKey != "", masked: ai.MaskAPIKey(cfg.OpenAIKey)},
		{name: "anthropic", hasKey: cfg.AnthropicKey != "", masked: ai.MaskAPIKey(cfg.AnthropicKey)},
		{name: "groq", hasKey: cfg.GroqKey != "", masked: ai.MaskAPIKey(cfg.GroqKey)},
		{name: "mistral", hasKey: cfg.MistralKey != "", masked: ai.MaskAPIKey(cfg.MistralKey)},
		{name: "azure", hasKey: cfg.AzureKey != "", masked: ai.MaskAPIKey(cfg.AzureKey)},
	}

	if targetProvider == "" {
		// Interactive selection
		var items []string
		for _, p := range cloudProviders {
			label := p.name
			if p.hasKey {
				label += fmt.Sprintf(" (current: %s)", p.masked)
			} else {
				label += " (not configured)"
			}
			items = append(items, label)
		}

		sel := promptui.Select{
			Label: "Select provider to update API key",
			Items: items,
		}
		idx, _, err := sel.Run()
		if err != nil {
			return err
		}
		targetProvider = cloudProviders[idx].name
	}

	if !isSupportedAIProvider(targetProvider) || targetProvider == "ollama" {
		return fmt.Errorf("provider '%s' does not use API keys", targetProvider)
	}

	fmt.Printf("\n🔑 Update API key for: %s\n", targetProvider)

	// Find current key
	for _, p := range cloudProviders {
		if p.name == targetProvider && p.hasKey {
			fmt.Printf("   Current key: %s\n", p.masked)
		}
	}

	prompt := promptui.Prompt{
		Label: "New API key",
		Mask:  '*',
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("key cannot be empty")
			}
			return nil
		},
	}
	newKey, err := prompt.Run()
	if err != nil {
		return err
	}
	newKey = strings.TrimSpace(newKey)

	// Set the new key in config
	setProviderAPIKey(cfg, targetProvider, newKey)

	// Validate the new key immediately
	fmt.Println("\n🔍 Validating new key...")
	valCtx, valCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer valCancel()

	valCfg := buildSetupConfig(cfg, targetProvider, defaultModelForProvider(cfg, targetProvider))
	setProviderAPIKey(valCfg, targetProvider, newKey)
	keyStatus := ai.ValidateAPIKey(valCtx, valCfg)

	if !keyStatus.Valid {
		fmt.Printf("   ❌ %s\n", keyStatus.Message)
		fmt.Println()

		confirmPrompt := promptui.Prompt{
			Label:     "Save this key anyway",
			IsConfirm: true,
		}
		_, err := confirmPrompt.Run()
		if err != nil {
			fmt.Println("   Key not saved.")
			return nil
		}
	} else {
		fmt.Printf("   ✅ %s\n", keyStatus.Message)
	}

	// Save to config file
	if err := ai.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\n✅ API key for %s updated and saved.\n", targetProvider)
	return nil
}

func defaultModelForProvider(cfg *ai.Config, provider string) string {
	switch provider {
	case "openai":
		if strings.TrimSpace(cfg.OpenAIModel) != "" {
			return strings.TrimSpace(cfg.OpenAIModel)
		}
		return "gpt-4o-mini"
	case "anthropic":
		if strings.TrimSpace(cfg.AnthropicModel) != "" {
			return strings.TrimSpace(cfg.AnthropicModel)
		}
		return "claude-3-5-haiku-20241022"
	case "groq":
		if strings.TrimSpace(cfg.GroqModel) != "" {
			return strings.TrimSpace(cfg.GroqModel)
		}
		return "llama-3.3-70b-versatile"
	case "mistral":
		if strings.TrimSpace(cfg.MistralModel) != "" {
			return strings.TrimSpace(cfg.MistralModel)
		}
		return "mistral-large-latest"
	case "azure":
		if strings.TrimSpace(cfg.AzureDeployment) != "" {
			return strings.TrimSpace(cfg.AzureDeployment)
		}
		return "my-azure-deployment"
	default:
		if strings.TrimSpace(cfg.OllamaModel) != "" {
			return strings.TrimSpace(cfg.OllamaModel)
		}
		return "llama3.2"
	}
}

func isSupportedAIProvider(provider string) bool {
	switch provider {
	case "ollama", "openai", "anthropic", "groq", "mistral", "azure":
		return true
	default:
		return false
	}
}

func ollamaListContainsModel(listOutput string, model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return false
	}

	for _, line := range strings.Split(listOutput, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(strings.ToUpper(trimmed), "NAME") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		name := strings.ToLower(fields[0])
		if name == model || strings.HasPrefix(name, model+":") {
			return true
		}
	}

	return false
}
