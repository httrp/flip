package commands

// errors.go - Centralized error handling
//
// Provides:
//   - Custom error types with context
//   - User-friendly error messages
//   - Consistent error formatting

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCode represents a category of error
type ErrorCode string

const (
	ErrCodeConfig     ErrorCode = "CONFIG"     // Configuration errors
	ErrCodeBrain      ErrorCode = "BRAIN"      // Brain-related errors
	ErrCodeWorkspace  ErrorCode = "WORKSPACE"  // Workspace errors
	ErrCodeFile       ErrorCode = "FILE"       // File system errors
	ErrCodeGit        ErrorCode = "GIT"        // Git operation errors
	ErrCodeParse      ErrorCode = "PARSE"      // Parsing errors
	ErrCodeValidation ErrorCode = "VALIDATION" // Input validation errors
	ErrCodeNotFound   ErrorCode = "NOT_FOUND"  // Resource not found
	ErrCodePermission ErrorCode = "PERMISSION" // Permission denied
	ErrCodeInternal   ErrorCode = "INTERNAL"   // Internal errors
)

// FlipError is a structured error with context
type FlipError struct {
	Code    ErrorCode
	Message string
	Hint    string // User-friendly suggestion
	Cause   error  // Underlying error
}

func (e *FlipError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *FlipError) Unwrap() error {
	return e.Cause
}

// NewError creates a new FlipError
func NewError(code ErrorCode, message string) *FlipError {
	return &FlipError{
		Code:    code,
		Message: message,
	}
}

// WithCause adds an underlying error
func (e *FlipError) WithCause(cause error) *FlipError {
	e.Cause = cause
	return e
}

// WithHint adds a user-friendly hint
func (e *FlipError) WithHint(hint string) *FlipError {
	e.Hint = hint
	return e
}

// Common error constructors

// ErrConfigNotFound creates a config not found error
func ErrConfigNotFound(path string) *FlipError {
	return NewError(ErrCodeConfig, fmt.Sprintf("Configuration not found: %s", path)).
		WithHint("Run 'flip workspace new' to create a workspace first")
}

// ErrBrainNotFound creates a brain not found error
func ErrBrainNotFound(name string) *FlipError {
	return NewError(ErrCodeBrain, fmt.Sprintf("Brain '%s' not found", name)).
		WithHint("Run 'flip brain list' to see available brains")
}

// ErrWorkspaceNotFound creates a workspace not found error
func ErrWorkspaceNotFound(name string) *FlipError {
	return NewError(ErrCodeWorkspace, fmt.Sprintf("Workspace '%s' not found", name)).
		WithHint("Run 'flip workspace list' to see available workspaces")
}

// ErrNoActiveWorkspace creates an error for no active workspace
func ErrNoActiveWorkspace() *FlipError {
	return NewError(ErrCodeWorkspace, "No active workspace").
		WithHint("Run 'flip workspace switch <name>' to set an active workspace")
}

// ErrNoBrainInWorkspace creates an error when workspace has no brains
func ErrNoBrainInWorkspace() *FlipError {
	return NewError(ErrCodeBrain, "No brains in active workspace").
		WithHint("Run 'flip brain add <path>' to add a brain")
}

// ErrFileNotFound creates a file not found error
func ErrFileNotFound(path string) *FlipError {
	return NewError(ErrCodeFile, fmt.Sprintf("File not found: %s", path))
}

// ErrInvalidInput creates a validation error
func ErrInvalidInput(field, reason string) *FlipError {
	return NewError(ErrCodeValidation, fmt.Sprintf("Invalid %s: %s", field, reason))
}

// ErrGitOperation creates a git operation error
func ErrGitOperation(operation string, cause error) *FlipError {
	return NewError(ErrCodeGit, fmt.Sprintf("Git %s failed", operation)).
		WithCause(cause)
}

// PrintError outputs an error in a consistent format
func PrintError(err error) {
	var flipErr *FlipError
	if errors.As(err, &flipErr) {
		fmt.Printf("\n%s %s\n", IconError, flipErr.Message)
		if flipErr.Hint != "" {
			fmt.Printf("   💡 %s\n", flipErr.Hint)
		}
		if flipErr.Cause != nil && isVerbose() {
			fmt.Printf("   Cause: %v\n", flipErr.Cause)
		}
	} else {
		fmt.Printf("\n%s Error: %v\n", IconError, err)
	}
}

// PrintErrorf outputs a formatted error message
func PrintErrorf(format string, args ...interface{}) {
	fmt.Printf("\n%s %s\n", IconError, fmt.Sprintf(format, args...))
}

// PrintWarning outputs a warning message
func PrintWarning(message string) {
	fmt.Printf("%s %s\n", IconWarning, message)
}

// PrintWarningf outputs a formatted warning message
func PrintWarningf(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", IconWarning, fmt.Sprintf(format, args...))
}

// PrintSuccess outputs a success message
func PrintSuccess(message string) {
	fmt.Printf("%s %s\n", IconCheck, message)
}

// PrintSuccessf outputs a formatted success message
func PrintSuccessf(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", IconCheck, fmt.Sprintf(format, args...))
}

// PrintInfo outputs an info message
func PrintInfo(message string) {
	fmt.Printf("%s %s\n", IconInfo, message)
}

// PrintInfof outputs a formatted info message
func PrintInfof(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", IconInfo, fmt.Sprintf(format, args...))
}

// isVerbose checks if verbose mode is enabled
func isVerbose() bool {
	// TODO: Check global verbose flag
	return false
}

// IsNotFoundError checks if an error is a "not found" type error
func IsNotFoundError(err error) bool {
	var flipErr *FlipError
	if errors.As(err, &flipErr) {
		return flipErr.Code == ErrCodeNotFound
	}
	return strings.Contains(err.Error(), "not found")
}

// IsConfigError checks if an error is a configuration error
func IsConfigError(err error) bool {
	var flipErr *FlipError
	if errors.As(err, &flipErr) {
		return flipErr.Code == ErrCodeConfig
	}
	return false
}
