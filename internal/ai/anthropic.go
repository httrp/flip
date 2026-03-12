package ai

// anthropic.go - Anthropic Claude Provider Implementation
//
// Supports Claude models via the Anthropic API.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AnthropicProvider implements the Provider interface for Anthropic
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
	timeout time.Duration
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(cfg *Config) *AnthropicProvider {
	baseURL := "https://api.anthropic.com/v1"

	model := cfg.AnthropicModel
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &AnthropicProvider{
		apiKey:  cfg.AnthropicKey,
		baseURL: baseURL,
		model:   model,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) IsAvailable(ctx context.Context) bool {
	return p.apiKey != ""
}

// anthropicRequest is the request body for Anthropic Messages API
type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the response from Anthropic Messages API
type anthropicResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Content      []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// anthropicStreamEvent is an event from streaming Anthropic API
type anthropicStreamEvent struct {
	Type         string `json:"type"`
	Index        int    `json:"index,omitempty"`
	ContentBlock *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content_block,omitempty"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"delta,omitempty"`
	Message *anthropicResponse `json:"message,omitempty"`
}

func (p *AnthropicProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}

	// Build messages (Anthropic doesn't use system role in messages)
	var messages []anthropicMessage
	for _, m := range req.Messages {
		// Skip system messages, they go in the system field
		if m.Role == RoleSystem {
			continue
		}
		messages = append(messages, anthropicMessage{Role: string(m.Role), Content: m.Content})
	}

	// Build request body
	body := anthropicRequest{
		Model:     model,
		Messages:  messages,
		System:    req.System,
		MaxTokens: maxTokens,
		Stream:    false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, NewProviderError("anthropic", ErrCodeUnknown, "failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, NewProviderError("anthropic", ErrCodeUnknown, "failed to create request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, NewProviderError("anthropic", ErrCodeConnection, "failed to connect to Anthropic", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, NewProviderError("anthropic", ErrCodeAuth, "invalid API key", nil)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, NewProviderError("anthropic", ErrCodeRateLimit, "rate limit exceeded", nil)
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, NewProviderError("anthropic", ErrCodeUnknown, fmt.Sprintf("API error %d: %s", resp.StatusCode, string(bodyBytes)), nil)
	}

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, NewProviderError("anthropic", ErrCodeUnknown, "failed to decode response", err)
	}

	// Extract text content
	var content strings.Builder
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			content.WriteString(c.Text)
		}
	}

	return &CompletionResponse{
		Content: content.String(),
		Model:   anthropicResp.Model,
		Usage: Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
		FinishReason: anthropicResp.StopReason,
	}, nil
}

func (p *AnthropicProvider) CompleteStream(ctx context.Context, req *CompletionRequest) (Stream, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}

	// Build messages
	var messages []anthropicMessage
	for _, m := range req.Messages {
		if m.Role == RoleSystem {
			continue
		}
		messages = append(messages, anthropicMessage{Role: string(m.Role), Content: m.Content})
	}

	// Build request body
	body := anthropicRequest{
		Model:     model,
		Messages:  messages,
		System:    req.System,
		MaxTokens: maxTokens,
		Stream:    true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, NewProviderError("anthropic", ErrCodeUnknown, "failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, NewProviderError("anthropic", ErrCodeUnknown, "failed to create request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, NewProviderError("anthropic", ErrCodeConnection, "failed to connect to Anthropic", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, NewProviderError("anthropic", ErrCodeUnknown, fmt.Sprintf("API error %d: %s", resp.StatusCode, string(bodyBytes)), nil)
	}

	return &anthropicStream{
		reader: bufio.NewReader(resp.Body),
		body:   resp.Body,
	}, nil
}

type anthropicStream struct {
	reader *bufio.Reader
	body   io.ReadCloser
	done   bool
}

func (s *anthropicStream) Next() (string, error) {
	if s.done {
		return "", io.EOF
	}

	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				s.done = true
			}
			return "", err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Type == "text_delta" {
				return event.Delta.Text, nil
			}
		case "message_stop":
			s.done = true
			return "", io.EOF
		}
	}
}

func (s *anthropicStream) Close() error {
	s.done = true
	return s.body.Close()
}

func (p *AnthropicProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Anthropic doesn't have a models endpoint, return known models
	models := []ModelInfo{
		{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Description: "Most intelligent model"},
		{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Description: "Fast and cost-effective"},
		{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Description: "Powerful model for complex tasks"},
		{ID: "claude-3-sonnet-20240229", Name: "Claude 3 Sonnet", Description: "Balanced performance"},
		{ID: "claude-3-haiku-20240307", Name: "Claude 3 Haiku", Description: "Fast and efficient"},
	}
	return models, nil
}
