package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		want   error
		name   string
		config Config
	}{
		{
			name: "default config",
			config: Config{
				Level:     LevelInfo,
				Format:    FormatText,
				Output:    "stdout",
				AddSource: true,
			},
			want: nil,
		},
		{
			name: "json format to stderr",
			config: Config{
				Level:     LevelDebug,
				Format:    FormatJSON,
				Output:    "stderr",
				AddSource: false,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if (err != nil) != (tt.want != nil) {
				t.Errorf("NewLogger() error = %v, want %v", err, tt.want)
			}

			if logger != nil {
				logger.Close()
			}
		})
	}
}

func TestLogger_FileOutput(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := Config{
		Level:     LevelInfo,
		Format:    FormatJSON,
		Output:    logFile,
		AddSource: false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	// Log a test message
	logger.Info("test message", "key", "value")

	// Verify file was created and contains our message
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file was not created")
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test message") {
		t.Errorf("Log file does not contain expected message")
	}
}

func TestLogger_WithComponent(t *testing.T) {
	var buf bytes.Buffer

	// Create a logger that writes to our buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{})
	baseLogger := slog.New(handler)

	logger := &Logger{
		Logger: baseLogger,
		config: DefaultConfig(),
		writer: &buf,
	}

	componentLogger := logger.WithComponent("test-component")
	componentLogger.Info("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log: %v", err)
	}

	if logEntry["component"] != "test-component" {
		t.Errorf("Expected component 'test-component', got %v", logEntry["component"])
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{})
	baseLogger := slog.New(handler)

	logger := &Logger{
		Logger: baseLogger,
		config: DefaultConfig(),
		writer: &buf,
	}

	fields := map[string]interface{}{
		"user_id": 123,
		"action":  "login",
	}

	fieldLogger := logger.WithFields(fields)
	fieldLogger.Info("user action")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log: %v", err)
	}

	if logEntry["user_id"] != float64(123) { // JSON numbers are float64
		t.Errorf("Expected user_id 123, got %v", logEntry["user_id"])
	}

	if logEntry["action"] != "login" {
		t.Errorf("Expected action 'login', got %v", logEntry["action"])
	}
}

func TestEventLogger(t *testing.T) {
	config := Config{
		Level:     LevelDebug,
		Format:    FormatJSON,
		Output:    "stdout",
		AddSource: false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	eventLogger := NewEventLogger(logger)
	defer eventLogger.Close()

	// Test matrix logging
	eventLogger.LogMatrix(LevelInfo, "matrix connected", "matrix-1", map[string]interface{}{
		"port": "/dev/ttyUSB0",
	})

	// Test stats logging
	eventLogger.LogStats(LevelInfo, "cpu usage collected", "cpu", 45.5, map[string]interface{}{
		"cores": 8,
	})

	// Test config logging
	eventLogger.LogConfig(LevelInfo, "config reloaded", "/etc/config.yaml", nil)

	// Test daemon logging
	eventLogger.LogDaemon(LevelInfo, "daemon started", "start", map[string]interface{}{
		"pid": 1234,
	})

	// Give time for async processing
	time.Sleep(100 * time.Millisecond)
}

func TestMetricsLogger(t *testing.T) {
	config := Config{
		Level:     LevelInfo,
		Format:    FormatJSON,
		Output:    "stdout",
		AddSource: false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	metricsLogger := NewMetricsLogger(logger)

	// Test counter metric
	metricsLogger.LogCounter("requests_total", 100, map[string]string{
		"method": "GET",
		"status": "200",
	})

	// Test gauge metric
	metricsLogger.LogGauge("memory_usage_bytes", 1024*1024*512, map[string]string{
		"type": "heap",
	})

	// Test histogram metric
	metricsLogger.LogHistogram("request_duration_seconds", 0.5, map[string]string{
		"endpoint": "/api/stats",
	})

	// Test timing metric
	metricsLogger.LogTiming("operation_duration", 250*time.Millisecond, map[string]string{
		"operation": "collect_stats",
	})
}

func TestPerformanceTracker(t *testing.T) {
	config := Config{
		Level:     LevelInfo,
		Format:    FormatJSON,
		Output:    "stdout",
		AddSource: false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	metricsLogger := NewMetricsLogger(logger)

	// Test successful operation tracking
	tracker := metricsLogger.StartTracking("test_operation", map[string]string{
		"component": "test",
	})

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	duration := tracker.Finish()
	if duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", duration)
	}

	// Test error operation tracking
	errorTracker := metricsLogger.StartTracking("error_operation", nil)

	time.Sleep(5 * time.Millisecond)

	testError := &testError{msg: "test error"}

	errorDuration := errorTracker.FinishWithError(testError)
	if errorDuration < 5*time.Millisecond {
		t.Errorf("Expected error duration >= 5ms, got %v", errorDuration)
	}
}

func TestGlobalLogger(t *testing.T) {
	// Test default global logger
	originalLogger := globalLogger
	globalLogger = nil // Reset for test

	defer func() { globalLogger = originalLogger }()

	logger := GetGlobalLogger()
	if logger == nil {
		t.Error("Expected non-nil global logger")
	}

	// Test setting custom global logger
	config := Config{
		Level:  LevelDebug,
		Format: FormatText,
		Output: "stderr",
	}

	customLogger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer customLogger.Close()

	SetGlobalLogger(customLogger)

	retrievedLogger := GetGlobalLogger()
	if retrievedLogger != customLogger {
		t.Error("Global logger was not set correctly")
	}

	// Test convenience functions
	Debug("debug message", "key", "value")
	Info("info message", "key", "value")
	Warn("warn message", "key", "value")
	Error("error message", "key", "value")

	componentLogger := WithComponent("test")
	if componentLogger == nil {
		t.Error("Expected non-nil component logger")
	}

	fieldsLogger := WithFields(map[string]interface{}{"test": "value"})
	if fieldsLogger == nil {
		t.Error("Expected non-nil fields logger")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Level != LevelInfo {
		t.Errorf("Expected level %v, got %v", LevelInfo, config.Level)
	}

	if config.Format != FormatText {
		t.Errorf("Expected format %v, got %v", FormatText, config.Format)
	}

	if config.Output != "stdout" {
		t.Errorf("Expected output 'stdout', got %v", config.Output)
	}

	if !config.AddSource {
		t.Error("Expected AddSource to be true")
	}
}

// Test error type for testing error logging.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestEventLogger_LogError(t *testing.T) {
	config := Config{
		Level:     LevelError,
		Format:    FormatJSON,
		Output:    "stdout",
		AddSource: false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	eventLogger := NewEventLogger(logger)
	defer eventLogger.Close()

	testErr := &testError{msg: "test error message"}
	eventLogger.LogError(testErr, "operation failed", map[string]interface{}{
		"operation": "test",
	})

	// Give time for async processing
	time.Sleep(100 * time.Millisecond)
}

func TestLogger_WithContext(t *testing.T) {
	config := DefaultConfig()

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	ctx := context.Background()
	contextLogger := logger.WithContext(ctx)

	if contextLogger == nil {
		t.Error("Expected non-nil context logger")

		return
	}

	// Test that we can log with the context logger
	contextLogger.Info("test message with context")
}
