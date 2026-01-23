package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevelSilent, "SILENT"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.level.String())
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", LogLevelDebug},
		{"DEBUG", LogLevelDebug},
		{"info", LogLevelInfo},
		{"INFO", LogLevelInfo},
		{"warn", LogLevelWarn},
		{"warning", LogLevelWarn},
		{"error", LogLevelError},
		{"silent", LogLevelSilent},
		{"none", LogLevelSilent},
		{"off", LogLevelSilent},
		{"unknown", LogLevelInfo}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLogLevel(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoggerSetLevel(t *testing.T) {
	logger := NewLogger()
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Set to warn level
	logger.SetLevel(LogLevelWarn)

	// Info should not appear
	logger.Info("info message")
	if buf.Len() > 0 {
		t.Error("Info message should not appear at warn level")
	}

	// Warn should appear
	logger.Warn("warn message")
	if buf.Len() == 0 {
		t.Error("Warn message should appear at warn level")
	}
}

func TestLoggerDisable(t *testing.T) {
	logger := NewLogger()
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	logger.Disable()
	logger.Error("should not appear")

	if buf.Len() > 0 {
		t.Error("Disabled logger should not output")
	}

	logger.Enable()
	logger.Error("should appear")

	if buf.Len() == 0 {
		t.Error("Enabled logger should output")
	}
}

func TestLoggerWithField(t *testing.T) {
	logger := NewLogger()
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	childLogger := logger.WithField("component", "test")
	childLogger.SetOutput(&buf)
	childLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "component=test") {
		t.Errorf("Expected field in output: %s", output)
	}
}

func TestLoggerWithFields(t *testing.T) {
	logger := NewLogger()
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	childLogger := logger.WithFields(map[string]interface{}{
		"brain": "main",
		"user":  "test",
	})
	childLogger.SetOutput(&buf)
	childLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "brain=main") {
		t.Errorf("Expected brain field in output: %s", output)
	}
}

func TestLoggerJSONFormat(t *testing.T) {
	logger := NewLogger()
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.SetFormat(LogFormatJSON)

	logger.Info("test message", map[string]interface{}{"key": "value"})

	output := buf.String()
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Errorf("Expected JSON format: %s", output)
	}
	if !strings.Contains(output, `"message":"test message"`) {
		t.Errorf("Expected message in JSON: %s", output)
	}
}

func TestLoggerTextFormat(t *testing.T) {
	logger := NewLogger()
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.SetFormat(LogFormatText)

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "[INF]") {
		t.Errorf("Expected [INF] in text format: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected message in output: %s", output)
	}
}

func TestLoggerMethods(t *testing.T) {
	logger := NewLogger()
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.SetLevel(LogLevelDebug)

	tests := []struct {
		method   func()
		expected string
	}{
		{func() { logger.Debug("debug msg") }, "[DBG]"},
		{func() { logger.Info("info msg") }, "[INF]"},
		{func() { logger.Warn("warn msg") }, "[WRN]"},
		{func() { logger.Error("error msg") }, "[ERR]"},
	}

	for _, tt := range tests {
		buf.Reset()
		tt.method()
		if !strings.Contains(buf.String(), tt.expected) {
			t.Errorf("Expected %s in output: %s", tt.expected, buf.String())
		}
	}
}

func TestLoggerFormatMethods(t *testing.T) {
	logger := NewLogger()
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.SetLevel(LogLevelDebug)

	logger.Debugf("debug %s", "formatted")
	if !strings.Contains(buf.String(), "debug formatted") {
		t.Errorf("Debugf output incorrect: %s", buf.String())
	}

	buf.Reset()
	logger.Infof("info %d", 42)
	if !strings.Contains(buf.String(), "info 42") {
		t.Errorf("Infof output incorrect: %s", buf.String())
	}

	buf.Reset()
	logger.Warnf("warn %v", true)
	if !strings.Contains(buf.String(), "warn true") {
		t.Errorf("Warnf output incorrect: %s", buf.String())
	}

	buf.Reset()
	logger.Errorf("error %s", "test")
	if !strings.Contains(buf.String(), "error test") {
		t.Errorf("Errorf output incorrect: %s", buf.String())
	}
}

func TestGetLogger(t *testing.T) {
	logger1 := GetLogger()
	logger2 := GetLogger()

	if logger1 != logger2 {
		t.Error("GetLogger should return singleton")
	}
}

func TestLogEntry(t *testing.T) {
	entry := LogEntry{
		Level:   "INFO",
		Message: "Test",
		Fields:  map[string]interface{}{"key": "value"},
	}

	if entry.Level != "INFO" {
		t.Error("Level mismatch")
	}
	if entry.Message != "Test" {
		t.Error("Message mismatch")
	}
	if entry.Fields["key"] != "value" {
		t.Error("Fields mismatch")
	}
}
