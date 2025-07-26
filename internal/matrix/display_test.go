package matrix

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockClient implements a mock matrix client for testing
type MockClient struct {
	mu                sync.Mutex
	commands          []Command
	brightness        byte
	lastPercentage    byte
	lastPattern       string
	animationEnabled  bool
	connectionError   error
}

func NewMockClient() *MockClient {
	return &MockClient{
		commands: make([]Command, 0),
	}
}

func (m *MockClient) SetBrightness(level byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.connectionError != nil {
		return m.connectionError
	}
	
	m.brightness = level
	m.commands = append(m.commands, BrightnessCommand(level))
	return nil
}

func (m *MockClient) ShowPercentage(percent byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.connectionError != nil {
		return m.connectionError
	}
	
	m.lastPercentage = percent
	m.lastPattern = "percentage"
	m.commands = append(m.commands, PercentageCommand(percent))
	return nil
}

func (m *MockClient) ShowGradient() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.connectionError != nil {
		return m.connectionError
	}
	
	m.lastPattern = "gradient"
	m.commands = append(m.commands, GradientCommand())
	return nil
}

func (m *MockClient) ShowZigZag() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.connectionError != nil {
		return m.connectionError
	}
	
	m.lastPattern = "zigzag"
	m.commands = append(m.commands, ZigZagCommand())
	return nil
}

func (m *MockClient) ShowFullBright() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.connectionError != nil {
		return m.connectionError
	}
	
	m.lastPattern = "fullbright"
	m.commands = append(m.commands, FullBrightCommand())
	return nil
}

func (m *MockClient) SetAnimate(enable bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.connectionError != nil {
		return m.connectionError
	}
	
	m.animationEnabled = enable
	m.commands = append(m.commands, AnimateCommand(enable))
	return nil
}

func (m *MockClient) DrawBitmap(pixels [39]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.connectionError != nil {
		return m.connectionError
	}
	
	m.commands = append(m.commands, DrawBWCommand(pixels))
	return nil
}

func (m *MockClient) StageColumn(col byte, pixels [34]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.connectionError != nil {
		return m.connectionError
	}
	
	m.commands = append(m.commands, StageColCommand(col, pixels))
	return nil
}

func (m *MockClient) FlushColumns() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.connectionError != nil {
		return m.connectionError
	}
	
	m.commands = append(m.commands, FlushColsCommand())
	return nil
}

// Test helper methods
func (m *MockClient) GetCommands() []Command {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Command(nil), m.commands...) // Return copy
}

func (m *MockClient) GetLastPattern() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastPattern
}

func (m *MockClient) GetBrightness() byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.brightness
}

func (m *MockClient) GetLastPercentage() byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastPercentage
}

func (m *MockClient) SetConnectionError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectionError = err
}

func (m *MockClient) ClearCommands() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commands = m.commands[:0]
}

func TestNewDisplayManager(t *testing.T) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	if dm == nil {
		t.Fatal("NewDisplayManager() returned nil")
	}
	
	// Test that client was set (we can't directly compare interface to concrete type)
	if dm.client == nil {
		t.Error("NewDisplayManager() should set the client field")
	}
	
	if dm.updateRate != time.Second {
		t.Errorf("NewDisplayManager() default update rate = %v, want %v", dm.updateRate, time.Second)
	}
	
	if dm.currentState == nil {
		t.Error("NewDisplayManager() currentState not initialized")
	}
}

func TestDisplayManagerSetUpdateRate(t *testing.T) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	newRate := 500 * time.Millisecond
	dm.SetUpdateRate(newRate)
	
	if dm.updateRate != newRate {
		t.Errorf("SetUpdateRate() = %v, want %v", dm.updateRate, newRate)
	}
}

func TestDisplayManagerUpdatePercentage(t *testing.T) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	// Set very short update rate to ensure updates happen
	dm.SetUpdateRate(1 * time.Millisecond)
	time.Sleep(2 * time.Millisecond) // Ensure enough time has passed
	
	tests := []struct {
		name    string
		key     string
		percent float64
		want    byte
	}{
		{"normal percentage", "cpu", 75.5, 75},
		{"zero percentage", "memory", 0.0, 0},
		{"max percentage", "disk", 100.0, 100},
		{"over max percentage", "network", 150.0, 100},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.ClearCommands()
			
			err := dm.UpdatePercentage(tt.key, tt.percent)
			if err != nil {
				t.Errorf("UpdatePercentage() error = %v", err)
				return
			}
			
			if mockClient.GetLastPercentage() != tt.want {
				t.Errorf("UpdatePercentage() sent %d, want %d", mockClient.GetLastPercentage(), tt.want)
			}
			
			state := dm.GetCurrentState()
			if state[tt.key] != tt.percent {
				t.Errorf("UpdatePercentage() state[%s] = %v, want %v", tt.key, state[tt.key], tt.percent)
			}
		})
	}
}

func TestDisplayManagerUpdatePercentageThrottling(t *testing.T) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	// Set long update rate to test throttling
	dm.SetUpdateRate(100 * time.Millisecond)
	
	// First update should work
	err := dm.UpdatePercentage("cpu", 50.0)
	if err != nil {
		t.Errorf("First UpdatePercentage() error = %v", err)
	}
	
	commands := mockClient.GetCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command after first update, got %d", len(commands))
	}
	
	// Second update immediately should be throttled
	mockClient.ClearCommands()
	err = dm.UpdatePercentage("cpu", 60.0)
	if err != nil {
		t.Errorf("Second UpdatePercentage() error = %v", err)
	}
	
	commands = mockClient.GetCommands()
	if len(commands) != 0 {
		t.Errorf("Expected 0 commands after throttled update, got %d", len(commands))
	}
}

func TestDisplayManagerUpdatePercentageMinimalChange(t *testing.T) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	dm.SetUpdateRate(1 * time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	
	// First update
	err := dm.UpdatePercentage("cpu", 50.0)
	if err != nil {
		t.Errorf("First UpdatePercentage() error = %v", err)
	}
	
	time.Sleep(2 * time.Millisecond) // Ensure enough time for next update
	
	// Second update with minimal change should be skipped
	mockClient.ClearCommands()
	err = dm.UpdatePercentage("cpu", 50.5)
	if err != nil {
		t.Errorf("Second UpdatePercentage() error = %v", err)
	}
	
	commands := mockClient.GetCommands()
	if len(commands) != 0 {
		t.Errorf("Expected 0 commands for minimal change, got %d", len(commands))
	}
	
	// Third update with significant change should work
	err = dm.UpdatePercentage("cpu", 55.0)
	if err != nil {
		t.Errorf("Third UpdatePercentage() error = %v", err)
	}
	
	commands = mockClient.GetCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command for significant change, got %d", len(commands))
	}
}

func TestDisplayManagerShowActivity(t *testing.T) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	dm.SetUpdateRate(1 * time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	
	tests := []struct {
		name           string
		active         bool
		expectedPattern string
	}{
		{"show active", true, "zigzag"},
		{"show inactive", false, "gradient"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.ClearCommands()
			
			err := dm.ShowActivity(tt.active)
			if err != nil {
				t.Errorf("ShowActivity() error = %v", err)
				return
			}
			
			if mockClient.GetLastPattern() != tt.expectedPattern {
				t.Errorf("ShowActivity() pattern = %s, want %s", mockClient.GetLastPattern(), tt.expectedPattern)
			}
			
			state := dm.GetCurrentState()
			if state["activity"] != tt.active {
				t.Errorf("ShowActivity() state[activity] = %v, want %v", state["activity"], tt.active)
			}
		})
	}
}

func TestDisplayManagerShowStatus(t *testing.T) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	tests := []struct {
		name            string
		status          string
		expectedPattern string
		expectError     bool
	}{
		{"normal status", "normal", "gradient", false},
		{"warning status", "warning", "zigzag", false},
		{"critical status", "critical", "fullbright", false},
		{"off status", "off", "", false}, // Special case - sets brightness to 0
		{"invalid status", "invalid", "", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.ClearCommands()
			
			err := dm.ShowStatus(tt.status)
			
			if (err != nil) != tt.expectError {
				t.Errorf("ShowStatus() error = %v, expectError %v", err, tt.expectError)
				return
			}
			
			if !tt.expectError {
				if tt.status == "off" {
					// Special case: off status sets brightness to 0
					if mockClient.GetBrightness() != 0 {
						t.Errorf("ShowStatus('off') brightness = %d, want 0", mockClient.GetBrightness())
					}
				} else {
					if mockClient.GetLastPattern() != tt.expectedPattern {
						t.Errorf("ShowStatus() pattern = %s, want %s", mockClient.GetLastPattern(), tt.expectedPattern)
					}
				}
				
				state := dm.GetCurrentState()
				if state["status"] != tt.status {
					t.Errorf("ShowStatus() state[status] = %v, want %v", state["status"], tt.status)
				}
			}
		})
	}
}

func TestDisplayManagerSetBrightness(t *testing.T) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	tests := []byte{0, 50, 128, 255}
	
	for _, level := range tests {
		t.Run("brightness level", func(t *testing.T) {
			mockClient.ClearCommands()
			
			err := dm.SetBrightness(level)
			if err != nil {
				t.Errorf("SetBrightness() error = %v", err)
				return
			}
			
			if mockClient.GetBrightness() != level {
				t.Errorf("SetBrightness() brightness = %d, want %d", mockClient.GetBrightness(), level)
			}
			
			state := dm.GetCurrentState()
			if state["brightness"] != level {
				t.Errorf("SetBrightness() state[brightness] = %v, want %v", state["brightness"], level)
			}
		})
	}
}

func TestDisplayManagerGetCurrentState(t *testing.T) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	dm.SetUpdateRate(1 * time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	
	// Set some state
	dm.UpdatePercentage("cpu", 75.0)
	dm.SetBrightness(128)
	dm.ShowActivity(true)
	dm.ShowStatus("warning")
	
	state := dm.GetCurrentState()
	
	expectedKeys := []string{"cpu", "brightness", "activity", "status"}
	for _, key := range expectedKeys {
		if _, exists := state[key]; !exists {
			t.Errorf("GetCurrentState() missing key %s", key)
		}
	}
	
	if state["cpu"] != 75.0 {
		t.Errorf("GetCurrentState() cpu = %v, want 75.0", state["cpu"])
	}
	
	if state["brightness"] != byte(128) {
		t.Errorf("GetCurrentState() brightness = %v, want 128", state["brightness"])
	}
	
	if state["activity"] != true {
		t.Errorf("GetCurrentState() activity = %v, want true", state["activity"])
	}
	
	if state["status"] != "warning" {
		t.Errorf("GetCurrentState() status = %v, want 'warning'", state["status"])
	}
}

func TestDisplayManagerConcurrency(t *testing.T) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	dm.SetUpdateRate(1 * time.Millisecond)
	
	// Test concurrent access
	var wg sync.WaitGroup
	numGoroutines := 10
	
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < 10; j++ {
				dm.UpdatePercentage("cpu", float64(j*10))
				dm.SetBrightness(byte(j * 25))
				dm.ShowActivity(j%2 == 0)
				
				// Get state without causing race conditions
				state := dm.GetCurrentState()
				if len(state) == 0 {
					t.Errorf("GetCurrentState() returned empty state in goroutine %d", id)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify final state is accessible
	state := dm.GetCurrentState()
	if len(state) == 0 {
		t.Error("GetCurrentState() returned empty state after concurrent operations")
	}
}

func TestDisplayManagerErrorHandling(t *testing.T) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	// Set client to return error
	expectedError := fmt.Errorf("connection failed")
	mockClient.SetConnectionError(expectedError)
	
	dm.SetUpdateRate(1 * time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	
	tests := []struct {
		name string
		op   func() error
	}{
		{"UpdatePercentage", func() error { return dm.UpdatePercentage("cpu", 50.0) }},
		{"ShowActivity", func() error { return dm.ShowActivity(true) }},
		{"ShowStatus", func() error { return dm.ShowStatus("normal") }},
		{"SetBrightness", func() error { return dm.SetBrightness(128) }},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op()
			if err == nil {
				t.Errorf("%s should return error when client fails", tt.name)
			}
		})
	}
}

func TestAbsFunction(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{5.0, 5.0},
		{-5.0, 5.0},
		{0.0, 0.0},
		{-0.0, 0.0},
		{123.456, 123.456},
		{-123.456, 123.456},
	}
	
	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.expected {
			t.Errorf("abs(%f) = %f, want %f", tt.input, result, tt.expected)
		}
	}
}

// Benchmark tests
func BenchmarkDisplayManagerUpdatePercentage(b *testing.B) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	dm.SetUpdateRate(1 * time.Nanosecond) // Allow all updates
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		dm.UpdatePercentage("cpu", float64(i%100))
	}
}

func BenchmarkDisplayManagerGetCurrentState(b *testing.B) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	
	// Set some state
	dm.UpdatePercentage("cpu", 75.0)
	dm.SetBrightness(128)
	dm.ShowActivity(true)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		dm.GetCurrentState()
	}
}

func BenchmarkDisplayManagerConcurrentAccess(b *testing.B) {
	mockClient := NewMockClient()
	dm := NewDisplayManager(mockClient)
	dm.SetUpdateRate(1 * time.Nanosecond)
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 4 {
			case 0:
				dm.UpdatePercentage("cpu", float64(i%100))
			case 1:
				dm.SetBrightness(byte(i % 256))
			case 2:
				dm.ShowActivity(i%2 == 0)
			case 3:
				dm.GetCurrentState()
			}
			i++
		}
	})
}