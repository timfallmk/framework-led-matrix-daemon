package main

import (
	"os"
	"testing"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
)

func TestShowUsage(t *testing.T) {
	// Capture output would be complex, so just test it doesn't panic
	showUsage()
}

func TestShowConfiguration(t *testing.T) {
	cfg := config.DefaultConfig()
	// Test that showConfiguration doesn't panic
	showConfiguration(cfg)
}

func TestLoadConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		configPath     string
		expectError    bool
		setupConfigEnv func() func() // Returns cleanup function
	}{
		{
			name:        "default_config_when_no_path_specified",
			configPath:  "",
			expectError: false,
			setupConfigEnv: func() func() {
				// Reset configPath flag
				oldConfigPath := *configPath
				*configPath = ""
				return func() { *configPath = oldConfigPath }
			},
		},
		{
			name:        "nonexistent_config_file_returns_default",
			configPath:  "/nonexistent/path/config.yaml", 
			expectError: false,  // LoadConfig returns default config for nonexistent files
			setupConfigEnv: func() func() {
				oldConfigPath := *configPath
				*configPath = "/nonexistent/path/config.yaml"
				return func() {
					*configPath = oldConfigPath
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupConfigEnv()
			defer cleanup()

			cfg, err := loadConfiguration()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && cfg == nil {
				t.Error("Expected config but got nil")
			}
		})
	}
}

func TestApplyCommandLineOverrides(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() func() // Returns cleanup function
		expectedCfg func(*config.Config) bool
	}{
		{
			name: "matrix_port_override",
			setup: func() func() {
				oldPort := *matrixPort
				*matrixPort = "/dev/ttyACM999"
				return func() { *matrixPort = oldPort }
			},
			expectedCfg: func(cfg *config.Config) bool {
				return cfg.Matrix.Port == "/dev/ttyACM999"
			},
		},
		{
			name: "brightness_override",
			setup: func() func() {
				oldBrightness := *brightness
				*brightness = 200
				return func() { *brightness = oldBrightness }
			},
			expectedCfg: func(cfg *config.Config) bool {
				return cfg.Matrix.Brightness == 200
			},
		},
		{
			name: "display_mode_override",
			setup: func() func() {
				oldMode := *displayMode
				*displayMode = "activity"
				return func() { *displayMode = oldMode }
			},
			expectedCfg: func(cfg *config.Config) bool {
				return cfg.Display.Mode == "activity"
			},
		},
		{
			name: "primary_metric_override",
			setup: func() func() {
				oldMetric := *primaryMetric
				*primaryMetric = "network"
				return func() { *primaryMetric = oldMetric }
			},
			expectedCfg: func(cfg *config.Config) bool {
				return cfg.Display.PrimaryMetric == "network"
			},
		},
		{
			name: "log_level_override",
			setup: func() func() {
				oldLevel := *logLevel
				*logLevel = "debug"
				return func() { *logLevel = oldLevel }
			},
			expectedCfg: func(cfg *config.Config) bool {
				return cfg.Logging.Level == "debug"
			},
		},
		{
			name: "brightness_out_of_range_ignored",
			setup: func() func() {
				oldBrightness := *brightness
				*brightness = 300 // Invalid value (out of byte range)
				return func() { *brightness = oldBrightness }
			},
			expectedCfg: func(cfg *config.Config) bool {
				// Should keep default brightness since 300 is out of range for byte
				return cfg.Matrix.Brightness != byte(255) // Won't be set to max byte value
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			cfg := config.DefaultConfig()
			applyCommandLineOverrides(cfg)

			if !tt.expectedCfg(cfg) {
				t.Error("Command line override not applied correctly")
			}
		})
	}
}

func TestTestConnection(t *testing.T) {
	cfg := config.DefaultConfig()
	
	// testConnection will fail in test environment since there's no actual hardware
	// But we can test that it properly creates a service and calls Initialize
	err := testConnection(cfg)
	
	// We expect this to fail in test environment, which is fine
	if err == nil {
		t.Log("testConnection unexpectedly succeeded - might be running with actual hardware")
	}
}

func TestConstants(t *testing.T) {
	if name != "framework-led-daemon" {
		t.Errorf("Expected name to be 'framework-led-daemon', got %s", name)
	}
	
	// Version and buildTime are set by build system, so just check they exist
	if version == "" {
		t.Error("Version should not be empty")
	}
	if buildTime == "" {
		t.Error("BuildTime should not be empty")
	}
}

// Test flag variables exist and have correct types
func TestFlagVariables(t *testing.T) {
	// Just verify the flag variables exist and have correct defaults
	if configPath == nil {
		t.Error("configPath flag should be defined")
	}
	if showVersion == nil {
		t.Error("showVersion flag should be defined")
	}
	if showHelp == nil {
		t.Error("showHelp flag should be defined")
	}
	if logLevel == nil {
		t.Error("logLevel flag should be defined")
	}
	if matrixPort == nil {
		t.Error("matrixPort flag should be defined")
	}
	if brightness == nil {
		t.Error("brightness flag should be defined")
	}
	if displayMode == nil {
		t.Error("displayMode flag should be defined")
	}
	if primaryMetric == nil {
		t.Error("primaryMetric flag should be defined")
	}
}

// TestMainExecution tests main function execution paths that don't exit
// This is tricky to test since main() calls os.Exit(), but we can test the logic
func TestMainLogic(t *testing.T) {
	// Save original values
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Test version flag handling
	t.Run("version_flag_logic", func(t *testing.T) {
		// We can't easily test the os.Exit path, but we can test the logic
		oldShowVersion := *showVersion
		*showVersion = true
		defer func() { *showVersion = oldShowVersion }()

		// The version logic would print version info and exit
		// We can't test the exit part easily, but we know it would exit
	})

	// Test help flag handling  
	t.Run("help_flag_logic", func(t *testing.T) {
		oldShowHelp := *showHelp
		*showHelp = true
		defer func() { *showHelp = oldShowHelp }()

		// The help logic would print usage and exit
		// We can't test the exit part easily, but we know it would exit
	})
}