package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/stats"
)

func TestConstants(t *testing.T) {
	if LEDWidth != 34 {
		t.Errorf("Expected LEDWidth to be 34, got %d", LEDWidth)
	}

	if LEDHeight != 9 {
		t.Errorf("Expected LEDHeight to be 9, got %d", LEDHeight)
	}

	// Version and buildTime are set by build system, so just check they exist
	if version == "" {
		t.Error("Version should not be empty")
	}

	if buildTime == "" {
		t.Error("BuildTime should not be empty")
	}
}

func TestMockDisplayManager(t *testing.T) {
	manager := &MockDisplayManager{}

	t.Run("UpdatePercentage", func(t *testing.T) {
		err := manager.UpdatePercentage("test", 50.0)
		if err != nil {
			t.Errorf("UpdatePercentage failed: %v", err)
		}

		if len(manager.currentPattern) == 0 {
			t.Error("Expected currentPattern to be set")
		}

		if manager.lastUpdate.IsZero() {
			t.Error("Expected lastUpdate to be set")
		}
	})

	t.Run("ShowActivity", func(t *testing.T) {
		err := manager.ShowActivity(true)
		if err != nil {
			t.Errorf("ShowActivity failed: %v", err)
		}

		if len(manager.currentPattern) == 0 {
			t.Error("Expected currentPattern to be set")
		}

		err = manager.ShowActivity(false)
		if err != nil {
			t.Errorf("ShowActivity failed: %v", err)
		}
	})

	t.Run("ShowStatus", func(t *testing.T) {
		testCases := []string{"normal", "warning", "critical", "unknown"}

		for _, status := range testCases {
			err := manager.ShowStatus(status)
			if err != nil {
				t.Errorf("ShowStatus(%s) failed: %v", status, err)
			}

			if len(manager.currentPattern) == 0 {
				t.Error("Expected currentPattern to be set")
			}
		}
	})

	t.Run("SetBrightness", func(t *testing.T) {
		err := manager.SetBrightness(128)
		if err != nil {
			t.Errorf("SetBrightness failed: %v", err)
		}

		if manager.brightness != 128 {
			t.Errorf("Expected brightness to be 128, got %d", manager.brightness)
		}
	})

	t.Run("GetCurrentState", func(t *testing.T) {
		manager.brightness = 200
		manager.lastUpdate = time.Now()
		manager.currentPattern = []byte{1, 2, 3}

		state := manager.GetCurrentState()
		if state["brightness"] != byte(200) {
			t.Errorf("Expected brightness 200, got %v", state["brightness"])
		}

		if state["pattern_size"] != 3 {
			t.Errorf("Expected pattern_size 3, got %v", state["pattern_size"])
		}
	})

	t.Run("SetUpdateRate", func(t *testing.T) {
		// Should not panic
		manager.SetUpdateRate(time.Second)
	})
}

func TestPatternCreation(t *testing.T) {
	t.Run("createProgressBar", func(t *testing.T) {
		pattern := createProgressBar(50.0)

		expectedSize := LEDWidth * LEDHeight
		if len(pattern) != expectedSize {
			t.Errorf("Expected pattern size %d, got %d", expectedSize, len(pattern))
		}

		// Check that some pixels are filled (50% should fill roughly half)
		filledCount := 0

		for _, pixel := range pattern {
			if pixel == 1 {
				filledCount++
			}
		}

		expectedFilled := expectedSize / 2
		if filledCount < expectedFilled-10 || filledCount > expectedFilled+10 {
			t.Errorf("Expected around %d filled pixels, got %d", expectedFilled, filledCount)
		}
	})

	t.Run("createZigZagPattern", func(t *testing.T) {
		pattern := createZigZagPattern()

		expectedSize := LEDWidth * LEDHeight
		if len(pattern) != expectedSize {
			t.Errorf("Expected pattern size %d, got %d", expectedSize, len(pattern))
		}

		// Should have some filled and some empty pixels
		filledCount := 0

		for _, pixel := range pattern {
			if pixel == 1 {
				filledCount++
			}
		}

		if filledCount == 0 || filledCount == expectedSize {
			t.Error("ZigZag pattern should have mixed filled/empty pixels")
		}
	})

	t.Run("createGradientPattern", func(t *testing.T) {
		pattern := createGradientPattern()

		expectedSize := LEDWidth * LEDHeight
		if len(pattern) != expectedSize {
			t.Errorf("Expected pattern size %d, got %d", expectedSize, len(pattern))
		}
	})

	t.Run("createSolidPattern", func(t *testing.T) {
		pattern := createSolidPattern()

		expectedSize := LEDWidth * LEDHeight
		if len(pattern) != expectedSize {
			t.Errorf("Expected pattern size %d, got %d", expectedSize, len(pattern))
		}

		// All pixels should be filled
		for i, pixel := range pattern {
			if pixel != 1 {
				t.Errorf("Expected all pixels to be filled, but pixel %d is %d", i, pixel)
			}
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("abs", func(t *testing.T) {
		tests := []struct {
			input    int
			expected int
		}{
			{5, 5},
			{-5, 5},
			{0, 0},
			{-100, 100},
			{100, 100},
		}

		for _, tt := range tests {
			result := abs(tt.input)
			if result != tt.expected {
				t.Errorf("abs(%d) = %d, expected %d", tt.input, result, tt.expected)
			}
		}
	})
}

func TestPrintSimulatedDisplay(t *testing.T) {
	cfg := config.DefaultConfig()
	summary := &stats.StatsSummary{
		CPUUsage:        50.0,
		MemoryUsage:     70.0,
		DiskActivity:    1024 * 1024,
		NetworkActivity: 512 * 1024,
		Status:          stats.StatusNormal,
	}

	// Test different display modes
	modes := []string{"percentage", "activity", "status", "gradient"}
	metrics := []string{"cpu", "memory", "disk", "network"}

	for _, mode := range modes {
		for _, metric := range metrics {
			t.Run(fmt.Sprintf("mode_%s_metric_%s", mode, metric), func(t *testing.T) {
				cfg.Display.Mode = mode
				cfg.Display.PrimaryMetric = metric

				// Test that printSimulatedDisplay doesn't panic
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("printSimulatedDisplay() panicked with mode=%s, metric=%s: %v", mode, metric, r)
					}
				}()

				printSimulatedDisplay(summary, cfg)
			})
		}
	}
}

func TestMainLogic(t *testing.T) {
	// Test individual components that main() uses
	t.Run("config_loading", func(t *testing.T) {
		cfg := config.DefaultConfig()

		// Verify default values are reasonable
		if cfg.Display.Mode == "" {
			t.Error("Default config should have a display mode")
		}

		if cfg.Display.PrimaryMetric == "" {
			t.Error("Default config should have a primary metric")
		}
	})

	t.Run("mock_display_manager_integration", func(t *testing.T) {
		mockDM := &MockDisplayManager{}
		cfg := config.DefaultConfig()

		// Test that we can create components like main() does
		if cfg.Display.Mode == "" {
			cfg.Display.Mode = "percentage"
		}

		if cfg.Display.PrimaryMetric == "" {
			cfg.Display.PrimaryMetric = "cpu"
		}

		// Test mock display manager methods work properly
		err := mockDM.UpdatePercentage("test", 50.0)
		if err != nil {
			t.Errorf("Mock display manager UpdatePercentage failed: %v", err)
		}

		// Verify mock display manager state was updated
		if len(mockDM.currentPattern) == 0 {
			t.Error("Mock display manager should have a pattern after UpdatePercentage")
		}

		if mockDM.lastUpdate.IsZero() {
			t.Error("Mock display manager should have updated lastUpdate timestamp")
		}
	})

	t.Run("constants_validation", func(t *testing.T) {
		// Test that LED dimensions are positive and reasonable
		if LEDWidth <= 0 {
			t.Errorf("LEDWidth should be positive, got %d", LEDWidth)
		}

		if LEDHeight <= 0 {
			t.Errorf("LEDHeight should be positive, got %d", LEDHeight)
		}

		// Test that version info is set
		if version == "" {
			t.Error("Version should not be empty")
		}

		if buildTime == "" {
			t.Error("BuildTime should not be empty")
		}
	})
}
