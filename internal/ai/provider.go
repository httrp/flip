package ai

// provider.go - AI Provider Interface and Types
//
// Defines the common interface for all AI providers (Ollama, OpenAI, Anthropic, etc.)
// Supports both streaming and non-streaming completions.

import (
	"context"
	"fmt"
	"io"
	"time"
)

// Provider is the common interface for all AI providers
type Provider interface {
	// Name returns the provider name (e.g., "ollama", "openai", "anthropic")
	Name() string

	// IsAvailable checks if the provider is configured and reachable
	IsAvailable(ctx context.Context) bool

	// Complete sends a prompt and returns the full response
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// CompleteStream sends a prompt and streams the response
	CompleteStream(ctx context.Context, req *CompletionRequest) (Stream, error)

	// ListModels returns available models for this provider
	ListModels(ctx context.Context) ([]ModelInfo, error)
}

// Stream represents a streaming response from an AI provider
type Stream interface {
	// Next returns the next chunk of the response
	// Returns io.EOF when complete
	Next() (string, error)

	// Close releases resources associated with the stream
	Close() error
}

// CompletionRequest contains parameters for a completion request
type CompletionRequest struct {
	// Model specifies which model to use (provider-specific)
	// If empty, the provider's default model is used
	Model string

	// System is the system prompt that sets the AI's behavior
	System string

	// Messages contains the conversation history
	Messages []Message

	// Temperature controls randomness (0.0-1.0, default varies by provider)
	Temperature float64

	// MaxTokens limits the response length (0 = provider default)
	MaxTokens int

	// Options contains provider-specific options
	Options map[string]interface{}
}

// Message represents a single message in the conversation
type Message struct {
	Role    Role
	Content string
}

// Role represents the role of a message sender
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// CompletionResponse contains the result of a completion request
type CompletionResponse struct {
	// Content is the generated text
	Content string

	// Model is the actual model used
	Model string

	// Usage contains token usage statistics
	Usage Usage

	// FinishReason indicates why generation stopped
	FinishReason string
}

// Usage contains token usage statistics
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// ModelInfo describes an available model
type ModelInfo struct {
	ID          string
	Name        string
	Description string
	ContextSize int
}

// ProviderError represents an error from an AI provider
type ProviderError struct {
	Provider string
	Code     string
	Message  string
	Err      error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return e.Provider + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Provider + ": " + e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new provider error
func NewProviderError(provider, code, message string, err error) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Code:     code,
		Message:  message,
		Err:      err,
	}
}

// Common error codes
const (
	ErrCodeConfiguration = "CONFIG_ERROR"
	ErrCodeConnection    = "CONNECTION_ERROR"
	ErrCodeRateLimit     = "RATE_LIMIT"
	ErrCodeAuth          = "AUTH_ERROR"
	ErrCodeModelNotFound = "MODEL_NOT_FOUND"
	ErrCodeTimeout       = "TIMEOUT"
	ErrCodeUnknown       = "UNKNOWN"
)

// simpleStream wraps a channel into a Stream interface
type simpleStream struct {
	ch     chan string
	err    error
	closed bool
}

func (s *simpleStream) Next() (string, error) {
	if s.closed {
		return "", io.EOF
	}
	chunk, ok := <-s.ch
	if !ok {
		s.closed = true
		if s.err != nil {
			return "", s.err
		}
		return "", io.EOF
	}
	return chunk, nil
}

func (s *simpleStream) Close() error {
	s.closed = true
	return nil
}

// NewSimpleStream creates a stream from a channel
func NewSimpleStream(ch chan string) Stream {
	return &simpleStream{ch: ch}
}

// KeyStatus represents the result of an API key validation check.
type KeyStatus struct {
	// Provider name
	Provider string

	// Valid indicates whether the key was accepted by the API
	Valid bool

	// Message provides a human-readable status description
	Message string

	// ExpiresAt contains the key expiry time, if the provider reports it.
	// Zero value means the provider does not expose expiry information.
	ExpiresAt time.Time

	// RateLimited indicates whether the key is currently rate-limited
	RateLimited bool
}

// HasExpiry returns true if the provider reported an expiry date.
func (ks *KeyStatus) HasExpiry() bool {
	return !ks.ExpiresAt.IsZero()
}

// ExpiryString returns a human-readable expiry string, or "" if unknown.
func (ks *KeyStatus) ExpiryString() string {
	if ks.ExpiresAt.IsZero() {
		return ""
	}
	remaining := time.Until(ks.ExpiresAt)
	if remaining <= 0 {
		return "expired"
	}
	days := int(remaining.Hours() / 24)
	if days > 30 {
		return ks.ExpiresAt.Format("2006-01-02")
	}
	if days == 0 {
		return "expires today"
	}
	if days == 1 {
		return "expires tomorrow"
	}
	return fmt.Sprintf("expires in %d days", days)
}
