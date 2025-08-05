package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	Matrices     []map[string]interface{} `yaml:"matrices"`
	DualMode     string                  `yaml:"dual_mode"` // "mirror", "split", "extended", "independent"
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
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

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
			Level:      "info",
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
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
