package ai

// openai.go - OpenAI Provider Implementation
//
// Supports OpenAI API and OpenAI-compatible APIs (like Groq, Together, etc.)
// Also used as base for Azure OpenAI.

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

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	orgID   string
	client  *http.Client
	timeout time.Duration
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(cfg *Config) *OpenAIProvider {
	baseURL := cfg.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	model := cfg.OpenAIModel
	if model == "" {
		model = "gpt-4o-mini"
	}

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &OpenAIProvider{
		apiKey:  cfg.OpenAIKey,
		baseURL: baseURL,
		model:   model,
		orgID:   cfg.OpenAIOrgID,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) IsAvailable(ctx context.Context) bool {
	if p.apiKey == "" {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// openaiRequest is the request body for OpenAI Chat API
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiResponse is the response from OpenAI Chat API
type openaiResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openaiMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// openaiStreamChunk is a single chunk from streaming OpenAI API
type openaiStreamChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build messages
	var messages []openaiMessage
	if req.System != "" {
		messages = append(messages, openaiMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		messages = append(messages, openaiMessage{Role: string(m.Role), Content: m.Content})
	}

	// Build request body
	body := openaiRequest{
		Model:       model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, NewProviderError("openai", ErrCodeUnknown, "failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, NewProviderError("openai", ErrCodeUnknown, "failed to create request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	if p.orgID != "" {
		httpReq.Header.Set("OpenAI-Organization", p.orgID)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, NewProviderError("openai", ErrCodeConnection, "failed to connect to OpenAI", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, NewProviderError("openai", ErrCodeAuth, "invalid API key", nil)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, NewProviderError("openai", ErrCodeRateLimit, "rate limit exceeded", nil)
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, NewProviderError("openai", ErrCodeUnknown, fmt.Sprintf("API error %d: %s", resp.StatusCode, string(bodyBytes)), nil)
	}

	var openaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, NewProviderError("openai", ErrCodeUnknown, "failed to decode response", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, NewProviderError("openai", ErrCodeUnknown, "no response from model", nil)
	}

	return &CompletionResponse{
		Content: openaiResp.Choices[0].Message.Content,
		Model:   openaiResp.Model,
		Usage: Usage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
		FinishReason: openaiResp.Choices[0].FinishReason,
	}, nil
}

func (p *OpenAIProvider) CompleteStream(ctx context.Context, req *CompletionRequest) (Stream, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build messages
	var messages []openaiMessage
	if req.System != "" {
		messages = append(messages, openaiMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		messages = append(messages, openaiMessage{Role: string(m.Role), Content: m.Content})
	}

	// Build request body
	body := openaiRequest{
		Model:       model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, NewProviderError("openai", ErrCodeUnknown, "failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, NewProviderError("openai", ErrCodeUnknown, "failed to create request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	if p.orgID != "" {
		httpReq.Header.Set("OpenAI-Organization", p.orgID)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, NewProviderError("openai", ErrCodeConnection, "failed to connect to OpenAI", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, NewProviderError("openai", ErrCodeUnknown, fmt.Sprintf("API error %d: %s", resp.StatusCode, string(bodyBytes)), nil)
	}

	return &openaiStream{
		reader: bufio.NewReader(resp.Body),
		body:   resp.Body,
	}, nil
}

type openaiStream struct {
	reader *bufio.Reader
	body   io.ReadCloser
	done   bool
}

func (s *openaiStream) Next() (string, error) {
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
		if data == "[DONE]" {
			s.done = true
			return "", io.EOF
		}

		var chunk openaiStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // Skip malformed chunks
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			return chunk.Choices[0].Delta.Content, nil
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].FinishReason != nil {
			s.done = true
			return "", io.EOF
		}
	}
}

func (s *openaiStream) Close() error {
	s.done = true
	return s.body.Close()
}

func (p *OpenAIProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, NewProviderError("openai", ErrCodeUnknown, "failed to create request", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NewProviderError("openai", ErrCodeConnection, "failed to connect to OpenAI", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError("openai", ErrCodeUnknown, fmt.Sprintf("API error %d", resp.StatusCode), nil)
	}

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, NewProviderError("openai", ErrCodeUnknown, "failed to decode response", err)
	}

	var models []ModelInfo
	for _, m := range result.Data {
		// Filter to only chat models
		if strings.HasPrefix(m.ID, "gpt-") || strings.Contains(m.ID, "turbo") {
			models = append(models, ModelInfo{
				ID:   m.ID,
				Name: m.ID,
			})
		}
	}
	return models, nil
}
