package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// LogFormat represents the logging format
type LogFormat string

const (
	FormatJSON LogFormat = "json"
	FormatText LogFormat = "text"
)

// Config holds the logger configuration
type Config struct {
	Level     LogLevel  `yaml:"level" json:"level"`
	Format    LogFormat `yaml:"format" json:"format"`
	Output    string    `yaml:"output" json:"output"` // "stdout", "stderr", or file path
	AddSource bool      `yaml:"add_source" json:"add_source"`
}

// DefaultConfig returns a Config populated with sensible defaults: Info level,
// text format, stdout output, and AddSource enabled.
func DefaultConfig() Config {
	return Config{
		Level:     LevelInfo,
		Format:    FormatText,
		Output:    "stdout",
		AddSource: true,
	}
}

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
	config Config
	writer io.Writer
}

// Timestamps in log records are rendered using RFC3339.
func NewLogger(config Config) (*Logger, error) {
	var writer io.Writer
	var err error

	// Determine output writer
	switch config.Output {
	case "stdout", "":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// File output
		if err := os.MkdirAll(filepath.Dir(config.Output), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		writer, err = os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	// Convert level
	var level slog.Level
	switch config.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: config.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}

	switch config.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(writer, opts)
	case FormatText:
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	logger := &Logger{
		Logger: slog.New(handler),
		config: config,
		writer: writer,
	}

	return logger, nil
}

// WithContext returns a logger with the given context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: l.Logger.With(),
		config: l.config,
		writer: l.writer,
	}
}

// WithComponent adds a component field to the logger
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With("component", component),
		config: l.config,
		writer: l.writer,
	}
}

// WithFields adds structured fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{
		Logger: l.Logger.With(args...),
		config: l.config,
		writer: l.writer,
	}
}

// Close closes the logger and any associated resources
func (l *Logger) Close() error {
	if closer, ok := l.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// LogEvent represents a structured log event for metrics
type LogEvent struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// EventLogger provides structured event logging for observability
type EventLogger struct {
	logger *Logger
	events chan LogEvent
	done   chan struct{}
}

// NewEventLogger creates and returns an EventLogger that asynchronously processes structured observability events.
// The returned EventLogger uses the provided Logger as its output, allocates a buffered event channel (capacity 1000),
// and starts a background goroutine to process events. Call Close on the EventLogger to stop the processor and drain
// any pending events before shutdown.
func NewEventLogger(logger *Logger) *EventLogger {
	el := &EventLogger{
		logger: logger,
		events: make(chan LogEvent, 1000), // Buffer for async logging
		done:   make(chan struct{}),
	}
	
	go el.processEvents()
	return el
}

// LogMatrix logs matrix-related events
func (el *EventLogger) LogMatrix(level LogLevel, message string, matrixID string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["matrix_id"] = matrixID
	
	el.logEvent(level, "matrix", message, fields, nil)
}

// LogStats logs statistics collection events
func (el *EventLogger) LogStats(level LogLevel, message string, statsType string, value float64, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["stats_type"] = statsType
	fields["value"] = value
	
	el.logEvent(level, "stats", message, fields, nil)
}

// LogConfig logs configuration-related events
func (el *EventLogger) LogConfig(level LogLevel, message string, configPath string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["config_path"] = configPath
	
	el.logEvent(level, "config", message, fields, nil)
}

// LogDaemon logs daemon lifecycle events
func (el *EventLogger) LogDaemon(level LogLevel, message string, action string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["action"] = action
	
	el.logEvent(level, "daemon", message, fields, nil)
}

// LogError logs error events with stack trace context
func (el *EventLogger) LogError(err error, message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	
	// Add stack trace context
	if pc, file, line, ok := runtime.Caller(1); ok {
		fields["caller_file"] = filepath.Base(file)
		fields["caller_line"] = line
		if fn := runtime.FuncForPC(pc); fn != nil {
			fields["caller_func"] = fn.Name()
		}
	}
	
	el.logEvent(LevelError, "error", message, fields, err)
}

func (el *EventLogger) logEvent(level LogLevel, component, message string, fields map[string]interface{}, err error) {
	event := LogEvent{
		Level:     string(level),
		Message:   message,
		Component: component,
		Timestamp: time.Now(),
		Fields:    fields,
	}
	
	if err != nil {
		event.Error = err.Error()
	}
	
	select {
	case el.events <- event:
	case <-el.done:
		return
	default:
		// Channel full, log directly to avoid blocking
		el.logEventDirect(event)
	}
}

func (el *EventLogger) logEventDirect(event LogEvent) {
	logger := el.logger.WithComponent(event.Component)
	
	args := make([]interface{}, 0, len(event.Fields)*2+4)
	args = append(args, "timestamp", event.Timestamp)
	
	for k, v := range event.Fields {
		args = append(args, k, v)
	}
	
	if event.Error != "" {
		args = append(args, "error", event.Error)
	}
	
	switch LogLevel(event.Level) {
	case LevelDebug:
		logger.Debug(event.Message, args...)
	case LevelInfo:
		logger.Info(event.Message, args...)
	case LevelWarn:
		logger.Warn(event.Message, args...)
	case LevelError:
		logger.Error(event.Message, args...)
	}
}

func (el *EventLogger) processEvents() {
	for {
		select {
		case event := <-el.events:
			el.logEventDirect(event)
		case <-el.done:
			// Drain remaining events
			for {
				select {
				case event := <-el.events:
					el.logEventDirect(event)
				default:
					return
				}
			}
		}
	}
}

// Close stops the event logger
func (el *EventLogger) Close() {
	close(el.done)
}

// MetricsLogger provides structured metrics logging
type MetricsLogger struct {
	logger *Logger
}

// NewMetricsLogger returns a MetricsLogger that uses the provided Logger scoped to the
// "metrics" component for all metric entries.
func NewMetricsLogger(logger *Logger) *MetricsLogger {
	return &MetricsLogger{
		logger: logger.WithComponent("metrics"),
	}
}

// LogCounter logs a counter metric
func (ml *MetricsLogger) LogCounter(name string, value int64, labels map[string]string) {
	fields := map[string]interface{}{
		"metric_type": "counter",
		"metric_name": name,
		"value":       value,
	}
	
	if labels != nil {
		for k, v := range labels {
			fields["label_"+k] = v
		}
	}
	
	ml.logger.Info("counter metric", slog.Any("fields", fields))
}

// LogGauge logs a gauge metric
func (ml *MetricsLogger) LogGauge(name string, value float64, labels map[string]string) {
	fields := map[string]interface{}{
		"metric_type": "gauge",
		"metric_name": name,
		"value":       value,
	}
	
	if labels != nil {
		for k, v := range labels {
			fields["label_"+k] = v
		}
	}
	
	ml.logger.Info("gauge metric", slog.Any("fields", fields))
}

// LogHistogram logs a histogram metric
func (ml *MetricsLogger) LogHistogram(name string, value float64, labels map[string]string) {
	fields := map[string]interface{}{
		"metric_type": "histogram",
		"metric_name": name,
		"value":       value,
	}
	
	if labels != nil {
		for k, v := range labels {
			fields["label_"+k] = v
		}
	}
	
	ml.logger.Info("histogram metric", slog.Any("fields", fields))
}

// LogTiming logs timing information
func (ml *MetricsLogger) LogTiming(name string, duration time.Duration, labels map[string]string) {
	fields := map[string]interface{}{
		"metric_type":     "timing",
		"metric_name":     name,
		"duration_ms":     duration.Milliseconds(),
		"duration_string": duration.String(),
	}
	
	if labels != nil {
		for k, v := range labels {
			fields["label_"+k] = v
		}
	}
	
	ml.logger.Info("timing metric", slog.Any("fields", fields))
}

// Performance tracking helpers
type PerformanceTracker struct {
	logger    *MetricsLogger
	startTime time.Time
	operation string
	labels    map[string]string
}

// StartTracking begins performance tracking for an operation
func (ml *MetricsLogger) StartTracking(operation string, labels map[string]string) *PerformanceTracker {
	return &PerformanceTracker{
		logger:    ml,
		startTime: time.Now(),
		operation: operation,
		labels:    labels,
	}
}

// Finish completes the performance tracking and logs the duration
func (pt *PerformanceTracker) Finish() time.Duration {
	duration := time.Since(pt.startTime)
	pt.logger.LogTiming(pt.operation, duration, pt.labels)
	return duration
}

// FinishWithError completes tracking and logs an error if one occurred
func (pt *PerformanceTracker) FinishWithError(err error) time.Duration {
	duration := time.Since(pt.startTime)
	
	labels := pt.labels
	if labels == nil {
		labels = make(map[string]string)
	}
	
	if err != nil {
		labels["error"] = "true"
		labels["error_message"] = err.Error()
	} else {
		labels["error"] = "false"
	}
	
	pt.logger.LogTiming(pt.operation, duration, labels)
	return duration
}

// Global logger instance
var globalLogger *Logger

// SetGlobalLogger sets the package-level global logger used by the convenience logging helpers.
// Passing nil clears the global logger; GetGlobalLogger will create and return a default logger on next use.
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the package-level global *Logger.
// If no global logger has been set, it lazily creates and caches a default logger using DefaultConfig.
// Note: logger construction errors are ignored; if creation fails, this may return nil.
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		// Fallback to default logger
		config := DefaultConfig()
		logger, _ := NewLogger(config)
		globalLogger = logger
	}
	return globalLogger
}

// Debug logs a message at the debug level using the package global logger.
// Any additional arguments are forwarded to the global Logger's Debug method (typically key/value pairs for structured fields).
func Debug(msg string, args ...interface{}) {
	GetGlobalLogger().Debug(msg, args...)
}

// Info logs an informational message using the package-level global logger.
// It delegates to the global logger's Info method with the provided message and optional key/value pairs.
func Info(msg string, args ...interface{}) {
	GetGlobalLogger().Info(msg, args...)
}

// Warn logs a warning-level message through the package global logger.
// 
// msg is the log message; args may be provided as optional key/value pairs to include with the entry.
func Warn(msg string, args ...interface{}) {
	GetGlobalLogger().Warn(msg, args...)
}

// Error logs msg at error level using the package global logger.
// It is a convenience wrapper that delegates to the global logger and accepts
// optional key/value arguments to attach structured fields to the log entry.
func Error(msg string, args ...interface{}) {
	GetGlobalLogger().Error(msg, args...)
}

// WithComponent returns a new *Logger derived from the package global logger with the given
// component name attached as the `component` field for all subsequent log entries.
func WithComponent(component string) *Logger {
	return GetGlobalLogger().WithComponent(component)
}

// WithFields returns a copy of the global logger with the provided structured fields attached.
// The returned *Logger will include these fields on all subsequent log entries emitted from it.
// The input map is used as-is (keys become field names); callers should not rely on it being copied.
func WithFields(fields map[string]interface{}) *Logger {
	return GetGlobalLogger().WithFields(fields)
}