package main

import (
	"os"
	"testing"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/testutils"
)

func TestShowUsage(t *testing.T) {
	// Test that showUsage doesn't panic and can be called safely
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("showUsage() panicked: %v", r)
		}
	}()

	showUsage()
}

func TestShowConfiguration(t *testing.T) {
	cfg := config.DefaultConfig()
	// Test that showConfiguration doesn't panic and handles valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("showConfiguration() panicked: %v", r)
		}
	}()

	showConfiguration(cfg)
}

func TestLoadConfiguration(t *testing.T) {
	tests := []struct {
		setupConfigEnv func() func()
		name           string
		configPath     string
		expectError    bool
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
			expectError: false, // LoadConfig returns default config for nonexistent files
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
		setup       func() func()
		expectedCfg func(*config.Config) bool
		name        string
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
				defaultBrightness := config.DefaultConfig().Matrix.Brightness

				return cfg.Matrix.Brightness == defaultBrightness
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
	// Skip test in short mode or CI environment
	testutils.SkipIfCI(t, "Integration test")

	cfg := config.DefaultConfig()
	cfg.Matrix.Port = "/dev/null" // Use a safe port that won't cause issues

	// testConnection will fail in test environment since there's no actual hardware
	// But we can test that it handles connection failures gracefully
	err := testConnection(cfg)

	// We expect this to fail in test environment due to no hardware
	if err == nil {
		t.Log("testConnection unexpectedly succeeded - might be running with actual hardware")
		// Verify the error is related to connection, not a panic or unexpected failure
	} else if err.Error() == "" {
		t.Error("Expected meaningful error message from testConnection")
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

// Test flag variables exist and have correct types.
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
// This is tricky to test since main() calls os.Exit(), but we can test the logic.
func TestMainLogic(t *testing.T) {
	// Save original values
	origArgs := os.Args

	defer func() { os.Args = origArgs }()

	// Test version flag handling
	t.Run("version_flag_logic", func(t *testing.T) {
		// Test that version and build constants are properly set
		if version == "" {
			t.Error("Version constant should not be empty")
		}

		if buildTime == "" {
			t.Error("BuildTime constant should not be empty")
		}

		if name != "framework-led-daemon" {
			t.Errorf("Expected name to be 'framework-led-daemon', got %s", name)
		}
	})

	// Test configuration loading logic
	t.Run("configuration_loading_logic", func(t *testing.T) {
		cfg, err := loadConfiguration()
		if err != nil {
			t.Errorf("loadConfiguration() should not fail: %v", err)
		}

		if cfg == nil {
			t.Error("loadConfiguration() should return a valid config")
		}

		// Test command line overrides work
		applyCommandLineOverrides(cfg)

		if cfg == nil {
			t.Error("Config should remain valid after applying overrides")
		}
	})
}
