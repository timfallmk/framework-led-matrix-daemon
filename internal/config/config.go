package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Matrix  MatrixConfig  `yaml:"matrix"`
	Stats   StatsConfig   `yaml:"stats"`
	Display DisplayConfig `yaml:"display"`
	Daemon  DaemonConfig  `yaml:"daemon"`
	Logging LoggingConfig `yaml:"logging"`
}

type MatrixConfig struct {
	// Legacy single matrix support
	Port         string        `yaml:"port"`
	BaudRate     int           `yaml:"baud_rate"`
	AutoDiscover bool          `yaml:"auto_discover"`
	Timeout      time.Duration `yaml:"timeout"`
	Brightness   byte          `yaml:"brightness"`

	// Multi-matrix support - using external type to avoid import cycles
	Matrices []map[string]interface{} `yaml:"matrices"`
	DualMode string                   `yaml:"dual_mode"` // "mirror", "split", "extended", "independent"
}

type StatsConfig struct {
	CollectInterval time.Duration `yaml:"collect_interval"`
	EnableCPU       bool          `yaml:"enable_cpu"`
	EnableMemory    bool          `yaml:"enable_memory"`
	EnableDisk      bool          `yaml:"enable_disk"`
	EnableNetwork   bool          `yaml:"enable_network"`
	Thresholds      Thresholds    `yaml:"thresholds"`
}

type Thresholds struct {
	CPUWarning     float64 `yaml:"cpu_warning"`
	CPUCritical    float64 `yaml:"cpu_critical"`
	MemoryWarning  float64 `yaml:"memory_warning"`
	MemoryCritical float64 `yaml:"memory_critical"`
	DiskWarning    float64 `yaml:"disk_warning"`
	DiskCritical   float64 `yaml:"disk_critical"`
}

type DisplayConfig struct {
	UpdateRate      time.Duration            `yaml:"update_rate"`
	Mode            string                   `yaml:"mode"`
	PrimaryMetric   string                   `yaml:"primary_metric"`
	ShowActivity    bool                     `yaml:"show_activity"`
	EnableAnimation bool                     `yaml:"enable_animation"`
	CustomPatterns  map[string]PatternConfig `yaml:"custom_patterns"`
}

type PatternConfig struct {
	Pattern    string                 `yaml:"pattern"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

type DaemonConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	User        string `yaml:"user"`
	Group       string `yaml:"group"`
	PidFile     string `yaml:"pid_file"`
	LogFile     string `yaml:"log_file"`
}

type LoggingConfig struct {
	Level           string `yaml:"level"`             // debug, info, warn, error
	Format          string `yaml:"format"`            // text, json
	Output          string `yaml:"output"`            // stdout, stderr, or file path
	AddSource       bool   `yaml:"add_source"`        // include source file/line in logs
	EventBufferSize int    `yaml:"event_buffer_size"` // Buffer size for async event logging
	// Legacy file logging options (deprecated in favor of structured logging)
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

// Use this as the base configuration before applying file-based loading or environment overrides.
func DefaultConfig() *Config {
	return &Config{
		Matrix: MatrixConfig{
			// Legacy single matrix support
			Port:         "",
			BaudRate:     115200,
			AutoDiscover: true,
			Timeout:      1 * time.Second,
			Brightness:   100,

			// Multi-matrix defaults - empty by default, user can configure
			DualMode: "",
			Matrices: []map[string]interface{}{},
		},
		Stats: StatsConfig{
			CollectInterval: 2 * time.Second,
			EnableCPU:       true,
			EnableMemory:    true,
			EnableDisk:      true,
			EnableNetwork:   false,
			Thresholds: Thresholds{
				CPUWarning:     70.0,
				CPUCritical:    90.0,
				MemoryWarning:  80.0,
				MemoryCritical: 95.0,
				DiskWarning:    80.0,
				DiskCritical:   95.0,
			},
		},
		Display: DisplayConfig{
			UpdateRate:      1 * time.Second,
			Mode:            "percentage",
			PrimaryMetric:   "cpu",
			ShowActivity:    true,
			EnableAnimation: false,
			CustomPatterns:  make(map[string]PatternConfig),
		},
		Daemon: DaemonConfig{
			Name:        "framework-led-daemon",
			Description: "Framework LED Matrix System Statistics Display",
			User:        "",
			Group:       "",
			PidFile:     "/var/run/framework-led-daemon.pid",
			LogFile:     "/var/log/framework-led-daemon.log",
		},
		Logging: LoggingConfig{
			Level:           "info",
			Format:          "text",
			Output:          "stdout",
			AddSource:       true,
			EventBufferSize: 1000,
			// Legacy options with defaults
			File:       "",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = getDefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func (c *Config) SaveConfig(path string) error {
	if path == "" {
		path = getDefaultConfigPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (c *Config) Validate() error {
	if c.Matrix.BaudRate <= 0 {
		return fmt.Errorf("matrix baud_rate must be positive")
	}

	if c.Stats.CollectInterval <= 0 {
		return fmt.Errorf("stats collect_interval must be positive")
	}

	if c.Display.UpdateRate <= 0 {
		return fmt.Errorf("display update_rate must be positive")
	}

	validModes := map[string]bool{
		"percentage": true,
		"gradient":   true,
		"activity":   true,
		"status":     true,
		"custom":     true,
	}
	if !validModes[c.Display.Mode] {
		return fmt.Errorf("invalid display mode: %s", c.Display.Mode)
	}

	validMetrics := map[string]bool{
		"cpu":     true,
		"memory":  true,
		"disk":    true,
		"network": true,
	}
	if !validMetrics[c.Display.PrimaryMetric] {
		return fmt.Errorf("invalid primary metric: %s", c.Display.PrimaryMetric)
	}

	if c.Stats.Thresholds.CPUWarning >= c.Stats.Thresholds.CPUCritical {
		return fmt.Errorf("cpu_warning threshold must be less than cpu_critical")
	}

	if c.Stats.Thresholds.MemoryWarning >= c.Stats.Thresholds.MemoryCritical {
		return fmt.Errorf("memory_warning threshold must be less than memory_critical")
	}

	// Validate dual matrix configuration
	validDualModes := map[string]bool{
		"mirror":      true,
		"split":       true,
		"extended":    true,
		"independent": true,
	}
	if c.Matrix.DualMode != "" && !validDualModes[c.Matrix.DualMode] {
		return fmt.Errorf("invalid dual_mode: %s", c.Matrix.DualMode)
	}

	// Validate individual matrix configurations
	for i, matrix := range c.Matrix.Matrices {
		if role, ok := matrix["role"].(string); ok && role != "" {
			if role != "primary" && role != "secondary" {
				return fmt.Errorf("matrix[%d] invalid role: %s", i, role)
			}
		}

		if metrics, ok := matrix["metrics"].([]interface{}); ok {
			for _, metric := range metrics {
				if metricStr, ok := metric.(string); ok {
					if !validMetrics[metricStr] {
						return fmt.Errorf("matrix[%d] invalid metric: %s", i, metricStr)
					}
				}
			}
		}
	}

	// Validate logging configuration
	if err := c.validateLogging(); err != nil {
		return fmt.Errorf("logging configuration: %w", err)
	}

	return nil
}

func (c *Config) validateLogging() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	validFormats := map[string]bool{
		"text": true,
		"json": true,
	}

	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid logging level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid logging format: %s (must be text or json)", c.Logging.Format)
	}

	// Validate output - can be stdout, stderr, or a file path
	if c.Logging.Output != "" && c.Logging.Output != "stdout" && c.Logging.Output != "stderr" {
		// If it's a file path, check if the directory exists or can be created
		dir := filepath.Dir(c.Logging.Output)
		if dir != "." && dir != "/" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("cannot create log directory %s: %w", dir, err)
			}
		}
	}

	return nil
}

// ConvertMatrices converts the generic matrix configuration to SingleMatrixConfig structs
func (c *Config) ConvertMatrices() []SingleMatrixConfig {
	var matrices []SingleMatrixConfig

	for _, m := range c.Matrix.Matrices {
		matrix := SingleMatrixConfig{}

		if name, ok := m["name"].(string); ok {
			matrix.Name = name
		}
		if port, ok := m["port"].(string); ok {
			matrix.Port = port
		}
		if role, ok := m["role"].(string); ok {
			matrix.Role = role
		}
		if brightness, ok := m["brightness"].(int); ok {
			matrix.Brightness = byte(brightness)
		} else if brightness, ok := m["brightness"].(float64); ok {
			matrix.Brightness = byte(brightness)
		}

		if metrics, ok := m["metrics"].([]interface{}); ok {
			for _, metric := range metrics {
				if metricStr, ok := metric.(string); ok {
					matrix.Metrics = append(matrix.Metrics, metricStr)
				}
			}
		}

		matrices = append(matrices, matrix)
	}

	return matrices
}

// SingleMatrixConfig represents configuration for a single matrix
// This is a separate type to avoid import cycles with the matrix package
type SingleMatrixConfig struct {
	Name       string   `yaml:"name"`
	Port       string   `yaml:"port"`
	Role       string   `yaml:"role"`
	Brightness byte     `yaml:"brightness"`
	Metrics    []string `yaml:"metrics"`
}

func getDefaultConfigPath() string {
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return filepath.Join(configDir, "framework-led-daemon", "config.yaml")
	}

	if homeDir := os.Getenv("HOME"); homeDir != "" {
		return filepath.Join(homeDir, ".config", "framework-led-daemon", "config.yaml")
	}

	return "./config.yaml"
}

func GetConfigPaths() []string {
	var paths []string

	paths = append(paths, getDefaultConfigPath())

	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		paths = append(paths, filepath.Join(configDir, "framework-led-daemon.yaml"))
	}

	paths = append(paths, "/etc/framework-led-daemon/config.yaml")
	paths = append(paths, "/usr/local/etc/framework-led-daemon/config.yaml")
	paths = append(paths, "./configs/config.yaml")

	return paths
}

// FindConfig searches the list of candidate configuration file paths returned by
// GetConfigPaths and returns the absolute path of the first one that exists.
//
// If a candidate exists but its absolute path cannot be resolved, the original
// (non-absolute) path is returned as a fallback. If no config file is found in
// any of the standard locations, an error is returned.
func FindConfig() (string, error) {
	for _, path := range GetConfigPaths() {
		if _, err := os.Stat(path); err == nil {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return path, nil // fallback to original path
			}
			return absPath, nil
		}
	}
	return "", fmt.Errorf("no config file found in standard locations")
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s' (value: %v): %s", e.Field, e.Value, e.Message)
}

// ValidateDetailed performs comprehensive validation with detailed error reporting
func (c *Config) ValidateDetailed() []ValidationError {
	var errors []ValidationError

	// Matrix configuration validation
	if c.Matrix.BaudRate <= 0 {
		errors = append(errors, ValidationError{
			Field:   "matrix.baud_rate",
			Value:   c.Matrix.BaudRate,
			Message: "must be a positive integer (typical values: 9600, 19200, 38400, 57600, 115200)",
		})
	}

	// Stats configuration validation
	if c.Stats.CollectInterval <= 0 {
		errors = append(errors, ValidationError{
			Field:   "stats.collect_interval",
			Value:   c.Stats.CollectInterval,
			Message: "must be a positive duration (e.g., '1s', '500ms')",
		})
	}

	if c.Stats.CollectInterval < 100*time.Millisecond {
		errors = append(errors, ValidationError{
			Field:   "stats.collect_interval",
			Value:   c.Stats.CollectInterval,
			Message: "should be at least 100ms to avoid excessive CPU usage",
		})
	}

	// Display configuration validation
	if c.Display.UpdateRate <= 0 {
		errors = append(errors, ValidationError{
			Field:   "display.update_rate",
			Value:   c.Display.UpdateRate,
			Message: "must be a positive duration (e.g., '1s', '500ms')",
		})
	}

	if c.Display.UpdateRate < 50*time.Millisecond {
		errors = append(errors, ValidationError{
			Field:   "display.update_rate",
			Value:   c.Display.UpdateRate,
			Message: "should be at least 50ms to avoid hardware stress",
		})
	}

	validModes := map[string]bool{
		"percentage": true,
		"gradient":   true,
		"activity":   true,
		"status":     true,
		"custom":     true,
	}
	if !validModes[c.Display.Mode] {
		errors = append(errors, ValidationError{
			Field:   "display.mode",
			Value:   c.Display.Mode,
			Message: "must be one of: percentage, gradient, activity, status, custom",
		})
	}

	validMetrics := map[string]bool{
		"cpu":     true,
		"memory":  true,
		"disk":    true,
		"network": true,
	}
	if !validMetrics[c.Display.PrimaryMetric] {
		errors = append(errors, ValidationError{
			Field:   "display.primary_metric",
			Value:   c.Display.PrimaryMetric,
			Message: "must be one of: cpu, memory, disk, network",
		})
	}

	// Threshold validation with cross-field checks
	if c.Stats.Thresholds.CPUWarning < 0 || c.Stats.Thresholds.CPUWarning > 100 {
		errors = append(errors, ValidationError{
			Field:   "stats.thresholds.cpu_warning",
			Value:   c.Stats.Thresholds.CPUWarning,
			Message: "must be between 0 and 100 (percentage)",
		})
	}

	if c.Stats.Thresholds.CPUCritical < 0 || c.Stats.Thresholds.CPUCritical > 100 {
		errors = append(errors, ValidationError{
			Field:   "stats.thresholds.cpu_critical",
			Value:   c.Stats.Thresholds.CPUCritical,
			Message: "must be between 0 and 100 (percentage)",
		})
	}

	if c.Stats.Thresholds.CPUWarning >= c.Stats.Thresholds.CPUCritical {
		errors = append(errors, ValidationError{
			Field:   "stats.thresholds.cpu_warning",
			Value:   c.Stats.Thresholds.CPUWarning,
			Message: fmt.Sprintf("must be less than cpu_critical (%.1f)", c.Stats.Thresholds.CPUCritical),
		})
	}

	if c.Stats.Thresholds.MemoryWarning < 0 || c.Stats.Thresholds.MemoryWarning > 100 {
		errors = append(errors, ValidationError{
			Field:   "stats.thresholds.memory_warning",
			Value:   c.Stats.Thresholds.MemoryWarning,
			Message: "must be between 0 and 100 (percentage)",
		})
	}

	if c.Stats.Thresholds.MemoryCritical < 0 || c.Stats.Thresholds.MemoryCritical > 100 {
		errors = append(errors, ValidationError{
			Field:   "stats.thresholds.memory_critical",
			Value:   c.Stats.Thresholds.MemoryCritical,
			Message: "must be between 0 and 100 (percentage)",
		})
	}

	if c.Stats.Thresholds.MemoryWarning >= c.Stats.Thresholds.MemoryCritical {
		errors = append(errors, ValidationError{
			Field:   "stats.thresholds.memory_warning",
			Value:   c.Stats.Thresholds.MemoryWarning,
			Message: fmt.Sprintf("must be less than memory_critical (%.1f)", c.Stats.Thresholds.MemoryCritical),
		})
	}

	// Dual matrix validation
	validDualModes := map[string]bool{
		"mirror":      true,
		"split":       true,
		"extended":    true,
		"independent": true,
	}
	if c.Matrix.DualMode != "" && !validDualModes[c.Matrix.DualMode] {
		errors = append(errors, ValidationError{
			Field:   "matrix.dual_mode",
			Value:   c.Matrix.DualMode,
			Message: "must be one of: mirror, split, extended, independent",
		})
	}

	// Individual matrix validation
	for i, matrix := range c.Matrix.Matrices {
		if role, ok := matrix["role"].(string); ok && role != "" {
			if role != "primary" && role != "secondary" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("matrix.matrices[%d].role", i),
					Value:   role,
					Message: "must be either 'primary' or 'secondary'",
				})
			}
		}

		if brightness, ok := matrix["brightness"]; ok {
			var brightnessVal float64
			switch v := brightness.(type) {
			case int:
				brightnessVal = float64(v)
			case float64:
				brightnessVal = v
			default:
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("matrix.matrices[%d].brightness", i),
					Value:   brightness,
					Message: "must be a number between 0 and 255",
				})
				continue
			}

			if brightnessVal < 0 || brightnessVal > 255 {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("matrix.matrices[%d].brightness", i),
					Value:   brightnessVal,
					Message: "must be between 0 and 255",
				})
			}
		}

		if metrics, ok := matrix["metrics"].([]interface{}); ok {
			for j, metric := range metrics {
				if metricStr, ok := metric.(string); ok {
					if !validMetrics[metricStr] {
						errors = append(errors, ValidationError{
							Field:   fmt.Sprintf("matrix.matrices[%d].metrics[%d]", i, j),
							Value:   metricStr,
							Message: "must be one of: cpu, memory, disk, network",
						})
					}
				} else {
					errors = append(errors, ValidationError{
						Field:   fmt.Sprintf("matrix.matrices[%d].metrics[%d]", i, j),
						Value:   metric,
						Message: "must be a string metric name",
					})
				}
			}
		}
	}

	// Daemon configuration validation
	if c.Daemon.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "daemon.name",
			Value:   c.Daemon.Name,
			Message: "cannot be empty",
		})
	}

	// Logging configuration validation
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}
	if c.Logging.Level != "" && !validLogLevels[c.Logging.Level] {
		errors = append(errors, ValidationError{
			Field:   "logging.level",
			Value:   c.Logging.Level,
			Message: "must be one of: debug, info, warn, error, fatal",
		})
	}

	return errors
}

// ApplyEnvironmentOverrides applies environment variable overrides to the configuration
func (c *Config) ApplyEnvironmentOverrides() {
	envOverrides := map[string]func(string){
		"FRAMEWORK_LED_PORT": func(v string) { c.Matrix.Port = v },
		"FRAMEWORK_LED_BAUD_RATE": func(v string) {
			if i, err := strconv.Atoi(v); err == nil {
				c.Matrix.BaudRate = i
			}
		},
		"FRAMEWORK_LED_AUTO_DISCOVER": func(v string) { c.Matrix.AutoDiscover = strings.ToLower(v) == "true" },
		"FRAMEWORK_LED_BRIGHTNESS": func(v string) {
			if i, err := strconv.Atoi(v); err == nil && i >= 0 && i <= 255 {
				c.Matrix.Brightness = byte(i)
			}
		},
		"FRAMEWORK_LED_DUAL_MODE": func(v string) { c.Matrix.DualMode = v },
		"FRAMEWORK_LED_COLLECT_INTERVAL": func(v string) {
			if d, err := time.ParseDuration(v); err == nil {
				c.Stats.CollectInterval = d
			}
		},
		"FRAMEWORK_LED_ENABLE_CPU":     func(v string) { c.Stats.EnableCPU = strings.ToLower(v) == "true" },
		"FRAMEWORK_LED_ENABLE_MEMORY":  func(v string) { c.Stats.EnableMemory = strings.ToLower(v) == "true" },
		"FRAMEWORK_LED_ENABLE_DISK":    func(v string) { c.Stats.EnableDisk = strings.ToLower(v) == "true" },
		"FRAMEWORK_LED_ENABLE_NETWORK": func(v string) { c.Stats.EnableNetwork = strings.ToLower(v) == "true" },
		"FRAMEWORK_LED_UPDATE_RATE": func(v string) {
			if d, err := time.ParseDuration(v); err == nil {
				c.Display.UpdateRate = d
			}
		},
		"FRAMEWORK_LED_DISPLAY_MODE":   func(v string) { c.Display.Mode = v },
		"FRAMEWORK_LED_PRIMARY_METRIC": func(v string) { c.Display.PrimaryMetric = v },
		"FRAMEWORK_LED_SHOW_ACTIVITY":  func(v string) { c.Display.ShowActivity = strings.ToLower(v) == "true" },
		"FRAMEWORK_LED_LOG_LEVEL":      func(v string) { c.Logging.Level = v },
		"FRAMEWORK_LED_LOG_FILE":       func(v string) { c.Logging.File = v },
		"FRAMEWORK_LED_LOG_FORMAT":     func(v string) { c.Logging.Format = v }, // "text" or "json"
		"FRAMEWORK_LED_LOG_OUTPUT":     func(v string) { c.Logging.Output = v }, // "stdout", "stderr", or file path
		"FRAMEWORK_LED_LOG_ADD_SOURCE": func(v string) { c.Logging.AddSource = strings.ToLower(v) == "true" },
		"FRAMEWORK_LED_LOG_EVENT_BUFFER_SIZE": func(v string) {
			if i, err := strconv.Atoi(v); err == nil && i > 0 {
				c.Logging.EventBufferSize = i
			}
		},
	}

	for envVar, applyFunc := range envOverrides {
		if value := os.Getenv(envVar); value != "" {
			applyFunc(value)
		}
	}
}

// ConfigWatcher provides hot-reload functionality for configuration files
type ConfigWatcher struct {
	configPath string
	config     *Config
	mutex      sync.RWMutex
	stopCh     chan struct{}
	reloadCh   chan *Config
	errorCh    chan error
	watcher    *fsnotify.Watcher
}

// NewConfigWatcher returns a new ConfigWatcher configured to monitor the given
// configPath. The watcher is initialized with initialConfig and ready-to-use
// channels for reload notifications and error reporting. The returned watcher
// uses fsnotify for efficient file change detection.
func NewConfigWatcher(configPath string, initialConfig *Config) *ConfigWatcher {
	return &ConfigWatcher{
		configPath: configPath,
		config:     initialConfig,
		stopCh:     make(chan struct{}),
		reloadCh:   make(chan *Config, 1),
		errorCh:    make(chan error, 1),
	}
}

// Start begins watching the configuration file for changes
func (w *ConfigWatcher) Start(ctx context.Context) error {
	// Create file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	w.watcher = watcher

	// Add config file to watcher
	if err := w.watcher.Add(w.configPath); err != nil {
		w.watcher.Close()
		return fmt.Errorf("failed to add config file to watcher: %w", err)
	}

	go w.watchLoop(ctx)
	return nil
}

// Stop stops the configuration watcher
func (w *ConfigWatcher) Stop() {
	close(w.stopCh)
	if w.watcher != nil {
		w.watcher.Close()
	}
}

// GetConfig returns the current configuration (thread-safe)
func (w *ConfigWatcher) GetConfig() *Config {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.config
}

// ReloadChannel returns a channel that receives new configurations
func (w *ConfigWatcher) ReloadChannel() <-chan *Config {
	return w.reloadCh
}

// ErrorChannel returns a channel that receives reload errors
func (w *ConfigWatcher) ErrorChannel() <-chan error {
	return w.errorCh
}

func (w *ConfigWatcher) watchLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case event := <-w.watcher.Events:
			// Only react to Write and Create events
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if err := w.reloadConfig(); err != nil {
					select {
					case w.errorCh <- err:
					default:
						// Error channel is full, skip
					}
				}
			}
		case err := <-w.watcher.Errors:
			select {
			case w.errorCh <- fmt.Errorf("file watcher error: %w", err):
			default:
				// Error channel is full, skip
			}
		}
	}
}

func (w *ConfigWatcher) reloadConfig() error {
	// Load new configuration
	newConfig, err := LoadConfig(w.configPath)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	// Apply environment overrides
	newConfig.ApplyEnvironmentOverrides()

	// Validate new configuration
	if validationErrors := newConfig.ValidateDetailed(); len(validationErrors) > 0 {
		var errorMsgs []string
		for _, ve := range validationErrors {
			errorMsgs = append(errorMsgs, ve.Error())
		}
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errorMsgs, "; "))
	}

	// Update stored configuration
	w.mutex.Lock()
	w.config = newConfig
	w.mutex.Unlock()

	// Notify about reload
	select {
	case w.reloadCh <- newConfig:
	default:
		// Reload channel is full, skip
	}

	return nil
}

// LoadConfigWithEnv loads a configuration from the given path, applies environment
// variable overrides, and runs detailed validation before returning the final
// Config.
//
// If path is empty the loader will use the default config discovery logic used
// by LoadConfig. Environment overrides are applied on top of values loaded
// from file. If any validation issues are found by ValidateDetailed the function
// returns a non-nil error that aggregates all validation messages.
func LoadConfigWithEnv(path string) (*Config, error) {
	config, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	config.ApplyEnvironmentOverrides()

	if validationErrors := config.ValidateDetailed(); len(validationErrors) > 0 {
		var errorMsgs []string
		for _, ve := range validationErrors {
			errorMsgs = append(errorMsgs, ve.Error())
		}
		return nil, fmt.Errorf("configuration validation failed: %s", strings.Join(errorMsgs, "; "))
	}

	return config, nil
}
