package commands

import (
	"errors"
	"testing"
)

func TestFlipErrorCreation(t *testing.T) {
	err := NewError(ErrCodeBrain, "Test error message")

	if err.Code != ErrCodeBrain {
		t.Errorf("Expected code %s, got %s", ErrCodeBrain, err.Code)
	}
	if err.Message != "Test error message" {
		t.Errorf("Expected message 'Test error message', got %s", err.Message)
	}
}

func TestFlipErrorWithCause(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewError(ErrCodeFile, "File operation failed").WithCause(cause)

	if err.Cause != cause {
		t.Error("Cause not set correctly")
	}

	// Test Unwrap
	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Error("Unwrap should return cause")
	}
}

func TestFlipErrorWithHint(t *testing.T) {
	err := NewError(ErrCodeConfig, "Config missing").WithHint("Try running init")

	if err.Hint != "Try running init" {
		t.Errorf("Hint not set correctly: %s", err.Hint)
	}
}

func TestFlipErrorString(t *testing.T) {
	// Without cause
	err := NewError(ErrCodeValidation, "Invalid input")
	expected := "[VALIDATION] Invalid input"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}

	// With cause
	cause := errors.New("bad value")
	errWithCause := NewError(ErrCodeValidation, "Invalid input").WithCause(cause)
	if errWithCause.Error() != "[VALIDATION] Invalid input: bad value" {
		t.Errorf("Unexpected error string: %s", errWithCause.Error())
	}
}

func TestErrConfigNotFound(t *testing.T) {
	err := ErrConfigNotFound("/path/to/config")

	if err.Code != ErrCodeConfig {
		t.Error("Should be CONFIG error")
	}
	if err.Hint == "" {
		t.Error("Should have a hint")
	}
}

func TestErrBrainNotFound(t *testing.T) {
	err := ErrBrainNotFound("mybrain")

	if err.Code != ErrCodeBrain {
		t.Error("Should be BRAIN error")
	}
	if err.Hint == "" {
		t.Error("Should have a hint")
	}
}

func TestErrWorkspaceNotFound(t *testing.T) {
	err := ErrWorkspaceNotFound("myworkspace")

	if err.Code != ErrCodeWorkspace {
		t.Error("Should be WORKSPACE error")
	}
}

func TestErrNoActiveWorkspace(t *testing.T) {
	err := ErrNoActiveWorkspace()

	if err.Code != ErrCodeWorkspace {
		t.Error("Should be WORKSPACE error")
	}
	if err.Hint == "" {
		t.Error("Should have a hint")
	}
}

func TestErrNoBrainInWorkspace(t *testing.T) {
	err := ErrNoBrainInWorkspace()

	if err.Code != ErrCodeBrain {
		t.Error("Should be BRAIN error")
	}
}

func TestErrFileNotFound(t *testing.T) {
	err := ErrFileNotFound("/missing/file.md")

	if err.Code != ErrCodeFile {
		t.Error("Should be FILE error")
	}
}

func TestErrInvalidInput(t *testing.T) {
	err := ErrInvalidInput("name", "cannot be empty")

	if err.Code != ErrCodeValidation {
		t.Error("Should be VALIDATION error")
	}
}

func TestErrGitOperation(t *testing.T) {
	cause := errors.New("network error")
	err := ErrGitOperation("push", cause)

	if err.Code != ErrCodeGit {
		t.Error("Should be GIT error")
	}
	if err.Cause != cause {
		t.Error("Should have cause")
	}
}

func TestIsNotFoundError(t *testing.T) {
	// FlipError with NOT_FOUND code
	flipErr := NewError(ErrCodeNotFound, "Resource not found")
	if !IsNotFoundError(flipErr) {
		t.Error("Should detect NOT_FOUND FlipError")
	}

	// Regular error with "not found" in message
	regularErr := errors.New("file not found")
	if !IsNotFoundError(regularErr) {
		t.Error("Should detect 'not found' in message")
	}

	// Error without "not found"
	otherErr := errors.New("permission denied")
	if IsNotFoundError(otherErr) {
		t.Error("Should not detect 'not found' in unrelated error")
	}
}

func TestIsConfigError(t *testing.T) {
	configErr := NewError(ErrCodeConfig, "Config issue")
	if !IsConfigError(configErr) {
		t.Error("Should detect CONFIG error")
	}

	otherErr := NewError(ErrCodeBrain, "Brain issue")
	if IsConfigError(otherErr) {
		t.Error("Should not detect CONFIG for BRAIN error")
	}

	regularErr := errors.New("regular error")
	if IsConfigError(regularErr) {
		t.Error("Should not detect CONFIG for regular error")
	}
}

func TestErrorCodeConstants(t *testing.T) {
	codes := map[ErrorCode]string{
		ErrCodeConfig:     "CONFIG",
		ErrCodeBrain:      "BRAIN",
		ErrCodeWorkspace:  "WORKSPACE",
		ErrCodeFile:       "FILE",
		ErrCodeGit:        "GIT",
		ErrCodeParse:      "PARSE",
		ErrCodeValidation: "VALIDATION",
		ErrCodeNotFound:   "NOT_FOUND",
		ErrCodePermission: "PERMISSION",
		ErrCodeInternal:   "INTERNAL",
	}

	for code, expected := range codes {
		if string(code) != expected {
			t.Errorf("ErrorCode %v should be %s", code, expected)
		}
	}
}

func TestErrorChaining(t *testing.T) {
	// Create nested errors
	rootCause := errors.New("network timeout")
	gitErr := ErrGitOperation("fetch", rootCause)
	
	// Test errors.Is / errors.As
	var flipErr *FlipError
	if !errors.As(gitErr, &flipErr) {
		t.Error("Should be able to extract FlipError")
	}

	// Test unwrap chain
	if !errors.Is(gitErr, rootCause) {
		t.Error("Should be able to detect root cause via errors.Is")
	}
}
