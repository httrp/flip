package ai

// ollama.go - Ollama Provider Implementation
//
// Ollama is the default provider for flip AI features.
// It requires no API key and works locally out of the box.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	host    string
	model   string
	client  *http.Client
	timeout time.Duration
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(cfg *Config) *OllamaProvider {
	host := cfg.OllamaHost
	if host == "" {
		host = "http://localhost:11434"
	}
	model := cfg.OllamaModel
	if model == "" {
		model = "llama3.2"
	}
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &OllamaProvider{
		host:    host,
		model:   model,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", p.host+"/api/tags", nil)
	if err != nil {
		return false
	}

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

// ollamaRequest is the request body for Ollama API
type ollamaRequest struct {
	Model    string           `json:"model"`
	Messages []ollamaMessage  `json:"messages"`
	Stream   bool             `json:"stream"`
	Options  *ollamaOptions   `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// ollamaResponse is the response from non-streaming Ollama API
type ollamaResponse struct {
	Model     string        `json:"model"`
	Message   ollamaMessage `json:"message"`
	Done      bool          `json:"done"`
	TotalTime int64         `json:"total_duration"`
	EvalCount int           `json:"eval_count"`
}

// ollamaStreamResponse is a single chunk from streaming Ollama API
type ollamaStreamResponse struct {
	Model   string        `json:"model"`
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

func (p *OllamaProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build messages
	var messages []ollamaMessage
	if req.System != "" {
		messages = append(messages, ollamaMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		messages = append(messages, ollamaMessage{Role: string(m.Role), Content: m.Content})
	}

	// Build request body
	body := ollamaRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
		Options: &ollamaOptions{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, NewProviderError("ollama", ErrCodeUnknown, "failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.host+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, NewProviderError("ollama", ErrCodeUnknown, "failed to create request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, NewProviderError("ollama", ErrCodeConnection, "failed to connect to Ollama", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, NewProviderError("ollama", ErrCodeUnknown, fmt.Sprintf("API error %d: %s", resp.StatusCode, string(bodyBytes)), nil)
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, NewProviderError("ollama", ErrCodeUnknown, "failed to decode response", err)
	}

	return &CompletionResponse{
		Content: ollamaResp.Message.Content,
		Model:   ollamaResp.Model,
		Usage: Usage{
			CompletionTokens: ollamaResp.EvalCount,
		},
		FinishReason: "stop",
	}, nil
}

func (p *OllamaProvider) CompleteStream(ctx context.Context, req *CompletionRequest) (Stream, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build messages
	var messages []ollamaMessage
	if req.System != "" {
		messages = append(messages, ollamaMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		messages = append(messages, ollamaMessage{Role: string(m.Role), Content: m.Content})
	}

	// Build request body
	body := ollamaRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
		Options: &ollamaOptions{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, NewProviderError("ollama", ErrCodeUnknown, "failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.host+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, NewProviderError("ollama", ErrCodeUnknown, "failed to create request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, NewProviderError("ollama", ErrCodeConnection, "failed to connect to Ollama", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, NewProviderError("ollama", ErrCodeUnknown, fmt.Sprintf("API error %d: %s", resp.StatusCode, string(bodyBytes)), nil)
	}

	return &ollamaStream{
		reader: bufio.NewReader(resp.Body),
		body:   resp.Body,
	}, nil
}

type ollamaStream struct {
	reader *bufio.Reader
	body   io.ReadCloser
	done   bool
}

func (s *ollamaStream) Next() (string, error) {
	if s.done {
		return "", io.EOF
	}

	line, err := s.reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			s.done = true
		}
		return "", err
	}

	var chunk ollamaStreamResponse
	if err := json.Unmarshal(line, &chunk); err != nil {
		return "", err
	}

	if chunk.Done {
		s.done = true
		return "", io.EOF
	}

	return chunk.Message.Content, nil
}

func (s *ollamaStream) Close() error {
	s.done = true
	return s.body.Close()
}

func (p *OllamaProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.host+"/api/tags", nil)
	if err != nil {
		return nil, NewProviderError("ollama", ErrCodeUnknown, "failed to create request", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NewProviderError("ollama", ErrCodeConnection, "failed to connect to Ollama", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError("ollama", ErrCodeUnknown, fmt.Sprintf("API error %d", resp.StatusCode), nil)
	}

	var result struct {
		Models []struct {
			Name       string `json:"name"`
			Model      string `json:"model"`
			ModifiedAt string `json:"modified_at"`
			Size       int64  `json:"size"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, NewProviderError("ollama", ErrCodeUnknown, "failed to decode response", err)
	}

	var models []ModelInfo
	for _, m := range result.Models {
		models = append(models, ModelInfo{
			ID:   m.Name,
			Name: m.Name,
		})
	}
	return models, nil
}
