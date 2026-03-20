package commands

// ai_status.go - AI status and models commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/httrp/flip/internal/ai"
	"github.com/spf13/cobra"
)

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
