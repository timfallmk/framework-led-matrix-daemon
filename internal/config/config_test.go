package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Verify matrix defaults
	if cfg.Matrix.BaudRate != 115200 {
		t.Errorf("Expected baud rate 115200, got %d", cfg.Matrix.BaudRate)
	}

	if !cfg.Matrix.AutoDiscover {
		t.Error("Expected auto discover to be true")
	}

	if cfg.Matrix.Brightness != 100 {
		t.Errorf("Expected brightness 100, got %d", cfg.Matrix.Brightness)
	}

	// Verify stats defaults
	if cfg.Stats.CollectInterval != 2*time.Second {
		t.Errorf("Expected collect interval 2s, got %v", cfg.Stats.CollectInterval)
	}

	if !cfg.Stats.EnableCPU || !cfg.Stats.EnableMemory || !cfg.Stats.EnableDisk {
		t.Error("Expected CPU, Memory, and Disk monitoring to be enabled by default")
	}

	if cfg.Stats.EnableNetwork {
		t.Error("Expected Network monitoring to be disabled by default")
	}

	// Verify display defaults
	if cfg.Display.Mode != "percentage" {
		t.Errorf("Expected display mode 'percentage', got %s", cfg.Display.Mode)
	}

	if cfg.Display.PrimaryMetric != "cpu" {
		t.Errorf("Expected primary metric 'cpu', got %s", cfg.Display.PrimaryMetric)
	}

	// Verify thresholds
	thresholds := cfg.Stats.Thresholds
	if thresholds.CPUWarning != 70.0 || thresholds.CPUCritical != 90.0 {
		t.Errorf("Expected CPU thresholds 70/90, got %.1f/%.1f",
			thresholds.CPUWarning, thresholds.CPUCritical)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		config  *Config
		name    string
		errMsg  string
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid baud rate",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Matrix.BaudRate = -1

				return cfg
			}(),
			wantErr: true,
			errMsg:  "matrix baud_rate must be positive",
		},
		{
			name: "invalid collect interval",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Stats.CollectInterval = -1 * time.Second

				return cfg
			}(),
			wantErr: true,
			errMsg:  "stats collect_interval must be positive",
		},
		{
			name: "invalid display mode",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Display.Mode = "invalid"

				return cfg
			}(),
			wantErr: true,
			errMsg:  "invalid display mode: invalid",
		},
		{
			name: "invalid primary metric",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Display.PrimaryMetric = "invalid"

				return cfg
			}(),
			wantErr: true,
			errMsg:  "invalid primary metric: invalid",
		},
		{
			name: "cpu warning >= critical",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Stats.Thresholds.CPUWarning = 95.0
				cfg.Stats.Thresholds.CPUCritical = 90.0

				return cfg
			}(),
			wantErr: true,
			errMsg:  "cpu_warning threshold must be less than cpu_critical",
		},
		{
			name: "memory warning >= critical",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Stats.Thresholds.MemoryWarning = 95.0
				cfg.Stats.Thresholds.MemoryCritical = 90.0

				return cfg
			}(),
			wantErr: true,
			errMsg:  "memory_warning threshold must be less than memory_critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		validate func(*Config) error
		name     string
		yamlData string
		wantErr  bool
	}{
		{
			name: "valid YAML config",
			yamlData: `
matrix:
  baud_rate: 9600
  brightness: 50
stats:
  collect_interval: 5s
  enable_network: true
display:
  mode: stringGradient
  primary_metric: "memory"
`,
			wantErr: false,
			validate: func(cfg *Config) error {
				if cfg.Matrix.BaudRate != 9600 {
					t.Errorf("Expected baud rate 9600, got %d", cfg.Matrix.BaudRate)
				}
				if cfg.Matrix.Brightness != 50 {
					t.Errorf("Expected brightness 50, got %d", cfg.Matrix.Brightness)
				}
				if cfg.Stats.CollectInterval != 5*time.Second {
					t.Errorf("Expected collect interval 5s, got %v", cfg.Stats.CollectInterval)
				}
				if !cfg.Stats.EnableNetwork {
					t.Error("Expected network monitoring to be enabled")
				}
				if cfg.Display.Mode != stringGradient {
					t.Errorf("Expected display mode 'gradient', got %s", cfg.Display.Mode)
				}
				if cfg.Display.PrimaryMetric != "memory" {
					t.Errorf("Expected primary metric 'memory', got %s", cfg.Display.PrimaryMetric)
				}

				return nil
			},
		},
		{
			name: "invalid YAML syntax",
			yamlData: `
matrix:
  baud_rate: invalid_number
`,
			wantErr: true,
		},
		{
			name: "invalid config values",
			yamlData: `
matrix:
  baud_rate: -1
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			configFile := filepath.Join(tmpDir, "test_config.yaml")

			err := os.WriteFile(configFile, []byte(tt.yamlData), 0o644)
			if err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadConfig(configFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr && tt.validate != nil {
				if err := tt.validate(cfg); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path/config.yaml")
	if err != nil {
		t.Errorf("LoadConfig() with non-existent file should return default config, got error: %v", err)
	}

	if cfg == nil {
		t.Error("LoadConfig() should return default config when file doesn't exist")
	}

	// Should be equivalent to default config
	defaultCfg := DefaultConfig()
	if !reflect.DeepEqual(cfg, defaultCfg) {
		t.Error("LoadConfig() with non-existent file should return default config")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultConfig()
	cfg.Matrix.BaudRate = 9600
	cfg.Display.Mode = stringGradient

	configFile := filepath.Join(tmpDir, "saved_config.yaml")

	err := cfg.SaveConfig(configFile)
	if err != nil {
		t.Fatalf("SaveConfig() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("SaveConfig() did not create config file")
	}

	// Load the saved config and verify
	loadedCfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedCfg.Matrix.BaudRate != 9600 {
		t.Errorf("Expected saved baud rate 9600, got %d", loadedCfg.Matrix.BaudRate)
	}

	if loadedCfg.Display.Mode != stringGradient {
		t.Errorf("Expected saved display mode 'gradient', got %s", loadedCfg.Display.Mode)
	}
}

func TestGetConfigPaths(t *testing.T) {
	paths := GetConfigPaths()

	if len(paths) == 0 {
		t.Error("GetConfigPaths() should return at least one path")
	}

	// Should include common paths
	found := false

	for _, path := range paths {
		if filepath.Base(path) == "config.yaml" {
			found = true

			break
		}
	}

	if !found {
		t.Error("GetConfigPaths() should include config.yaml files")
	}
}

func TestFindConfig(t *testing.T) {
	// Create a temporary config file in current directory
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = os.Chdir(originalDir) }()

	_ = os.Chdir(tmpDir)

	// Create configs directory and file
	os.MkdirAll("configs", 0o755)

	configFile := "configs/config.yaml"

	err = os.WriteFile(configFile, []byte("matrix:\n  baud_rate: 115200"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	foundPath, err := FindConfig()
	if err != nil {
		t.Errorf("FindConfig() failed: %v", err)
	}

	if !filepath.IsAbs(foundPath) {
		t.Errorf("FindConfig() should return absolute path, got %s", foundPath)
	}

	if filepath.Base(foundPath) != "config.yaml" {
		t.Errorf("FindConfig() should find config.yaml, got %s", filepath.Base(foundPath))
	}
}

func TestFindConfigNotFound(t *testing.T) {
	// Create empty temporary directory
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = os.Chdir(originalDir) }()

	_ = os.Chdir(tmpDir)

	_, err = FindConfig()
	if err == nil {
		t.Error("FindConfig() should return error when no config file is found")
	}
}

func TestConfigEnvironmentVariables(t *testing.T) {
	// Test XDG_CONFIG_HOME environment variable
	originalXDG := os.Getenv("XDG_CONFIG_HOME")

	defer func() {
		if originalXDG != "" {
			t.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	testDir := "/tmp/test_xdg"
	t.Setenv("XDG_CONFIG_HOME", testDir)

	paths := GetConfigPaths()

	expectedPath := filepath.Join(testDir, "framework-led-daemon", "config.yaml")
	if len(paths) == 0 || paths[0] != expectedPath {
		t.Errorf("Expected first path to be %s, got %v", expectedPath, paths)
	}
}

// Benchmark tests.
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DefaultConfig()
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	cfg := DefaultConfig()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cfg.Validate()
	}
}

func BenchmarkLoadConfig(b *testing.B) {
	// Create temporary config file
	tmpFile, err := os.CreateTemp(b.TempDir(), "benchmark_config_*.yaml")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	yamlData := `
matrix:
  baud_rate: 115200
  brightness: 100
stats:
  collect_interval: 2s
  enable_cpu: true
display:
  mode: "percentage"
  primary_metric: "cpu"
`

	tmpFile.WriteString(yamlData)
	tmpFile.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		LoadConfig(tmpFile.Name())
	}
}
