package visualizer

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/stats"
)

const (
	modePercentage = "percentage"
	modeGradient   = "gradient"
	modeActivity   = "activity"
	modeStatus     = "status"
)

// MockDisplayManager implements a mock display manager for testing.
type MockDisplayManager struct {
	updateError         error
	currentState        map[string]interface{}
	callCounts          map[string]int
	lastPercentageKey   string
	lastStatus          string
	updateRate          time.Duration
	lastPercentageValue float64
	lastActivity        bool
	lastBrightness      byte
}

func NewMockDisplayManager() *MockDisplayManager {
	return &MockDisplayManager{
		currentState: make(map[string]interface{}),
		updateRate:   time.Second,
		callCounts:   make(map[string]int),
	}
}

func (m *MockDisplayManager) UpdatePercentage(key string, percent float64) error {
	m.callCounts["UpdatePercentage"]++
	if m.updateError != nil {
		return m.updateError
	}

	m.lastPercentageKey = key
	m.lastPercentageValue = percent
	m.currentState[key] = percent

	return nil
}

func (m *MockDisplayManager) ShowActivity(active bool) error {
	m.callCounts["ShowActivity"]++
	if m.updateError != nil {
		return m.updateError
	}

	m.lastActivity = active
	m.currentState["activity"] = active

	return nil
}

func (m *MockDisplayManager) ShowStatus(status string) error {
	m.callCounts["ShowStatus"]++
	if m.updateError != nil {
		return m.updateError
	}

	m.lastStatus = status
	m.currentState["status"] = status

	return nil
}

func (m *MockDisplayManager) SetBrightness(level byte) error {
	m.callCounts["SetBrightness"]++
	if m.updateError != nil {
		return m.updateError
	}

	m.lastBrightness = level
	m.currentState["brightness"] = level

	return nil
}

func (m *MockDisplayManager) GetCurrentState() map[string]interface{} {
	m.callCounts["GetCurrentState"]++

	state := make(map[string]interface{})
	for k, v := range m.currentState {
		state[k] = v
	}

	return state
}

func (m *MockDisplayManager) SetUpdateRate(rate time.Duration) {
	m.callCounts["SetUpdateRate"]++
	m.updateRate = rate
}

func (m *MockDisplayManager) SetUpdateError(err error) {
	m.updateError = err
}

func (m *MockDisplayManager) GetCallCount(method string) int {
	return m.callCounts[method]
}

func (m *MockDisplayManager) Reset() {
	m.currentState = make(map[string]interface{})
	m.callCounts = make(map[string]int)
	m.updateError = nil
	m.lastPercentageKey = ""
	m.lastPercentageValue = 0
	m.lastActivity = false
	m.lastStatus = ""
	m.lastBrightness = 0
}

func TestNewVisualizer(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()

	visualizer := NewVisualizer(mockDisplay, cfg)

	if visualizer == nil {
		t.Fatal("NewVisualizer() returned nil")
	}

	if visualizer.display != mockDisplay {
		t.Error("NewVisualizer() display not set correctly")
	}

	if visualizer.config != cfg {
		t.Error("NewVisualizer() config not set correctly")
	}

	if !visualizer.lastUpdate.IsZero() {
		t.Error("NewVisualizer() lastUpdate should be zero initially")
	}
}

func TestVisualizerUpdateDisplayPercentageMode(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	cfg.Display.Mode = modePercentage
	cfg.Display.UpdateRate = 1 * time.Millisecond // Allow frequent updates

	visualizer := NewVisualizer(mockDisplay, cfg)

	tests := []struct {
		name          string
		primaryMetric string
		summary       *stats.StatsSummary
		expectedKey   string
		expectedValue float64
	}{
		{
			name:          "CPU percentage",
			primaryMetric: "cpu",
			summary: &stats.StatsSummary{
				CPUUsage:     75.5,
				MemoryUsage:  60.0,
				DiskActivity: 1024.0,
			},
			expectedKey:   "cpu",
			expectedValue: 75.5,
		},
		{
			name:          "Memory percentage",
			primaryMetric: "memory",
			summary: &stats.StatsSummary{
				CPUUsage:     75.5,
				MemoryUsage:  60.2,
				DiskActivity: 1024.0,
			},
			expectedKey:   "memory",
			expectedValue: 60.2,
		},
		{
			name:          "Disk activity percentage",
			primaryMetric: "disk",
			summary: &stats.StatsSummary{
				CPUUsage:        75.5,
				MemoryUsage:     60.0,
				DiskActivity:    5 * 1024 * 1024, // 5MB/s
				NetworkActivity: 1024.0,
			},
			expectedKey:   "disk",
			expectedValue: 50.0, // Should be normalized
		},
		{
			name:          "Network activity percentage",
			primaryMetric: "network",
			summary: &stats.StatsSummary{
				CPUUsage:        75.5,
				MemoryUsage:     60.0,
				DiskActivity:    1024.0,
				NetworkActivity: 2 * 1024 * 1024, // 2MB/s
			},
			expectedKey:   "network",
			expectedValue: 20.0, // Should be normalized
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDisplay.Reset()

			cfg.Display.PrimaryMetric = tt.primaryMetric
			visualizer.UpdateConfig(cfg)

			time.Sleep(2 * time.Millisecond) // Ensure enough time has passed

			err := visualizer.UpdateDisplay(tt.summary)
			if err != nil {
				t.Errorf("UpdateDisplay() error = %v", err)

				return
			}

			if mockDisplay.lastPercentageKey != tt.expectedKey {
				t.Errorf("UpdateDisplay() key = %s, want %s", mockDisplay.lastPercentageKey, tt.expectedKey)
			}

			if mockDisplay.lastPercentageValue != tt.expectedValue {
				t.Errorf("UpdateDisplay() value = %.1f, want %.1f", mockDisplay.lastPercentageValue, tt.expectedValue)
			}
		})
	}
}

func TestVisualizerUpdateDisplayGradientMode(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	cfg.Display.Mode = modeGradient
	cfg.Display.UpdateRate = 1 * time.Millisecond

	visualizer := NewVisualizer(mockDisplay, cfg)

	summary := &stats.StatsSummary{
		CPUUsage:    50.0,
		MemoryUsage: 40.0,
		Status:      stats.StatusNormal,
	}

	time.Sleep(2 * time.Millisecond)

	err := visualizer.UpdateDisplay(summary)
	if err != nil {
		t.Errorf("UpdateDisplay() error = %v", err)
	}

	if mockDisplay.GetCallCount("ShowStatus") != 1 {
		t.Errorf("UpdateDisplay() should call ShowStatus once, got %d calls", mockDisplay.GetCallCount("ShowStatus"))
	}

	if mockDisplay.lastStatus != "normal" {
		t.Errorf("UpdateDisplay() status = %s, want 'normal'", mockDisplay.lastStatus)
	}
}

func TestVisualizerUpdateDisplayActivityMode(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	cfg.Display.Mode = modeActivity
	cfg.Display.UpdateRate = 1 * time.Millisecond

	visualizer := NewVisualizer(mockDisplay, cfg)

	tests := []struct {
		summary  *stats.StatsSummary
		name     string
		expected bool
	}{
		{
			name: "system active - high CPU",
			summary: &stats.StatsSummary{
				CPUUsage:        15.0, // Above 10% threshold
				DiskActivity:    100.0,
				NetworkActivity: 100.0,
			},
			expected: true,
		},
		{
			name: "system active - high disk activity",
			summary: &stats.StatsSummary{
				CPUUsage:        5.0,
				DiskActivity:    2048.0, // Above 1024 threshold
				NetworkActivity: 100.0,
			},
			expected: true,
		},
		{
			name: "system active - high network activity",
			summary: &stats.StatsSummary{
				CPUUsage:        5.0,
				DiskActivity:    100.0,
				NetworkActivity: 2048.0, // Above 1024 threshold
			},
			expected: true,
		},
		{
			name: "system inactive",
			summary: &stats.StatsSummary{
				CPUUsage:        5.0,   // Below 10% threshold
				DiskActivity:    100.0, // Below 1024 threshold
				NetworkActivity: 100.0, // Below 1024 threshold
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDisplay.Reset()
			time.Sleep(2 * time.Millisecond)

			err := visualizer.UpdateDisplay(tt.summary)
			if err != nil {
				t.Errorf("UpdateDisplay() error = %v", err)

				return
			}

			if mockDisplay.lastActivity != tt.expected {
				t.Errorf("UpdateDisplay() activity = %v, want %v", mockDisplay.lastActivity, tt.expected)
			}
		})
	}
}

func TestVisualizerUpdateDisplayStatusMode(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	cfg.Display.Mode = modeStatus
	cfg.Display.UpdateRate = 1 * time.Millisecond

	visualizer := NewVisualizer(mockDisplay, cfg)

	tests := []struct {
		name     string
		expected string
		status   stats.SystemStatus
	}{
		{"normal status", "normal", stats.StatusNormal},
		{"warning status", "warning", stats.StatusWarning},
		{"critical status", "critical", stats.StatusCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDisplay.Reset()
			time.Sleep(2 * time.Millisecond)

			summary := &stats.StatsSummary{Status: tt.status}

			err := visualizer.UpdateDisplay(summary)
			if err != nil {
				t.Errorf("UpdateDisplay() error = %v", err)

				return
			}

			if mockDisplay.lastStatus != tt.expected {
				t.Errorf("UpdateDisplay() status = %s, want %s", mockDisplay.lastStatus, tt.expected)
			}
		})
	}
}

func TestVisualizerUpdateDisplayCustomMode(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	cfg.Display.Mode = "custom"
	cfg.Display.UpdateRate = 1 * time.Millisecond

	visualizer := NewVisualizer(mockDisplay, cfg)

	summary := &stats.StatsSummary{}

	time.Sleep(2 * time.Millisecond)

	err := visualizer.UpdateDisplay(summary)
	if err == nil {
		t.Error("UpdateDisplay() with custom mode should return error (not implemented)")
	}

	expectedError := "custom mode not yet implemented"
	if err.Error() != expectedError {
		t.Errorf("UpdateDisplay() error = %v, want %v", err.Error(), expectedError)
	}
}

func TestVisualizerUpdateDisplayInvalidMode(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	cfg.Display.Mode = "invalid_mode"
	cfg.Display.UpdateRate = 1 * time.Millisecond

	visualizer := NewVisualizer(mockDisplay, cfg)

	summary := &stats.StatsSummary{}

	time.Sleep(2 * time.Millisecond)

	err := visualizer.UpdateDisplay(summary)
	if err == nil {
		t.Error("UpdateDisplay() with invalid mode should return error")
	}

	expectedError := "unknown display mode: invalid_mode"
	if err.Error() != expectedError {
		t.Errorf("UpdateDisplay() error = %v, want %v", err.Error(), expectedError)
	}
}

func TestVisualizerUpdateThrottling(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	cfg.Display.Mode = modePercentage
	cfg.Display.UpdateRate = 100 * time.Millisecond // Long update rate

	visualizer := NewVisualizer(mockDisplay, cfg)

	summary := &stats.StatsSummary{CPUUsage: 50.0}

	// First update should work
	err := visualizer.UpdateDisplay(summary)
	if err != nil {
		t.Errorf("First UpdateDisplay() error = %v", err)
	}

	if mockDisplay.GetCallCount("UpdatePercentage") != 1 {
		t.Errorf("Expected 1 call to UpdatePercentage, got %d", mockDisplay.GetCallCount("UpdatePercentage"))
	}

	// Second update immediately should be throttled
	mockDisplay.Reset()

	err = visualizer.UpdateDisplay(summary)
	if err != nil {
		t.Errorf("Second UpdateDisplay() error = %v", err)
	}

	if mockDisplay.GetCallCount("UpdatePercentage") != 0 {
		t.Errorf("Expected 0 calls to UpdatePercentage (throttled), got %d", mockDisplay.GetCallCount("UpdatePercentage"))
	}
}

func TestVisualizerNormalizeActivity(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	visualizer := NewVisualizer(mockDisplay, cfg)

	tests := []struct {
		name     string
		activity float64
		expected float64
	}{
		{"zero activity", 0.0, 0.0},
		{"low activity", 1024.0 * 1024.0, 10.0},            // 1MB/s -> 10%
		{"medium activity", 5 * 1024.0 * 1024.0, 50.0},     // 5MB/s -> 50%
		{"max activity", 10 * 1024.0 * 1024.0, 100.0},      // 10MB/s -> 100%
		{"over max activity", 20 * 1024.0 * 1024.0, 100.0}, // 20MB/s -> 100% (capped)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := visualizer.normalizeActivity(tt.activity)
			if result != tt.expected {
				t.Errorf("normalizeActivity(%.1f) = %.1f, want %.1f", tt.activity, result, tt.expected)
			}
		})
	}
}

func TestVisualizerIsSystemActive(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	visualizer := NewVisualizer(mockDisplay, cfg)

	tests := []struct {
		summary  *stats.StatsSummary
		name     string
		expected bool
	}{
		{
			name: "high CPU",
			summary: &stats.StatsSummary{
				CPUUsage:        15.0,
				DiskActivity:    100.0,
				NetworkActivity: 100.0,
			},
			expected: true,
		},
		{
			name: "high disk activity",
			summary: &stats.StatsSummary{
				CPUUsage:        5.0,
				DiskActivity:    2048.0,
				NetworkActivity: 100.0,
			},
			expected: true,
		},
		{
			name: "high network activity",
			summary: &stats.StatsSummary{
				CPUUsage:        5.0,
				DiskActivity:    100.0,
				NetworkActivity: 2048.0,
			},
			expected: true,
		},
		{
			name: "low activity all around",
			summary: &stats.StatsSummary{
				CPUUsage:        5.0,
				DiskActivity:    100.0,
				NetworkActivity: 100.0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := visualizer.isSystemActive(tt.summary)
			if result != tt.expected {
				t.Errorf("isSystemActive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVisualizerShouldAnimate(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	cfg.Display.EnableAnimation = true
	cfg.Stats.Thresholds.CPUWarning = 70.0
	cfg.Stats.Thresholds.MemoryWarning = 80.0

	visualizer := NewVisualizer(mockDisplay, cfg)

	tests := []struct {
		summary  *stats.StatsSummary
		name     string
		expected bool
	}{
		{
			name: "animation disabled",
			summary: &stats.StatsSummary{
				CPUUsage:    75.0,
				MemoryUsage: 85.0,
			},
			expected: false, // Will be overridden by disabled animation
		},
		{
			name: "high CPU usage",
			summary: &stats.StatsSummary{
				CPUUsage:    75.0,
				MemoryUsage: 60.0,
			},
			expected: true,
		},
		{
			name: "high memory usage",
			summary: &stats.StatsSummary{
				CPUUsage:    50.0,
				MemoryUsage: 85.0,
			},
			expected: true,
		},
		{
			name: "high disk activity",
			summary: &stats.StatsSummary{
				CPUUsage:        50.0,
				MemoryUsage:     60.0,
				DiskActivity:    2 * 1024 * 1024, // 2MB/s
				NetworkActivity: 100.0,
			},
			expected: true,
		},
		{
			name: "high network activity",
			summary: &stats.StatsSummary{
				CPUUsage:        50.0,
				MemoryUsage:     60.0,
				DiskActivity:    100.0,
				NetworkActivity: 2 * 1024 * 1024, // 2MB/s
			},
			expected: true,
		},
		{
			name: "low activity all around",
			summary: &stats.StatsSummary{
				CPUUsage:        50.0,
				MemoryUsage:     60.0,
				DiskActivity:    100.0,
				NetworkActivity: 100.0,
			},
			expected: false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Enable animation for all tests except the first one
			cfg.Display.EnableAnimation = (i != 0)
			visualizer.UpdateConfig(cfg)

			result := visualizer.shouldAnimate(tt.summary)
			if result != tt.expected {
				t.Errorf("shouldAnimate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVisualizerCreateCustomPattern(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	visualizer := NewVisualizer(mockDisplay, cfg)

	tests := []struct {
		name      string
		data      []float64
		width     int
		height    int
		expectErr bool
	}{
		{
			name:      "valid pattern",
			width:     3,
			height:    13, // 3*13 = 39
			data:      make([]float64, 39),
			expectErr: false,
		},
		{
			name:      "data length mismatch",
			width:     5,
			height:    5,
			data:      make([]float64, 20), // 5*5 = 25, but providing 20
			expectErr: true,
		},
		{
			name:      "empty data",
			width:     0,
			height:    0,
			data:      []float64{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill data with test values
			for i := range tt.data {
				tt.data[i] = float64(i) / float64(len(tt.data)) // 0.0 to 1.0
			}

			pixels, err := visualizer.CreateCustomPattern(tt.width, tt.height, tt.data)

			if (err != nil) != tt.expectErr {
				t.Errorf("CreateCustomPattern() error = %v, expectErr %v", err, tt.expectErr)

				return
			}

			if !tt.expectErr {
				expectedLen := 39
				if len(pixels) != expectedLen {
					t.Errorf("CreateCustomPattern() pixels length = %d, want %d", len(pixels), expectedLen)
				}

				// Verify pixel values are properly normalized to 0-255
				maxDataLen := len(tt.data)
				if maxDataLen > expectedLen {
					maxDataLen = expectedLen
				}

				for i := 0; i < maxDataLen; i++ {
					expected := byte(tt.data[i] * 255)
					if pixels[i] != expected {
						t.Errorf("CreateCustomPattern() pixels[%d] = %d, want %d", i, pixels[i], expected)

						break // Only show first mismatch
					}
				}
			}
		})
	}
}

func TestVisualizerCreateProgressBar(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	visualizer := NewVisualizer(mockDisplay, cfg)

	tests := []struct {
		name     string
		expected []byte
		percent  float64
		width    int
	}{
		{
			name:     "0% progress",
			percent:  0.0,
			width:    10,
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "50% progress",
			percent:  50.0,
			width:    10,
			expected: []byte{255, 255, 255, 255, 255, 0, 0, 0, 0, 0},
		},
		{
			name:     "100% progress",
			percent:  100.0,
			width:    10,
			expected: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		{
			name:     "partial pixel progress",
			percent:  25.0,
			width:    8,
			expected: []byte{255, 255, 0, 0, 0, 0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := visualizer.CreateProgressBar(tt.percent, tt.width)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("CreateProgressBar() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVisualizerSetBrightness(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	visualizer := NewVisualizer(mockDisplay, cfg)

	levels := []byte{0, 50, 128, 255}

	for _, level := range levels {
		t.Run("brightness level", func(t *testing.T) {
			mockDisplay.Reset()

			err := visualizer.SetBrightness(level)
			if err != nil {
				t.Errorf("SetBrightness() error = %v", err)

				return
			}

			if mockDisplay.lastBrightness != level {
				t.Errorf("SetBrightness() level = %d, want %d", mockDisplay.lastBrightness, level)
			}
		})
	}
}

func TestVisualizerGetCurrentState(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	visualizer := NewVisualizer(mockDisplay, cfg)

	// Set some state in the mock display
	mockDisplay.currentState["cpu"] = 75.0
	mockDisplay.currentState["brightness"] = byte(128)
	mockDisplay.currentState["activity"] = true

	state := visualizer.GetCurrentState()

	if state["cpu"] != 75.0 {
		t.Errorf("GetCurrentState() cpu = %v, want 75.0", state["cpu"])
	}

	if state["brightness"] != byte(128) {
		t.Errorf("GetCurrentState() brightness = %v, want 128", state["brightness"])
	}

	if state["activity"] != true {
		t.Errorf("GetCurrentState() activity = %v, want true", state["activity"])
	}
}

func TestVisualizerUpdateConfig(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	visualizer := NewVisualizer(mockDisplay, cfg)

	// Update config
	newCfg := config.DefaultConfig()
	newCfg.Display.UpdateRate = 2 * time.Second
	newCfg.Matrix.Brightness = 200

	visualizer.UpdateConfig(newCfg)

	if visualizer.config != newCfg {
		t.Error("UpdateConfig() should update the config reference")
	}

	if mockDisplay.GetCallCount("SetUpdateRate") != 1 {
		t.Errorf("UpdateConfig() should call SetUpdateRate once, got %d calls", mockDisplay.GetCallCount("SetUpdateRate"))
	}

	if mockDisplay.updateRate != 2*time.Second {
		t.Errorf("UpdateConfig() update rate = %v, want %v", mockDisplay.updateRate, 2*time.Second)
	}

	if mockDisplay.GetCallCount("SetBrightness") != 1 {
		t.Errorf("UpdateConfig() should call SetBrightness once, got %d calls", mockDisplay.GetCallCount("SetBrightness"))
	}

	if mockDisplay.lastBrightness != 200 {
		t.Errorf("UpdateConfig() brightness = %d, want 200", mockDisplay.lastBrightness)
	}
}

func TestVisualizerErrorHandling(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	cfg.Display.UpdateRate = 1 * time.Millisecond
	visualizer := NewVisualizer(mockDisplay, cfg)

	expectedError := errors.New("display error")
	mockDisplay.SetUpdateError(expectedError)

	summary := &stats.StatsSummary{
		CPUUsage: 50.0,
		Status:   stats.StatusNormal,
	}

	time.Sleep(2 * time.Millisecond)

	tests := []struct {
		name string
		mode string
	}{
		{"percentage mode", "percentage"},
		{"gradient mode", "gradient"},
		{"activity mode", "activity"},
		{"status mode", "status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Display.Mode = tt.mode
			visualizer.UpdateConfig(cfg)

			err := visualizer.UpdateDisplay(summary)
			if err == nil {
				t.Errorf("UpdateDisplay() in %s mode should return error when display fails", tt.mode)
			}
		})
	}
}

func TestVisualizerDrawCustomBitmap(t *testing.T) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	visualizer := NewVisualizer(mockDisplay, cfg)

	pixels := [39]byte{}
	for i := range pixels {
		pixels[i] = byte(i)
	}

	err := visualizer.DrawCustomBitmap(pixels)
	if err == nil {
		t.Error("DrawCustomBitmap() should return error (not implemented)")
	}

	expectedError := "custom bitmap drawing not yet implemented"
	if err.Error() != expectedError {
		t.Errorf("DrawCustomBitmap() error = %v, want %v", err.Error(), expectedError)
	}
}

// Benchmark tests.
func BenchmarkVisualizerUpdateDisplayPercentage(b *testing.B) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	cfg.Display.Mode = modePercentage
	cfg.Display.UpdateRate = 1 * time.Nanosecond // Allow all updates

	visualizer := NewVisualizer(mockDisplay, cfg)

	summary := &stats.StatsSummary{CPUUsage: 75.0}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		visualizer.UpdateDisplay(summary)
	}
}

func BenchmarkVisualizerNormalizeActivity(b *testing.B) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	visualizer := NewVisualizer(mockDisplay, cfg)

	activity := 5.0 * 1024.0 * 1024.0 // 5MB/s

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		visualizer.normalizeActivity(activity)
	}
}

func BenchmarkVisualizerCreateProgressBar(b *testing.B) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	visualizer := NewVisualizer(mockDisplay, cfg)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		visualizer.CreateProgressBar(75.0, 39)
	}
}

func BenchmarkVisualizerIsSystemActive(b *testing.B) {
	mockDisplay := NewMockDisplayManager()
	cfg := config.DefaultConfig()
	visualizer := NewVisualizer(mockDisplay, cfg)

	summary := &stats.StatsSummary{
		CPUUsage:        15.0,
		DiskActivity:    2048.0,
		NetworkActivity: 1024.0,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		visualizer.isSystemActive(summary)
	}
}
