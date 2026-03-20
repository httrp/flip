package commands

// ai.go - AI Integration Commands
//
// Provides AI-powered features for flip:
//   - flip ai summarize: Summarize notes about a topic from your brains
//   - flip ai research: Create a research note on a topic using AI
//   - flip ai improve: Improve/rewrite a note with AI assistance
//   - flip ai status: Check AI provider status
//
// Configuration via environment variables:
//   FLIP_AI_PROVIDER: ollama (default), openai, anthropic, azure
//   FLIP_FEATURE_AI: 0 to disable AI features

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/httrp/flip/internal/ai"
	"github.com/httrp/flip/internal/brain"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// NewAICommand creates the AI command group
func NewAICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "AI-powered features",
		Long: `AI-powered features for note-taking and knowledge management.

flip uses AI to help you:
  - Summarize your notes about a topic
  - Research and create new notes
  - Improve existing notes

Configuration:
  FLIP_AI_PROVIDER=ollama|openai|anthropic|groq|mistral|azure
  OLLAMA_HOST=http://localhost:11434 (for Ollama)
  OPENAI_API_KEY=sk-... (for OpenAI)
  ANTHROPIC_API_KEY=sk-ant-... (for Anthropic)
  GROQ_API_KEY=gsk_... (for Groq - fast & free)
  MISTRAL_API_KEY=... (for Mistral)`,
	}

	cmd.AddCommand(newAISummarizeCommand())
	cmd.AddCommand(newAIResearchCommand())
	cmd.AddCommand(newAIImproveCommand())
	cmd.AddCommand(newAIStatusCommand())
	cmd.AddCommand(newAIModelsCommand())
	cmd.AddCommand(newAISetupCommand())
	cmd.AddCommand(newAIRenewKeyCommand())

	return cmd
}

// newAIStatusCommand shows AI provider status
func newAIStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check AI provider status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAIStatus()
		},
	}
}

func runAIStatus() error {
	fmt.Println()
	fmt.Println("🤖 AI Provider Status")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	cfg := ai.LoadConfig()

	// Show config file location
	configPath := ai.ConfigFilePath()
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config:   %s\n", configPath)
		}
	}
	fmt.Printf("Active:   %s (%s)\n", cfg.Provider, cfg.Model)
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Build a list of all providers and their status
	type providerInfo struct {
		name      string
		model     string
		apiKey    string
		hasKey    bool
		available bool
		keyStatus *ai.KeyStatus
	}

	providers := []providerInfo{
		{name: "ollama", model: cfg.OllamaModel, hasKey: true},
		{name: "openai", model: cfg.OpenAIModel, apiKey: cfg.OpenAIKey, hasKey: cfg.OpenAIKey != ""},
		{name: "anthropic", model: cfg.AnthropicModel, apiKey: cfg.AnthropicKey, hasKey: cfg.AnthropicKey != ""},
		{name: "groq", model: cfg.GroqModel, apiKey: cfg.GroqKey, hasKey: cfg.GroqKey != ""},
		{name: "mistral", model: cfg.MistralModel, apiKey: cfg.MistralKey, hasKey: cfg.MistralKey != ""},
		{name: "azure", model: cfg.AzureDeployment, apiKey: cfg.AzureKey, hasKey: cfg.AzureKey != ""},
	}

	// Check availability and validate keys for configured providers
	for i, p := range providers {
		if !p.hasKey {
			continue
		}

		checkCfg := buildSetupConfig(cfg, p.name, p.model)
		client, err := ai.NewClientWithConfig(checkCfg)
		if err == nil {
			providers[i].available = client.IsAvailable(ctx)
		}

		// Validate API key for cloud providers
		if p.name != "ollama" {
			providers[i].keyStatus = ai.ValidateAPIKey(ctx, checkCfg)
		}
	}

	// Display all providers
	for _, p := range providers {
		var status string
		if !p.hasKey {
			status = "—  not configured"
		} else if p.keyStatus != nil && !p.keyStatus.Valid {
			status = "❌ key invalid"
		} else if p.available {
			status = "✅ available"
		} else {
			status = "⚠️  not reachable"
		}

		active := " "
		if p.name == cfg.Provider {
			active = "▸"
		}

		modelStr := ""
		if p.hasKey && p.model != "" {
			modelStr = fmt.Sprintf(" (%s)", p.model)
		}

		keyInfo := ""
		if p.hasKey && p.apiKey != "" {
			keyInfo = fmt.Sprintf("  key: %s", ai.MaskAPIKey(p.apiKey))
			if p.keyStatus != nil && p.keyStatus.HasExpiry() {
				keyInfo += fmt.Sprintf(" [%s]", p.keyStatus.ExpiryString())
			}
		}

		fmt.Printf(" %s %-10s %s%s%s\n", active, p.name, status, modelStr, keyInfo)
	}

	fmt.Println()

	// Show invalid key warnings
	hasInvalidKey := false
	for _, p := range providers {
		if p.keyStatus != nil && !p.keyStatus.Valid {
			if !hasInvalidKey {
				fmt.Println("⚠️  Key issues detected:")
				hasInvalidKey = true
			}
			fmt.Printf("   • %s: %s\n", p.name, p.keyStatus.Message)
		}
	}
	if hasInvalidKey {
		fmt.Println()
		fmt.Println("   Update a key with: flip ai setup --provider <name> --api-key <new-key>")
		fmt.Println("   Or interactively:  flip ai renew-key")
		fmt.Println()
	}

	// Show hint if no cloud providers configured
	hasCloudProvider := false
	for _, p := range providers {
		if p.name != "ollama" && p.hasKey {
			hasCloudProvider = true
			break
		}
	}
	if !hasCloudProvider {
		fmt.Println("Tip: Run 'flip ai setup' to configure additional providers.")
	}

	return nil
}

type aiModelsResponse struct {
	Provider     string         `json:"provider"`
	DefaultModel string         `json:"default_model"`
	Models       []ai.ModelInfo `json:"models"`
}

// newAIModelsCommand lists available models for the current provider
func newAIModelsCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "models",
		Short: "List available AI models",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			return runAIModels()
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON (for VS Code integration)")

	return cmd
}

func runAIModels() error {
	cfg := ai.LoadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ai.NewClient()
	if err != nil {
		if JSONOutput {
			OutputJSONError("ai models", err)
			return nil
		}
		return err
	}

	models, err := client.ListModels(ctx)
	if err != nil {
		if JSONOutput {
			OutputJSONError("ai models", err)
			return nil
		}
		return err
	}

	resp := aiModelsResponse{
		Provider:     cfg.Provider,
		DefaultModel: cfg.Model,
		Models:       models,
	}

	if JSONOutput {
		OutputJSONSuccess("ai models", resp)
		return nil
	}

	fmt.Println("\n🤖 Available Models")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Provider: %s\n", cfg.Provider)
	fmt.Printf("Default:  %s\n\n", cfg.Model)

	if len(models) == 0 {
		fmt.Println("No models found.")
		if cfg.Provider == "ollama" {
			fmt.Println("Hint: run `flip ai setup` to install a default model.")
		}
		return nil
	}

	for _, m := range models {
		label := m.ID
		if m.Name != "" && m.Name != m.ID {
			label = fmt.Sprintf("%s (%s)", m.Name, m.ID)
		}
		fmt.Printf("- %s\n", label)
		if m.Description != "" {
			fmt.Printf("  %s\n", m.Description)
		}
	}

	return nil
}

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

// newAISummarizeCommand creates notes summarizing content from brains
func newAISummarizeCommand() *cobra.Command {
	var (
		topic               string
		brains              []string
		output              string
		brain               string
		title               string
		model               string
		prompt              string
		promptExtra         string
		contextFiles        []string
		promptCreateTitle   string
		promptCreateBody    string
		promptCreateDefault bool
		timeout             int
		link                bool
		noLink              bool
		noStream            bool
		jsonOut             bool
		prepareOnly         bool
	)

	cmd := &cobra.Command{
		Use:          "summarize [topic]",
		SilenceUsage: true,
		Short:        "Summarize your notes about a topic",
		Long: `Search your brains for notes about a topic and create a summary.

This command:
1. Searches your brains for notes matching the topic
2. Sends the relevant content to your AI provider
3. Creates a new note with the AI-generated summary

Examples:
  flip ai summarize "project management"
  flip ai summarize "golang best practices" --brains danobrain
  flip ai summarize "meeting notes Q1" -o summary.md`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				topic = args[0]
			}
			JSONOutput = jsonOut
			return runAISummarize(topic, brains, output, brain, title, model, prompt, promptExtra, contextFiles, promptCreateTitle, promptCreateBody, promptCreateDefault, timeout, link, noLink, !noStream, prepareOnly)
		},
	}

	cmd.Flags().StringSliceVarP(&brains, "brains", "b", nil, "Specific brains to search (default: all)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: auto-generated)")
	cmd.Flags().StringVar(&brain, "brain", "", "Target brain to save the output (skip selection prompt)")
	cmd.Flags().StringVar(&title, "title", "", "Custom title for the generated note")
	cmd.Flags().StringVar(&model, "model", "", "Override model for this request")
	cmd.Flags().StringVar(&topic, "request", "", "Specific request for this run (alias of topic argument)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt note to apply (path or name)")
	cmd.Flags().StringVar(&prompt, "template", "", "Template note to apply (alias of --prompt)")
	cmd.Flags().StringVar(&promptExtra, "prompt-extra", "", "Additional prompt instructions for this request")
	cmd.Flags().StringVar(&promptExtra, "run-notes", "", "Additional run notes for this request (alias of --prompt-extra)")
	cmd.Flags().StringSliceVar(&contextFiles, "context-file", nil, "Additional context file(s) to include (repeat flag)")
	cmd.Flags().StringVar(&promptCreateTitle, "prompt-create-title", "", "Create a prompt note with this title")
	cmd.Flags().StringVar(&promptCreateTitle, "template-create-title", "", "Create a template note with this title (alias of --prompt-create-title)")
	cmd.Flags().StringVar(&promptCreateBody, "prompt-create-body", "", "Prompt body used when creating a prompt note")
	cmd.Flags().StringVar(&promptCreateBody, "template-create-body", "", "Template body used when creating a template note (alias of --prompt-create-body)")
	cmd.Flags().BoolVar(&promptCreateDefault, "prompt-create-default", false, "Mark created prompt note as default")
	cmd.Flags().BoolVar(&promptCreateDefault, "template-create-default", false, "Mark created template note as default (alias of --prompt-create-default)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Override AI timeout in seconds")
	cmd.Flags().BoolVar(&link, "link", false, "Always add link to today's journal without prompting")
	cmd.Flags().BoolVar(&noLink, "no-link", false, "Do not add a link to today's journal")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON (for VS Code integration)")
	cmd.Flags().BoolVar(&prepareOnly, "prepare-only", false, "Return prompts as JSON without calling AI (for Copilot integration)")

	return cmd
}

type summarySourceNote struct {
	BrainName string
	BrainPath string
	NotePath  string
	RelPath   string
	Title     string
}

func runAISummarize(topic string, brainNames []string, outputPath string, brainName string, title string, modelOverride string, promptOverride string, promptExtra string, contextFiles []string, promptCreateTitle string, promptCreateBody string, promptCreateDefault bool, timeoutSec int, link bool, noLink bool, stream bool, prepareOnly bool) error {
	// Get active workspace first
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if len(activeWs.Brains) == 0 {
		fmt.Println("⚠️  No brains found in active workspace. Please add a brain first.")
		return nil
	}

	// STEP 1: Select target brain for output
	var targetBrain *Brain
	if brainName != "" {
		for i := range activeWs.Brains {
			if activeWs.Brains[i].Name == brainName {
				targetBrain = &activeWs.Brains[i]
				break
			}
		}
		if targetBrain == nil {
			return fmt.Errorf("brain not found: %s", brainName)
		}
	} else {
		targetBrain, err = confirmOrSelectBrain(activeWs)
		if err != nil {
			return err
		}
	}

	// Detect brain type
	detector := brain.NewDetector()
	detection, _ := detector.DetectBrainType(targetBrain.Path)

	promptContent, promptNote, err := resolvePromptNoteForBrain(targetBrain, detection.Type, promptOverride, promptCreateTitle, promptCreateBody, promptCreateDefault, !JSONOutput)
	if err != nil {
		return err
	}

	// Get topic interactively if not provided
	if topic == "" {
		prompt := promptui.Prompt{
			Label: "What should be summarized? (request)",
		}
		topic, err = prompt.Run()
		if err != nil {
			return err
		}
		if topic == "" {
			return fmt.Errorf("topic is required")
		}
	}

	if !prepareOnly {
		PrintOrJSON("\n🔍 Searching for notes about: %s\n", topic)
	}

	var searchBrains []Brain

	if len(brainNames) > 0 {
		// Use specified brains
		for _, name := range brainNames {
			for _, b := range activeWs.Brains {
				if b.Name == name {
					searchBrains = append(searchBrains, b)
					break
				}
			}
		}
	} else {
		// Use all brains in active workspace
		searchBrains = activeWs.Brains
	}

	if len(searchBrains) == 0 {
		return fmt.Errorf("no brains to search")
	}

	// Search for matching notes
	var relevantContent strings.Builder
	var noteCount int
	var sourceNotes []summarySourceNote
	sourceSeen := make(map[string]bool)

	for _, brain := range searchBrains {
		notes, err := searchNotesInBrain(brain.Path, topic)
		if err != nil {
			continue
		}

		for _, note := range notes {
			content, err := os.ReadFile(note)
			if err != nil {
				continue
			}

			// Add note content with header
			relPath, _ := filepath.Rel(brain.Path, note)
			relevantContent.WriteString(fmt.Sprintf("\n--- %s/%s ---\n", brain.Name, relPath))
			relevantContent.WriteString(string(content))
			relevantContent.WriteString("\n")
			noteCount++

			key := brain.Name + "|" + relPath
			if !sourceSeen[key] {
				sourceSeen[key] = true
				titleFromFile := strings.TrimSuffix(filepath.Base(note), filepath.Ext(note))
				sourceNotes = append(sourceNotes, summarySourceNote{
					BrainName: brain.Name,
					BrainPath: brain.Path,
					NotePath:  note,
					RelPath:   relPath,
					Title:     titleFromFile,
				})
			}

			// Limit to avoid token overflow
			if relevantContent.Len() > 50000 {
				break
			}
		}
	}

	// Add explicit context files
	contextNotes, contextContent, err := loadContextFiles(contextFiles, targetBrain)
	if err != nil {
		return err
	}
	if strings.TrimSpace(contextContent) != "" {
		relevantContent.WriteString("\n\n--- Additional context files ---\n")
		relevantContent.WriteString(contextContent)
		relevantContent.WriteString("\n")
		noteCount += len(contextNotes)
		sourceNotes = append(sourceNotes, contextNotes...)
	}

	if noteCount == 0 {
		PrintlnOrJSON("❌ No notes found matching the topic")
		return nil
	}

	if !prepareOnly {
		PrintOrJSON("📄 Found %d relevant notes\n\n", noteCount)
	}

	// Create AI client (allow per-request timeout override) — skip for prepare-only
	cfg := ai.LoadConfig()
	if timeoutSec > 0 {
		cfg.Timeout = timeoutSec
	}

	var client *ai.Client
	var effectiveModel string
	ctx := context.Background()

	if prepareOnly {
		effectiveModel = modelOverride
	} else {
		var cErr error
		client, cErr = ai.NewClientWithConfig(cfg)
		if cErr != nil {
			return fmt.Errorf("failed to create AI client: %w", cErr)
		}
		var mErr error
		effectiveModel, mErr = resolveAIModelForRequest(ctx, client, cfg, modelOverride)
		if mErr != nil {
			return mErr
		}
	}

	// Build the prompt
	systemPrompt := `You are a helpful assistant that creates clear summaries of notes.
Given a collection of notes about a topic, create a well-structured summary that:
- Highlights the key points and insights
- Groups related information together
- Uses clear headings and bullet points
- Preserves important details and references
- Is written in markdown format

Keep the summary concise but comprehensive.`

	systemPrompt = applyPromptStack(systemPrompt, promptContent, promptExtra)

	userPrompt := fmt.Sprintf("Please summarize the following notes about \"%s\":\n\n%s", topic, relevantContent.String())

	// Resolve title, filename, output path early (needed for prepare-only)
	noteDate := time.Now().Format("2006-01-02")
	noteDir := filepath.Join(targetBrain.Path, "notes")
	interactiveTitle := !JSONOutput && strings.TrimSpace(title) == ""
	noteTitle, filename, err := resolveAITitleAndFilename(topic, title, noteDir, interactiveTitle)
	if err != nil {
		return err
	}
	if outputPath == "" {
		outputPath = filepath.Join(targetBrain.Path, "notes", filename)
	} else if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(targetBrain.Path, "notes", outputPath)
	}

	// --prepare-only: return prompts + metadata as JSON so the extension can use Copilot
	if prepareOnly {
		promptSection := buildPromptSection(outputPath, targetBrain, promptNote, promptExtra, true)
		sourcesSection := buildSummarySourcesSection(outputPath, targetBrain, sourceNotes, true)

		usedModel := effectiveModel
		if usedModel == "" {
			usedModel = cfg.Model
		}
		promptMeta := ""
		if promptNote != nil {
			relPrompt := relativePathFromBrain(promptNote.Path, targetBrain.Path)
			promptMeta = fmt.Sprintf("prompt_note: \"%s\"\n", filepath.ToSlash(relPrompt))
		}
		if strings.TrimSpace(promptExtra) != "" {
			promptMeta += "prompt_extra: true\n"
		}
		frontmatter := fmt.Sprintf("---\ntitle: \"%s\"\ndate: %s\ntype: summary\ntopic: \"%s\"\nsources: %d notes\nai_generated: true\nai_action: summary\nai_created_at: \"%s\"\nai_provider: \"%s\"\nai_model: \"%s\"\n%s---\n\n",
			noteTitle, noteDate, topic, noteCount, time.Now().Format(time.RFC3339), "__PROVIDER__", "__MODEL__", promptMeta)

		result := map[string]interface{}{
			"system_prompt":   systemPrompt,
			"user_prompt":     userPrompt,
			"output_path":     outputPath,
			"frontmatter":     frontmatter,
			"prompt_section":  promptSection,
			"sources_section": sourcesSection,
			"brain_name":      targetBrain.Name,
			"brain_path":      targetBrain.Path,
			"title":           noteTitle,
			"topic":           topic,
			"source_count":    noteCount,
		}
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal prepare-only result: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	req := &ai.CompletionRequest{
		System: systemPrompt,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: userPrompt},
		},
		Model:       effectiveModel,
		Temperature: 0.3, // Lower temperature for summarization
		MaxTokens:   4096,
	}

	PrintlnOrJSON("🤖 Generating summary...")
	PrintlnOrJSON()

	var result strings.Builder

	usedModel := effectiveModel

	if stream {
		// Stream the response
		streamResp, err := client.CompleteStream(ctx, req)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}
		defer streamResp.Close()

		for {
			chunk, err := streamResp.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("stream error: %w", err)
			}
			if !JSONOutput {
				fmt.Print(chunk)
			}
			result.WriteString(chunk)
		}
		if !JSONOutput {
			fmt.Println()
		}
	} else {
		// Non-streaming response
		resp, err := client.Complete(ctx, req)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}
		if !JSONOutput {
			fmt.Println(resp.Content)
		}
		result.WriteString(resp.Content)
		if resp.Model != "" {
			usedModel = resp.Model
		}
	}

	// Save to file — title, filename, outputPath already resolved above

	// Get AI config for metadata
	if usedModel == "" {
		usedModel = cfg.Model
	}

	promptMeta := ""
	if promptNote != nil {
		relPrompt := relativePathFromBrain(promptNote.Path, targetBrain.Path)
		promptMeta = fmt.Sprintf("prompt_note: \"%s\"\n", filepath.ToSlash(relPrompt))
	}
	if strings.TrimSpace(promptExtra) != "" {
		promptMeta += "prompt_extra: true\n"
	}

	// Create frontmatter with AI metadata
	frontmatter := fmt.Sprintf(`---
title: "%s"
date: %s
type: summary
topic: "%s"
sources: %d notes
ai_generated: true
ai_action: summary
ai_created_at: "%s"
ai_provider: "%s"
ai_model: "%s"
%s---

`, noteTitle, noteDate, topic, noteCount, time.Now().Format(time.RFC3339), cfg.Provider, usedModel, promptMeta)

	// Append sources/prompts section at the top
	promptSection := buildPromptSection(outputPath, targetBrain, promptNote, promptExtra, true)
	sourcesSection := buildSummarySourcesSection(outputPath, targetBrain, sourceNotes, true)
	finalContent := promptSection + sourcesSection + result.String()

	// Write file
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(frontmatter+finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if JSONOutput {
		OutputJSONSuccess("ai summarize", NoteResult{
			Action:    "created",
			Path:      outputPath,
			Title:     noteTitle,
			BrainName: targetBrain.Name,
			BrainPath: targetBrain.Path,
			BrainType: string(detection.Type),
		})
		return nil
	}

	fmt.Printf("\n\n✅ Summary saved to: %s\n", outputPath)

	// Add link to journal
	if !noLink {
		relPath := relativePathFromBrain(outputPath, targetBrain.Path)
		linkTitle := formatAILinkTitle(noteTitle, "Summary", cfg.Provider)
		if err := AddLinkToJournal(JournalLinkOptions{
			ItemType:    "note",
			ItemName:    linkTitle,
			ItemPath:    relPath,
			Brain:       targetBrain,
			Interactive: !link,
			BrainType:   detection.Type,
		}); err != nil {
			fmt.Printf("⚠️  Could not add journal link: %v\n", err)
		}
	}

	return nil
}

func buildSummarySourcesSection(summaryPath string, targetBrain *Brain, sources []summarySourceNote, force bool) string {
	if len(sources) == 0 && !force {
		return ""
	}

	lines := []string{"", "", "## Sources"}
	if len(sources) == 0 {
		lines = append(lines, "- None")
		return strings.Join(lines, "\n")
	}
	for _, source := range sources {
		label := source.Title
		if source.BrainName != "" && source.BrainName != targetBrain.Name {
			label = source.BrainName + ": " + label
		}

		linkPath := source.NotePath
		if source.BrainName == targetBrain.Name {
			rel, err := filepath.Rel(filepath.Dir(summaryPath), source.NotePath)
			if err == nil {
				linkPath = rel
			}
		}
		linkPath = filepath.ToSlash(linkPath)
		lines = append(lines, fmt.Sprintf("- [%s](%s)", label, linkPath))
	}

	return strings.Join(lines, "\n")
}

func buildPromptSection(notePath string, targetBrain *Brain, promptNote *PromptNoteInfo, promptExtra string, force bool) string {
	if promptNote == nil && strings.TrimSpace(promptExtra) == "" && !force {
		return ""
	}

	lines := []string{"", "", "## Template Context"}
	if promptNote != nil {
		label := promptNote.Title
		if label == "" {
			label = promptNote.Name
		}

		linkPath := promptNote.Path
		if targetBrain != nil && targetBrain.Path != "" {
			rel, err := filepath.Rel(filepath.Dir(notePath), promptNote.Path)
			if err == nil {
				linkPath = rel
			}
		}
		linkPath = filepath.ToSlash(linkPath)
		lines = append(lines, fmt.Sprintf("- Template note: [%s](%s)", label, linkPath))
	}

	extra := strings.TrimSpace(promptExtra)
	if extra != "" {
		lines = append(lines, "", "### Run Notes", "", extra)
	}
	if promptNote == nil && extra == "" {
		lines = append(lines, "- None")
	}

	return strings.Join(lines, "\n")
}

func loadContextFiles(filePaths []string, targetBrain *Brain) ([]summarySourceNote, string, error) {
	if len(filePaths) == 0 {
		return nil, "", nil
	}

	var notes []summarySourceNote
	var content strings.Builder
	seen := make(map[string]bool)

	for _, raw := range filePaths {
		candidate := strings.TrimSpace(raw)
		if candidate == "" {
			continue
		}

		abs := candidate
		if !filepath.IsAbs(abs) {
			resolved, err := filepath.Abs(abs)
			if err != nil {
				return nil, "", fmt.Errorf("failed to resolve context file '%s': %w", candidate, err)
			}
			abs = resolved
		}

		if seen[abs] {
			continue
		}
		seen[abs] = true

		fileContent, err := os.ReadFile(abs)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read context file '%s': %w", candidate, err)
		}

		content.WriteString(fmt.Sprintf("\n--- %s ---\n", filepath.Base(abs)))
		content.Write(fileContent)
		content.WriteString("\n")

		note := summarySourceNote{
			NotePath: abs,
			Title:    strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs)),
		}
		if targetBrain != nil {
			relPath, err := filepath.Rel(targetBrain.Path, abs)
			if err == nil && !strings.HasPrefix(relPath, "..") {
				note.BrainName = targetBrain.Name
				note.BrainPath = targetBrain.Path
				note.RelPath = relPath
			}
		}
		notes = append(notes, note)
	}

	return notes, strings.TrimSpace(content.String()), nil
}

func collectAutoContextFromBrain(targetBrain *Brain, request string) ([]summarySourceNote, string, error) {
	if targetBrain == nil {
		return nil, "", nil
	}

	notes, err := searchNotesInBrain(targetBrain.Path, request)
	if err != nil {
		return nil, "", err
	}
	if len(notes) == 0 {
		return nil, "", nil
	}

	var sourceNotes []summarySourceNote
	var content strings.Builder

	maxNotes := len(notes)
	if maxNotes > 6 {
		maxNotes = 6
	}

	for i := 0; i < maxNotes; i++ {
		notePath := notes[i]
		noteContent, readErr := os.ReadFile(notePath)
		if readErr != nil {
			continue
		}

		relPath, _ := filepath.Rel(targetBrain.Path, notePath)
		content.WriteString(fmt.Sprintf("\n--- %s/%s ---\n", targetBrain.Name, relPath))
		content.Write(noteContent)
		content.WriteString("\n")

		sourceNotes = append(sourceNotes, summarySourceNote{
			BrainName: targetBrain.Name,
			BrainPath: targetBrain.Path,
			NotePath:  notePath,
			RelPath:   relPath,
			Title:     strings.TrimSuffix(filepath.Base(notePath), filepath.Ext(notePath)),
		})

		if content.Len() > 30000 {
			break
		}
	}

	return sourceNotes, strings.TrimSpace(content.String()), nil
}

// searchNotesInBrain searches for notes matching a query in a brain
func searchNotesInBrain(brainPath string, query string) ([]string, error) {
	var matches []string
	query = strings.ToLower(query)
	keywords := strings.Fields(query)

	err := filepath.Walk(brainPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories and non-markdown files
		if info.IsDir() {
			// Skip hidden directories and common non-note directories
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "templates" || name == "assets" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// Check filename
		lowerName := strings.ToLower(info.Name())
		for _, kw := range keywords {
			if strings.Contains(lowerName, kw) {
				matches = append(matches, path)
				return nil
			}
		}

		// Check file content (limited scan)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lowerContent := strings.ToLower(string(content))
		matchCount := 0
		for _, kw := range keywords {
			if strings.Contains(lowerContent, kw) {
				matchCount++
			}
		}

		// Require at least half the keywords to match
		if matchCount >= len(keywords)/2+1 || (len(keywords) == 1 && matchCount == 1) {
			matches = append(matches, path)
		}

		return nil
	})

	return matches, err
}

// newAIResearchCommand creates research notes using AI
func newAIResearchCommand() *cobra.Command {
	var (
		topic               string
		output              string
		brain               string
		title               string
		model               string
		prompt              string
		promptExtra         string
		contextFiles        []string
		contextAutoBrain    bool
		promptCreateTitle   string
		promptCreateBody    string
		promptCreateDefault bool
		timeout             int
		link                bool
		noLink              bool
		noStream            bool
		jsonOut             bool
		prepareOnly         bool
	)

	cmd := &cobra.Command{
		Use:          "research [topic]",
		SilenceUsage: true,
		Short:        "Create a research note on a topic using AI",
		Long: `Create a new note with AI-researched content about a topic.

This command:
1. Sends your topic to the AI provider
2. Gets comprehensive information about the topic
3. Creates a new note with the research results

Examples:
  flip ai research "Kubernetes networking"
  flip ai research "best practices for Go error handling"
  flip ai research "introduction to graph databases" -o graph-db-intro.md`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				topic = args[0]
			}
			JSONOutput = jsonOut
			return runAIResearch(topic, output, brain, title, model, prompt, promptExtra, contextFiles, contextAutoBrain, promptCreateTitle, promptCreateBody, promptCreateDefault, timeout, link, noLink, !noStream, prepareOnly)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: auto-generated)")
	cmd.Flags().StringVar(&brain, "brain", "", "Target brain to save the output (skip selection prompt)")
	cmd.Flags().StringVar(&title, "title", "", "Custom title for the generated note")
	cmd.Flags().StringVar(&model, "model", "", "Override model for this request")
	cmd.Flags().StringVar(&topic, "request", "", "Specific request for this run (alias of topic argument)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt note to apply (path or name)")
	cmd.Flags().StringVar(&prompt, "template", "", "Template note to apply (alias of --prompt)")
	cmd.Flags().StringVar(&promptExtra, "prompt-extra", "", "Additional prompt instructions for this request")
	cmd.Flags().StringVar(&promptExtra, "run-notes", "", "Additional run notes for this request (alias of --prompt-extra)")
	cmd.Flags().StringSliceVar(&contextFiles, "context-file", nil, "Additional context file(s) to include (repeat flag)")
	cmd.Flags().BoolVar(&contextAutoBrain, "context-auto-brain", false, "Automatically search relevant notes in target brain and include as context")
	cmd.Flags().StringVar(&promptCreateTitle, "prompt-create-title", "", "Create a prompt note with this title")
	cmd.Flags().StringVar(&promptCreateTitle, "template-create-title", "", "Create a template note with this title (alias of --prompt-create-title)")
	cmd.Flags().StringVar(&promptCreateBody, "prompt-create-body", "", "Prompt body used when creating a prompt note")
	cmd.Flags().StringVar(&promptCreateBody, "template-create-body", "", "Template body used when creating a template note (alias of --prompt-create-body)")
	cmd.Flags().BoolVar(&promptCreateDefault, "prompt-create-default", false, "Mark created prompt note as default")
	cmd.Flags().BoolVar(&promptCreateDefault, "template-create-default", false, "Mark created template note as default (alias of --prompt-create-default)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Override AI timeout in seconds")
	cmd.Flags().BoolVar(&link, "link", false, "Always add link to today's journal without prompting")
	cmd.Flags().BoolVar(&noLink, "no-link", false, "Do not add a link to today's journal")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON (for VS Code integration)")
	cmd.Flags().BoolVar(&prepareOnly, "prepare-only", false, "Build prompts and create placeholder note, but skip AI call (output JSON with prompts)")

	return cmd
}

func runAIResearch(topic string, outputPath string, brainName string, title string, modelOverride string, promptOverride string, promptExtra string, contextFiles []string, contextAutoBrain bool, promptCreateTitle string, promptCreateBody string, promptCreateDefault bool, timeoutSec int, link bool, noLink bool, stream bool, prepareOnly bool) error {
	if !prepareOnly {
		if err := EnsureAISetup(JSONOutput); err != nil {
			return err
		}
	}
	// Get active workspace first
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if len(activeWs.Brains) == 0 {
		fmt.Println("⚠️  No brains found in active workspace. Please add a brain first.")
		return nil
	}

	// STEP 1: Select target brain for output
	var targetBrain *Brain
	if brainName != "" {
		for i := range activeWs.Brains {
			if activeWs.Brains[i].Name == brainName {
				targetBrain = &activeWs.Brains[i]
				break
			}
		}
		if targetBrain == nil {
			return fmt.Errorf("brain not found: %s", brainName)
		}
	} else {
		targetBrain, err = confirmOrSelectBrain(activeWs)
		if err != nil {
			return err
		}
	}

	// Detect brain type
	detector := brain.NewDetector()
	detection, _ := detector.DetectBrainType(targetBrain.Path)

	promptContent, promptNote, err := resolvePromptNoteForBrain(targetBrain, detection.Type, promptOverride, promptCreateTitle, promptCreateBody, promptCreateDefault, !JSONOutput)
	if err != nil {
		return err
	}

	// Get topic interactively if not provided
	if topic == "" {
		prompt := promptui.Prompt{
			Label: "What do you want to research? (request)",
		}
		topic, err = prompt.Run()
		if err != nil {
			return err
		}
		if topic == "" {
			return fmt.Errorf("topic is required")
		}
	}

	if !prepareOnly {
		PrintOrJSON("\n🔬 Researching: %s\n\n", topic)
	}

	// Create AI client (allow per-request timeout override) — skip for prepare-only
	cfg := ai.LoadConfig()
	if timeoutSec > 0 {
		cfg.Timeout = timeoutSec
	} else if cfg.Timeout < 120 {
		// Research tasks need more time than the default 60s
		cfg.Timeout = 120
	}

	var client *ai.Client
	var effectiveModel string
	ctx := context.Background()

	if prepareOnly {
		// In prepare-only mode, use model override directly (no AI client needed)
		effectiveModel = modelOverride
	} else {
		var err error
		client, err = ai.NewClientWithConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to create AI client: %w", err)
		}

		effectiveModel, err = resolveAIModelForRequest(ctx, client, cfg, modelOverride)
		if err != nil {
			return err
		}
	}	// Build the prompt
	systemPrompt := `You are a knowledgeable research assistant. When asked about a topic:
- Provide comprehensive, accurate information
- Structure the content with clear headings and sections
- Include key concepts, definitions, and explanations
- Add practical examples where relevant
- Note any important considerations or best practices
- Format output in clean markdown
- Be thorough but concise`

	systemPrompt = applyPromptStack(systemPrompt, promptContent, promptExtra)

	var contextParts []string
	var sourceNotes []summarySourceNote

	if contextAutoBrain {
		autoNotes, autoContent, err := collectAutoContextFromBrain(targetBrain, topic)
		if err != nil {
			return err
		}
		if strings.TrimSpace(autoContent) != "" {
			contextParts = append(contextParts, "### Brain context\n"+autoContent)
			sourceNotes = append(sourceNotes, autoNotes...)
		}
	}

	contextFileNotes, contextFileContent, err := loadContextFiles(contextFiles, targetBrain)
	if err != nil {
		return err
	}
	if strings.TrimSpace(contextFileContent) != "" {
		contextParts = append(contextParts, "### Context files\n"+contextFileContent)
		sourceNotes = append(sourceNotes, contextFileNotes...)
	}

	userPrompt := fmt.Sprintf("Please provide a comprehensive overview of: %s\n\nInclude key concepts, practical examples, and best practices.", topic)
	if len(contextParts) > 0 {
		userPrompt += "\n\nUse the following internal context if relevant:\n\n" + strings.Join(contextParts, "\n\n")
	}

	// Resolve title, filename and output path BEFORE the AI call
	noteDate := time.Now().Format("2006-01-02")
	noteDir := filepath.Join(targetBrain.Path, "notes")

	interactiveTitle := !JSONOutput && strings.TrimSpace(title) == ""
	noteTitle, filename, err := resolveAITitleAndFilename(topic, title, noteDir, interactiveTitle)
	if err != nil {
		return err
	}

	if outputPath == "" {
		outputPath = filepath.Join(targetBrain.Path, "notes", filename)
	} else if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(targetBrain.Path, "notes", outputPath)
	}

	usedModel := effectiveModel
	if usedModel == "" {
		usedModel = cfg.Model
	}

	promptMeta := ""
	if promptNote != nil {
		relPrompt := relativePathFromBrain(promptNote.Path, targetBrain.Path)
		promptMeta = fmt.Sprintf("prompt_note: \"%s\"\n", filepath.ToSlash(relPrompt))
	}
	if strings.TrimSpace(promptExtra) != "" {
		promptMeta += "prompt_extra: true\n"
	}

	// Build frontmatter (used for both placeholder and final file)
	buildFrontmatter := func(provider string, model string) string {
		return fmt.Sprintf(`---
title: "%s"
date: %s
type: research
topic: "%s"
ai_generated: true
ai_action: research
ai_created_at: "%s"
ai_provider: "%s"
ai_model: "%s"
%s---

`, noteTitle, noteDate, topic, time.Now().Format(time.RFC3339), provider, model, promptMeta)
	}

	// --prepare-only: return prompts + metadata as JSON so the extension can use Copilot
	if prepareOnly {
		promptSection := buildPromptSection(outputPath, targetBrain, promptNote, promptExtra, true)
		sourcesSection := buildSummarySourcesSection(outputPath, targetBrain, sourceNotes, true)
		result := map[string]interface{}{
			"system_prompt":   systemPrompt,
			"user_prompt":     userPrompt,
			"output_path":     outputPath,
			"frontmatter":     buildFrontmatter("__PROVIDER__", "__MODEL__"),
			"prompt_section":  promptSection,
			"sources_section": sourcesSection,
			"brain_name":      targetBrain.Name,
			"brain_path":      targetBrain.Path,
			"title":           noteTitle,
			"topic":           topic,
		}
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal prepare-only result: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Write placeholder file immediately so the user sees progress
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	placeholderContent := buildFrontmatter(cfg.Provider, usedModel) + fmt.Sprintf("> ⏳ **AI is working…** (%s / %s)\n>\n> Researching: *%s*\n>\n> This note will be updated automatically when the AI finishes.\n", cfg.Provider, usedModel, topic)
	if err := os.WriteFile(outputPath, []byte(placeholderContent), 0644); err != nil {
		return fmt.Errorf("failed to write placeholder file: %w", err)
	}

	PrintOrJSON("\n📝 Note created: %s\n", outputPath)

	// Try to open the file in the editor (best-effort)
	if !JSONOutput {
		if _, err := exec.LookPath("code"); err == nil {
			_ = exec.Command("code", outputPath).Start()
		}
	}

	req := &ai.CompletionRequest{
		System: systemPrompt,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: userPrompt},
		},
		Model:       effectiveModel,
		Temperature: 0.5,
		MaxTokens:   4096,
	}

	PrintlnOrJSON("🤖 Generating research note...")
	PrintlnOrJSON()

	var result strings.Builder
	var aiErr error

	if stream {
		streamResp, err := client.CompleteStream(ctx, req)
		if err != nil {
			aiErr = fmt.Errorf("AI error: %w", err)
		} else {
			defer streamResp.Close()

			for {
				chunk, err := streamResp.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					aiErr = fmt.Errorf("stream error: %w", err)
					break
				}
				if !JSONOutput {
					fmt.Print(chunk)
				}
				result.WriteString(chunk)
			}
			if !JSONOutput {
				fmt.Println()
			}
		}
	} else {
		resp, err := client.Complete(ctx, req)
		if err != nil {
			aiErr = fmt.Errorf("AI error: %w", err)
		} else {
			if !JSONOutput {
				fmt.Println(resp.Content)
			}
			result.WriteString(resp.Content)
			if resp.Model != "" {
				usedModel = resp.Model
			}
		}
	}

	// Update the file with final content (or error message)
	frontmatter := buildFrontmatter(cfg.Provider, usedModel)
	promptSection := buildPromptSection(outputPath, targetBrain, promptNote, promptExtra, true)
	sourcesSection := buildSummarySourcesSection(outputPath, targetBrain, sourceNotes, true)

	var finalContent string
	if aiErr != nil {
		// Write error info into the note so the user sees what happened
		errMsg := fmt.Sprintf("> ❌ **AI request failed**\n>\n> %s\n>\n> You can retry with: `flip ai research \"%s\" --timeout %d`\n\n",
			aiErr.Error(), topic, cfg.Timeout*2)
		if result.Len() > 0 {
			// Partial content was received before the error
			errMsg += "> ⚠️ Partial content received before error:\n\n"
		}
		finalContent = promptSection + sourcesSection + errMsg + result.String()
	} else {
		finalContent = promptSection + sourcesSection + result.String()
	}

	if err := os.WriteFile(outputPath, []byte(frontmatter+finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if JSONOutput {
		OutputJSONSuccess("ai research", NoteResult{
			Action:    "created",
			Path:      outputPath,
			Title:     noteTitle,
			BrainName: targetBrain.Name,
			BrainPath: targetBrain.Path,
			BrainType: string(detection.Type),
		})
	} else if aiErr != nil {
		fmt.Printf("\n\n⚠️  Research note saved with error: %s\n", outputPath)
		if strings.Contains(aiErr.Error(), "deadline exceeded") || strings.Contains(aiErr.Error(), "Timeout") {
			fmt.Printf("💡 Tip: Try again with a longer timeout: flip ai research \"%s\" --timeout %d\n", topic, cfg.Timeout*2)
		}
	} else {
		fmt.Printf("\n\n✅ Research note saved to: %s\n", outputPath)
	}

	// Add link to journal (always attempt, even on AI error – the note file exists)
	if !noLink {
		relPath := relativePathFromBrain(outputPath, targetBrain.Path)
		linkTitle := formatAILinkTitle(noteTitle, "Research", cfg.Provider)
		if err := AddLinkToJournal(JournalLinkOptions{
			ItemType:    "note",
			ItemName:    linkTitle,
			ItemPath:    relPath,
			Brain:       targetBrain,
			Interactive: !link && !JSONOutput,
			BrainType:   detection.Type,
		}); err != nil {
			if !JSONOutput {
				fmt.Printf("⚠️  Could not add journal link: %v\n", err)
			}
		}
	}

	// Return the AI error after everything else is done
	if aiErr != nil {
		return fmt.Errorf("failed to create research note: %w", aiErr)
	}

	return nil
}

// newAIImproveCommand improves existing notes with AI
func newAIImproveCommand() *cobra.Command {
	var (
		instruction string
		noStream    bool
		inPlace     bool
		model       string
		prompt      string
		promptExtra string
		timeout     int
		prepareOnly bool
	)

	cmd := &cobra.Command{
		Use:          "improve [file]",
		SilenceUsage: true,
		Short:        "Improve a note with AI assistance",
		Long: `Use AI to improve, rewrite, or enhance an existing note.

This command:
1. Reads the content of the specified note
2. Sends it to the AI with your improvement instructions
3. Shows the improved version (optionally saves in place)

Examples:
  flip ai improve notes/my-note.md
  flip ai improve notes/draft.md --instruction "make it more concise"
  flip ai improve notes/rough.md --instruction "fix grammar and improve flow" --in-place`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAIImprove(args[0], instruction, model, prompt, promptExtra, timeout, !noStream, inPlace, prepareOnly)
		},
	}

	cmd.Flags().StringVarP(&instruction, "instruction", "i", "", "Specific improvement instructions")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")
	cmd.Flags().BoolVar(&inPlace, "in-place", false, "Update the file in place")
	cmd.Flags().StringVar(&model, "model", "", "Override model for this request")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt note to apply (path or name)")
	cmd.Flags().StringVar(&prompt, "template", "", "Template note to apply (alias of --prompt)")
	cmd.Flags().StringVar(&promptExtra, "prompt-extra", "", "Additional prompt instructions for this request")
	cmd.Flags().StringVar(&promptExtra, "run-notes", "", "Additional run notes for this request (alias of --prompt-extra)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Override AI timeout in seconds")
	cmd.Flags().BoolVar(&prepareOnly, "prepare-only", false, "Return prompts as JSON without calling AI (for Copilot integration)")

	return cmd
}

func runAIImprove(filePath string, instruction string, modelOverride string, promptOverride string, promptExtra string, timeoutSec int, stream bool, inPlace bool, prepareOnly bool) error {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	promptContent := ""
	brainInfo, brainType := resolveBrainForFile(filePath)
	promptContent, _, err = resolvePromptNoteForBrain(brainInfo, brainType, promptOverride, "", "", false, promptOverride == "")
	if err != nil {
		return err
	}

	// Get instruction interactively if not provided
	if instruction == "" {
		prompt := promptui.Prompt{
			Label:   "How should the note be improved?",
			Default: "improve clarity and structure",
		}
		instruction, err = prompt.Run()
		if err != nil {
			return err
		}
	}

	if !prepareOnly {
		fmt.Printf("\n📝 Improving: %s\n", filePath)
		fmt.Printf("💡 Instruction: %s\n\n", instruction)
	}

	// Create AI client (allow per-request timeout override) — skip for prepare-only
	cfg := ai.LoadConfig()
	if timeoutSec > 0 {
		cfg.Timeout = timeoutSec
	}

	var client *ai.Client
	var effectiveModel string
	ctx := context.Background()

	if prepareOnly {
		effectiveModel = modelOverride
	} else {
		var cErr error
		client, cErr = ai.NewClientWithConfig(cfg)
		if cErr != nil {
			return fmt.Errorf("failed to create AI client: %w", cErr)
		}
		var mErr error
		effectiveModel, mErr = resolveAIModelForRequest(ctx, client, cfg, modelOverride)
		if mErr != nil {
			return mErr
		}
	}

	// Build the prompt
	systemPrompt := `You are a skilled editor helping to improve markdown notes.
When given a note and improvement instructions:
- Apply the requested improvements
- Preserve the overall structure and meaning
- Keep any existing frontmatter (YAML between --- markers)
- Maintain markdown formatting
- Output only the improved note, nothing else`

	systemPrompt = applyPromptStack(systemPrompt, promptContent, promptExtra)

	userPrompt := fmt.Sprintf("Please improve the following note according to this instruction: %s\n\n---\n\n%s", instruction, string(content))

	// --prepare-only: return prompts + metadata as JSON so the extension can use Copilot
	if prepareOnly {
		result := map[string]interface{}{
			"system_prompt": systemPrompt,
			"user_prompt":   userPrompt,
			"file_path":     filePath,
			"in_place":      inPlace,
			"instruction":   instruction,
		}
		if brainInfo != nil {
			result["brain_name"] = brainInfo.Name
			result["brain_path"] = brainInfo.Path
		}
		jsonBytes, jErr := json.MarshalIndent(result, "", "  ")
		if jErr != nil {
			return fmt.Errorf("failed to marshal prepare-only result: %w", jErr)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	req := &ai.CompletionRequest{
		System: systemPrompt,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: userPrompt},
		},
		Model:       effectiveModel,
		Temperature: 0.4,
		MaxTokens:   8192,
	}

	fmt.Println("🤖 Improving note...")
	fmt.Println()

	var result strings.Builder

	if stream {
		streamResp, err := client.CompleteStream(ctx, req)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}
		defer streamResp.Close()

		for {
			chunk, err := streamResp.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("stream error: %w", err)
			}
			fmt.Print(chunk)
			result.WriteString(chunk)
		}
		fmt.Println()
	} else {
		resp, err := client.Complete(ctx, req)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}
		fmt.Println(resp.Content)
		result.WriteString(resp.Content)
	}

	if inPlace {
		if err := os.WriteFile(filePath, []byte(result.String()), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("\n✅ File updated: %s\n", filePath)
	} else {
		fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("💡 Use --in-place to save changes to the file")
	}

	return nil
}

// aiGetDefaultBrain returns the default brain from a workspace
func aiGetDefaultBrain(ws *Workspace) *Brain {
	if ws == nil {
		return nil
	}
	for i := range ws.Brains {
		if ws.Brains[i].Name == ws.DefaultBrain {
			return &ws.Brains[i]
		}
	}
	if len(ws.Brains) > 0 {
		return &ws.Brains[0]
	}
	return nil
}

func resolvePromptNoteForBrain(brainInfo *Brain, brainType brain.BrainType, override string, promptCreateTitle string, promptCreateBody string, promptCreateDefault bool, interactive bool) (string, *PromptNoteInfo, error) {
	if brainInfo == nil {
		override = strings.TrimSpace(override)
		if override == "" {
			return "", nil, nil
		}
		lower := strings.ToLower(override)
		if lower == "none" || lower == "off" || lower == "no" {
			return "", nil, nil
		}
		if filepath.IsAbs(override) {
			content, err := loadPromptNoteContent(override)
			if err != nil {
				return "", nil, err
			}
			return content, &PromptNoteInfo{Path: override}, nil
		}
		return "", nil, fmt.Errorf("prompt note requires a brain context or absolute path")
	}

	if strings.TrimSpace(promptCreateTitle) != "" {
		promptNote, err := createPromptNote(brainInfo.Path, brainType, promptCreateTitle, promptCreateBody, promptCreateDefault)
		if err != nil {
			return "", nil, err
		}
		content, err := loadPromptNoteContent(promptNote.Path)
		if err != nil {
			return "", nil, err
		}
		return content, &promptNote, nil
	}

	prompts, defaultPrompt, err := findPromptNotes(brainInfo.Path, brainType)
	if err != nil {
		return "", nil, err
	}

	resolved, handled, err := resolvePromptOverride(override, brainInfo.Path, brainType, prompts)
	if err != nil {
		return "", nil, err
	}
	if handled {
		if resolved.Path == "" {
			return "", nil, nil
		}
		content, err := loadPromptNoteContent(resolved.Path)
		if err != nil {
			return "", nil, err
		}
		return content, &resolved, nil
	}

	if defaultPrompt != nil {
		content, err := loadPromptNoteContent(defaultPrompt.Path)
		if err != nil {
			return "", nil, err
		}
		return content, defaultPrompt, nil
	}

	if !interactive || len(prompts) == 0 {
		if interactive && len(prompts) == 0 {
			createSelector := promptui.Select{
				Label: "No prompt notes found. Create one?",
				Items: []string{"Yes", "No"},
			}
			_, createChoice, err := createSelector.Run()
			if err != nil {
				return "", nil, err
			}
			if createChoice == "Yes" {
				titlePrompt := promptui.Prompt{
					Label: "Prompt title",
					Validate: func(input string) error {
						if strings.TrimSpace(input) == "" {
							return fmt.Errorf("title cannot be empty")
						}
						return nil
					},
				}
				title, err := titlePrompt.Run()
				if err != nil {
					return "", nil, err
				}
				bodyPrompt := promptui.Prompt{
					Label: "Prompt instructions",
				}
				body, err := bodyPrompt.Run()
				if err != nil {
					return "", nil, err
				}
				defaultPromptSelect := promptui.Select{
					Label: "Set as default prompt?",
					Items: []string{"No", "Yes"},
				}
				_, defaultChoice, err := defaultPromptSelect.Run()
				if err != nil {
					return "", nil, err
				}
				setDefault := defaultChoice == "Yes"
				created, err := createPromptNote(brainInfo.Path, brainType, title, body, setDefault)
				if err != nil {
					return "", nil, err
				}
				content, err := loadPromptNoteContent(created.Path)
				if err != nil {
					return "", nil, err
				}
				return content, &created, nil
			}
		}
		return "", nil, nil
	}

	items := []string{"No prompt"}
	for _, prompt := range prompts {
		items = append(items, prompt.Title)
	}

	selector := promptui.Select{
		Label: "Select prompt note (optional)",
		Items: items,
		Size:  calculateMenuSize(len(items)),
	}
	idx, _, err := selector.Run()
	if err != nil {
		return "", nil, err
	}
	if idx == 0 {
		return "", nil, nil
	}

	selected := prompts[idx-1]
	content, err := loadPromptNoteContent(selected.Path)
	if err != nil {
		return "", nil, err
	}

	return content, &selected, nil
}

func applyPromptStack(systemPrompt string, promptContent string, promptExtra string) string {
	parts := []string{}
	if strings.TrimSpace(promptContent) != "" {
		parts = append(parts, strings.TrimSpace(promptContent))
	}
	if strings.TrimSpace(promptExtra) != "" {
		parts = append(parts, strings.TrimSpace(promptExtra))
	}
	parts = append(parts, systemPrompt)
	return strings.Join(parts, "\n\n")
}

func resolveAITitleAndFilename(topic string, title string, targetDir string, interactive bool) (string, string, error) {
	resolvedTitle := strings.TrimSpace(title)
	if resolvedTitle == "" {
		resolvedTitle = suggestTitleFromTopic(topic)
	}

	if interactive {
		prompt := promptui.Prompt{
			Label:   "Title",
			Default: resolvedTitle,
			Validate: func(input string) error {
				if strings.TrimSpace(input) == "" {
					return fmt.Errorf("title cannot be empty")
				}
				filename := aiFilenameFromTitle(strings.TrimSpace(input))
				if aiFileExists(filepath.Join(targetDir, filename)) {
					return fmt.Errorf("a note with this title already exists")
				}
				return nil
			},
		}
		value, err := prompt.Run()
		if err != nil {
			return "", "", err
		}
		resolvedTitle = strings.TrimSpace(value)
		filename := aiFilenameFromTitle(resolvedTitle)
		return resolvedTitle, filename, nil
	}

	slug := aiSlugFromTitle(resolvedTitle)
	filename := uniqueAIFilename(slug, targetDir)
	return resolvedTitle, filename, nil
}

func suggestTitleFromTopic(topic string) string {
	cleaned := strings.TrimSpace(topic)
	if cleaned == "" {
		return "New Note"
	}

	for _, sep := range []string{".", "?", "!"} {
		if idx := strings.Index(cleaned, sep); idx > 15 {
			cleaned = cleaned[:idx]
			break
		}
	}

	words := strings.Fields(cleaned)
	if len(words) > 8 {
		cleaned = strings.Join(words[:8], " ")
	}

	cleaned = strings.Trim(cleaned, "\"'")
	if cleaned == "" {
		return "New Note"
	}
	return cleaned
}

func aiFilenameFromTitle(title string) string {
	slug := aiSlugFromTitle(title)
	if slug == "" {
		slug = "note"
	}
	return fmt.Sprintf("%s.md", slug)
}

func aiSlugFromTitle(title string) string {
	input := strings.ToLower(strings.TrimSpace(title))
	if input == "" {
		return ""
	}

	var b strings.Builder
	lastDash := false
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if len(slug) > 40 {
		slug = slug[:40]
		slug = strings.Trim(slug, "-")
	}
	return slug
}

func uniqueAIFilename(slug string, targetDir string) string {
	base := fmt.Sprintf("%s.md", slug)
	if !aiFileExists(filepath.Join(targetDir, base)) {
		return base
	}
	for i := 2; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d.md", slug, i)
		if !aiFileExists(filepath.Join(targetDir, candidate)) {
			return candidate
		}
	}
	return fmt.Sprintf("%s.md", slug)
}

func aiFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func formatAILinkTitle(title string, action string, provider string) string {
	if provider == "" {
		return title
	}
	return fmt.Sprintf("%s (%s: %s)", title, action, provider)
}

func resolveAIModelForRequest(ctx context.Context, client *ai.Client, cfg *ai.Config, modelOverride string) (string, error) {
	modelOverride = strings.TrimSpace(modelOverride)
	if modelOverride != "" {
		return modelOverride, nil
	}

	// For all providers: list available models and let user pick interactively
	models, err := client.ListModels(ctx)
	if err != nil {
		// If listing fails, fall back to default
		return "", nil
	}

	if len(models) == 0 {
		if cfg.Provider == "ollama" {
			return "", fmt.Errorf("AI error: no Ollama models installed. Run `flip ai setup` (or `ollama pull %s`) and try again", cfg.Model)
		}
		return "", nil
	}

	// For Ollama: check if configured model exists, fallback if not
	if cfg.Provider == "ollama" {
		configured := strings.TrimSpace(cfg.Model)
		if configured == "" {
			return firstModelID(models), nil
		}

		found := false
		for _, m := range models {
			if strings.EqualFold(m.ID, configured) || strings.EqualFold(m.Name, configured) {
				found = true
				break
			}
		}
		if !found {
			fallback := firstModelID(models)
			if fallback != "" && !JSONOutput {
				fmt.Printf("⚠️  Ollama model '%s' not found locally, using '%s'\n", configured, fallback)
			}
			if fallback != "" {
				return fallback, nil
			}
		}
	}

	// Interactive model selection for all providers (when not in JSON mode)
	if !JSONOutput && len(models) > 1 {
		items := make([]string, len(models))
		defaultIdx := 0
		configuredModel := strings.TrimSpace(cfg.Model)
		for i, m := range models {
			desc := m.Name
			if m.Description != "" {
				desc = fmt.Sprintf("%s – %s", m.Name, m.Description)
			}
			if m.ContextSize > 0 {
				desc = fmt.Sprintf("%s (%dk ctx)", desc, m.ContextSize/1000)
			}
			items[i] = desc
			if strings.EqualFold(m.ID, configuredModel) || strings.EqualFold(m.Name, configuredModel) {
				defaultIdx = i
			}
		}

		sel := promptui.Select{
			Label:     "Select model",
			Items:     items,
			CursorPos: defaultIdx,
			Size:      10,
		}
		idx, _, err := sel.Run()
		if err != nil {
			return "", err
		}
		return models[idx].ID, nil
	}

	return "", nil
}

func firstModelID(models []ai.ModelInfo) string {
	for _, m := range models {
		if strings.TrimSpace(m.ID) != "" {
			return m.ID
		}
		if strings.TrimSpace(m.Name) != "" {
			return m.Name
		}
	}
	return ""
}

func resolveBrainForFile(filePath string) (*Brain, brain.BrainType) {
	brainPath := detectBrainPath(filePath)
	if brainPath == "" {
		return nil, brain.BrainTypeFlip
	}

	detector := brain.NewDetector()
	detection, _ := detector.DetectBrainType(brainPath)

	ws, err := getActiveWorkspace()
	if err == nil {
		absBrainPath, _ := filepath.Abs(brainPath)
		for i := range ws.Brains {
			brainAbs, _ := filepath.Abs(ws.Brains[i].Path)
			if filepath.Clean(brainAbs) == filepath.Clean(absBrainPath) {
				return &ws.Brains[i], detection.Type
			}
		}
	}

	return &Brain{
		Name: filepath.Base(brainPath),
		Path: brainPath,
		Type: string(detection.Type),
	}, detection.Type
}

// EnsureAISetup checks if an AI provider and model are configured,
// and prompts the user to set it up if they aren't.
func EnsureAISetup(jsonOutput bool) error {
	cfg := ai.LoadConfig()
	if cfg.Provider == "" || cfg.Model == "" {
		if jsonOutput {
			// Trigger VS Code extension to show setup UI
			OutputJSONError("ai check", fmt.Errorf("AI not set up. Run 'flip ai setup'"))
			return fmt.Errorf("setup required")
		}

		fmt.Println("⚠️  AI provider and model are not fully configured yet.")

		prompt := promptui.Prompt{
			Label:     "Would you like to run the setup now?",
			IsConfirm: true,
			Default:   "Y",
		}

		_, err := prompt.Run()
		if err != nil {
			return fmt.Errorf("AI configuration is required to use this command")
		}

		return runAISetup("", "", "", "", false, false)
	}
	return nil
}
