package commands

// logger.go - Structured logging for flip
//
// Provides:
//   - Log levels (Debug, Info, Warn, Error)
//   - Structured fields support
//   - Configurable output (stdout, file, silent)

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelSilent // No logging
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelSilent:
		return "SILENT"
	default:
		return "UNKNOWN"
	}
}

// LogFormat defines the output format
type LogFormat int

const (
	LogFormatText LogFormat = iota // Human-readable text
	LogFormatJSON                  // JSON for parsing
)

// Logger provides structured logging
type Logger struct {
	mu       sync.Mutex
	level    LogLevel
	format   LogFormat
	output   io.Writer
	fields   map[string]interface{} // Default fields for all logs
	disabled bool
}

// LogEntry represents a single log entry
type LogEntry struct {
	Time    time.Time              `json:"time"`
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// Global logger instance
var (
	defaultLogger *Logger
	loggerOnce    sync.Once
)

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	loggerOnce.Do(func() {
		defaultLogger = NewLogger()
	})
	return defaultLogger
}

// NewLogger creates a new logger with default settings
func NewLogger() *Logger {
	return &Logger{
		level:  LogLevelInfo,
		format: LogFormatText,
		output: os.Stderr,
		fields: make(map[string]interface{}),
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetFormat sets the output format
func (l *Logger) SetFormat(format LogFormat) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.format = format
}

// SetOutput sets the output writer
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// Disable turns off all logging
func (l *Logger) Disable() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.disabled = true
}

// Enable turns on logging
func (l *Logger) Enable() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.disabled = false
}

// WithField returns a new logger with an additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		level:  l.level,
		format: l.format,
		output: l.output,
		fields: make(map[string]interface{}),
	}

	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value

	return newLogger
}

// WithFields returns a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		level:  l.level,
		format: l.format,
		output: l.output,
		fields: make(map[string]interface{}),
	}

	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// log writes a log entry
func (l *Logger) log(level LogLevel, msg string, fields map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.disabled || level < l.level {
		return
	}

	entry := LogEntry{
		Time:    time.Now(),
		Level:   level.String(),
		Message: msg,
		Fields:  make(map[string]interface{}),
	}

	// Merge default fields with provided fields
	for k, v := range l.fields {
		entry.Fields[k] = v
	}
	for k, v := range fields {
		entry.Fields[k] = v
	}

	var output string
	if l.format == LogFormatJSON {
		data, _ := json.Marshal(entry)
		output = string(data)
	} else {
		output = l.formatText(entry)
	}

	fmt.Fprintln(l.output, output)
}

// formatText formats a log entry as human-readable text
func (l *Logger) formatText(entry LogEntry) string {
	var sb strings.Builder

	// Timestamp
	sb.WriteString(entry.Time.Format("15:04:05"))
	sb.WriteString(" ")

	// Level with color hints for terminal
	switch entry.Level {
	case "DEBUG":
		sb.WriteString("[DBG]")
	case "INFO":
		sb.WriteString("[INF]")
	case "WARN":
		sb.WriteString("[WRN]")
	case "ERROR":
		sb.WriteString("[ERR]")
	}
	sb.WriteString(" ")

	// Message
	sb.WriteString(entry.Message)

	// Fields
	if len(entry.Fields) > 0 {
		sb.WriteString(" ")
		first := true
		for k, v := range entry.Fields {
			if !first {
				sb.WriteString(" ")
			}
			sb.WriteString(fmt.Sprintf("%s=%v", k, v))
			first = false
		}
	}

	return sb.String()
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelDebug, msg, f)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(LogLevelDebug, fmt.Sprintf(format, args...), nil)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelInfo, msg, f)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LogLevelInfo, fmt.Sprintf(format, args...), nil)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelWarn, msg, f)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LogLevelWarn, fmt.Sprintf(format, args...), nil)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelError, msg, f)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LogLevelError, fmt.Sprintf(format, args...), nil)
}

// Package-level convenience functions

// Debug logs a debug message to the default logger
func Debug(msg string, fields ...map[string]interface{}) {
	GetLogger().Debug(msg, fields...)
}

// Debugf logs a formatted debug message to the default logger
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info logs an info message to the default logger  
func Info(msg string, fields ...map[string]interface{}) {
	GetLogger().Info(msg, fields...)
}

// Infof logs a formatted info message to the default logger
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn logs a warning message to the default logger
func Warn(msg string, fields ...map[string]interface{}) {
	GetLogger().Warn(msg, fields...)
}

// Warnf logs a formatted warning message to the default logger
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// LogError logs an error message to the default logger
func LogError(msg string, fields ...map[string]interface{}) {
	GetLogger().Error(msg, fields...)
}

// LogErrorf logs a formatted error message to the default logger
func LogErrorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// SetLogLevel sets the log level on the default logger
func SetLogLevel(level LogLevel) {
	GetLogger().SetLevel(level)
}

// SetLogFormat sets the log format on the default logger
func SetLogFormat(format LogFormat) {
	GetLogger().SetFormat(format)
}

// ParseLogLevel parses a log level string
func ParseLogLevel(s string) LogLevel {
	switch strings.ToLower(s) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarn
	case "error":
		return LogLevelError
	case "silent", "none", "off":
		return LogLevelSilent
	default:
		return LogLevelInfo
	}
}

// InitLogging initializes logging based on environment variables
func InitLogging() {
	// Check FLIP_LOG_LEVEL env var
	if level := os.Getenv("FLIP_LOG_LEVEL"); level != "" {
		SetLogLevel(ParseLogLevel(level))
	}

	// Check FLIP_LOG_FORMAT env var
	if format := os.Getenv("FLIP_LOG_FORMAT"); format == "json" {
		SetLogFormat(LogFormatJSON)
	}
}
