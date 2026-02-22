package matrix

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"go.bug.st/serial"
)

// MockPort implements a mock serial port for testing.
type MockPort struct {
	writeError  error
	readError   error
	writeData   []byte
	readData    []byte
	readIndex   int
	readTimeout time.Duration
	closed      bool
}

func NewMockPort() *MockPort {
	return &MockPort{
		readData: make([]byte, 0),
	}
}

func (m *MockPort) Write(data []byte) (int, error) {
	if m.writeError != nil {
		return 0, m.writeError
	}

	m.writeData = append(m.writeData, data...)

	return len(data), nil
}

func (m *MockPort) Read(buffer []byte) (int, error) {
	if m.readError != nil {
		return 0, m.readError
	}

	if m.readIndex >= len(m.readData) {
		return 0, errors.New("no more data to read")
	}

	n := copy(buffer, m.readData[m.readIndex:])
	m.readIndex += n

	return n, nil
}

func (m *MockPort) Close() error {
	m.closed = true

	return nil
}

func (m *MockPort) SetReadTimeout(timeout time.Duration) error {
	m.readTimeout = timeout

	return nil
}

func (m *MockPort) Break(d time.Duration) error {
	return nil
}

func (m *MockPort) SetMode(mode *serial.Mode) error {
	return nil
}

func (m *MockPort) SetDTR(dtr bool) error {
	return nil
}

func (m *MockPort) SetRTS(rts bool) error {
	return nil
}

func (m *MockPort) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	return &serial.ModemStatusBits{}, nil
}

func (m *MockPort) Drain() error {
	return nil
}

func (m *MockPort) ResetInputBuffer() error {
	return nil
}

func (m *MockPort) ResetOutputBuffer() error {
	return nil
}

func (m *MockPort) GetWrittenData() []byte {
	return m.writeData
}

func (m *MockPort) SetReadData(data []byte) {
	m.readData = data
	m.readIndex = 0
}

func (m *MockPort) SetWriteError(err error) {
	m.writeError = err
}

func (m *MockPort) SetReadError(err error) {
	m.readError = err
}

func (m *MockPort) IsClosed() bool {
	return m.closed
}

func TestNewClient(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.config == nil {
		t.Error("NewClient() config is nil")
	}

	if client.config.BaudRate != DefaultBaudRate {
		t.Errorf("NewClient() baud rate = %d, want %d", client.config.BaudRate, DefaultBaudRate)
	}

	if client.port != nil {
		t.Error("NewClient() should not have an active port connection")
	}
}

func TestClientSendCommand(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	tests := []struct {
		writeError  error
		name        string
		command     Command
		expectError bool
	}{
		{
			name:        "successful command send",
			command:     BrightnessCommand(128),
			writeError:  nil,
			expectError: false,
		},
		{
			name:        "write error",
			command:     VersionCommand(),
			writeError:  errors.New("write failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPort.writeData = nil // Reset write data
			mockPort.SetWriteError(tt.writeError)

			err := client.SendCommand(tt.command)

			if (err != nil) != tt.expectError {
				t.Errorf("SendCommand() error = %v, expectError %v", err, tt.expectError)

				return
			}

			if !tt.expectError {
				expectedData := tt.command.ToBytes()

				writtenData := mockPort.GetWrittenData()
				if !reflect.DeepEqual(writtenData, expectedData) {
					t.Errorf("SendCommand() wrote %v, want %v", writtenData, expectedData)
				}
			}
		})
	}
}

func TestClientSendCommandNoConnection(t *testing.T) {
	client := NewClient()
	// No port connection established

	err := client.SendCommand(VersionCommand())
	if err == nil {
		t.Error("SendCommand() should return error when not connected")
	}

	if err.Error() != "not connected to any port" {
		t.Errorf("SendCommand() error = %v, want 'not connected to any port'", err)
	}
}

func TestClientReadResponse(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	tests := []struct {
		readError     error
		name          string
		readData      []byte
		expectedBytes int
		expectError   bool
	}{
		{
			name:          "successful read",
			readData:      []byte{1, 2, 3},
			expectedBytes: 3,
			readError:     nil,
			expectError:   false,
		},
		{
			name:          "read error",
			readData:      nil,
			expectedBytes: 3,
			readError:     errors.New("read failed"),
			expectError:   true,
		},
		{
			name:          "partial read",
			readData:      []byte{1, 2},
			expectedBytes: 3,
			readError:     nil,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPort.SetReadData(tt.readData)
			mockPort.SetReadError(tt.readError)

			result, err := client.ReadResponse(tt.expectedBytes)

			if (err != nil) != tt.expectError {
				t.Errorf("ReadResponse() error = %v, expectError %v", err, tt.expectError)

				return
			}

			if !tt.expectError {
				if len(result) != len(tt.readData) {
					t.Errorf("ReadResponse() length = %d, want %d", len(result), len(tt.readData))
				}

				if !reflect.DeepEqual(result, tt.readData) {
					t.Errorf("ReadResponse() = %v, want %v", result, tt.readData)
				}
			}
		})
	}
}

func TestClientGetVersion(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	expectedVersion := []byte{1, 2, 3}
	mockPort.SetReadData(expectedVersion)

	version, err := client.GetVersion()
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}

	// Check that version command was sent
	writtenData := mockPort.GetWrittenData()

	expectedCommand := VersionCommand().ToBytes()
	if !reflect.DeepEqual(writtenData, expectedCommand) {
		t.Errorf("GetVersion() sent %v, want %v", writtenData, expectedCommand)
	}

	// Check returned version data
	if !reflect.DeepEqual(version, expectedVersion) {
		t.Errorf("GetVersion() = %v, want %v", version, expectedVersion)
	}
}

func TestClientSetBrightness(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	level := byte(128)

	err := client.SetBrightness(level)
	if err != nil {
		t.Fatalf("SetBrightness() error = %v", err)
	}

	expectedCommand := BrightnessCommand(level).ToBytes()

	writtenData := mockPort.GetWrittenData()
	if !reflect.DeepEqual(writtenData, expectedCommand) {
		t.Errorf("SetBrightness() sent %v, want %v", writtenData, expectedCommand)
	}
}

func TestClientShowPercentage(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	percent := byte(75)

	err := client.ShowPercentage(percent)
	if err != nil {
		t.Fatalf("ShowPercentage() error = %v", err)
	}

	expectedCommand := PercentageCommand(percent).ToBytes()

	writtenData := mockPort.GetWrittenData()
	if !reflect.DeepEqual(writtenData, expectedCommand) {
		t.Errorf("ShowPercentage() sent %v, want %v", writtenData, expectedCommand)
	}
}

func TestClientShowGradient(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	err := client.ShowGradient()
	if err != nil {
		t.Fatalf("ShowGradient() error = %v", err)
	}

	expectedCommand := GradientCommand().ToBytes()

	writtenData := mockPort.GetWrittenData()
	if !reflect.DeepEqual(writtenData, expectedCommand) {
		t.Errorf("ShowGradient() sent %v, want %v", writtenData, expectedCommand)
	}
}

func TestClientShowZigZag(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	err := client.ShowZigZag()
	if err != nil {
		t.Fatalf("ShowZigZag() error = %v", err)
	}

	expectedCommand := ZigZagCommand().ToBytes()

	writtenData := mockPort.GetWrittenData()
	if !reflect.DeepEqual(writtenData, expectedCommand) {
		t.Errorf("ShowZigZag() sent %v, want %v", writtenData, expectedCommand)
	}
}

func TestClientShowFullBright(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	err := client.ShowFullBright()
	if err != nil {
		t.Fatalf("ShowFullBright() error = %v", err)
	}

	expectedCommand := FullBrightCommand().ToBytes()

	writtenData := mockPort.GetWrittenData()
	if !reflect.DeepEqual(writtenData, expectedCommand) {
		t.Errorf("ShowFullBright() sent %v, want %v", writtenData, expectedCommand)
	}
}

func TestClientSetAnimate(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	tests := []struct {
		name   string
		enable bool
	}{
		{"enable animation", true},
		{"disable animation", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPort.writeData = nil // Reset

			err := client.SetAnimate(tt.enable)
			if err != nil {
				t.Fatalf("SetAnimate() error = %v", err)
			}

			expectedCommand := AnimateCommand(tt.enable).ToBytes()

			writtenData := mockPort.GetWrittenData()
			if !reflect.DeepEqual(writtenData, expectedCommand) {
				t.Errorf("SetAnimate() sent %v, want %v", writtenData, expectedCommand)
			}
		})
	}
}

func TestClientDrawBitmap(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	pixels := [39]byte{}
	for i := range pixels {
		pixels[i] = byte(i)
	}

	err := client.DrawBitmap(pixels)
	if err != nil {
		t.Fatalf("DrawBitmap() error = %v", err)
	}

	expectedCommand := DrawBWCommand(pixels).ToBytes()

	writtenData := mockPort.GetWrittenData()
	if !reflect.DeepEqual(writtenData, expectedCommand) {
		t.Errorf("DrawBitmap() sent %v, want %v", writtenData, expectedCommand)
	}
}

func TestClientStageColumn(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	col := byte(5)

	pixels := [34]byte{}
	for i := range pixels {
		pixels[i] = byte(i + 10)
	}

	err := client.StageColumn(col, pixels)
	if err != nil {
		t.Fatalf("StageColumn() error = %v", err)
	}

	expectedCommand := StageColCommand(col, pixels).ToBytes()

	writtenData := mockPort.GetWrittenData()
	if !reflect.DeepEqual(writtenData, expectedCommand) {
		t.Errorf("StageColumn() sent %v, want %v", writtenData, expectedCommand)
	}
}

func TestClientFlushColumns(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	err := client.FlushColumns()
	if err != nil {
		t.Fatalf("FlushColumns() error = %v", err)
	}

	expectedCommand := FlushColsCommand().ToBytes()

	writtenData := mockPort.GetWrittenData()
	if !reflect.DeepEqual(writtenData, expectedCommand) {
		t.Errorf("FlushColumns() sent %v, want %v", writtenData, expectedCommand)
	}
}

func TestClientDisconnect(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	err := client.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() error = %v", err)
	}

	if !mockPort.IsClosed() {
		t.Error("Disconnect() should close the port")
	}

	if client.port != nil {
		t.Error("Disconnect() should set port to nil")
	}
}

func TestClientDisconnectNoConnection(t *testing.T) {
	client := NewClient()
	// No connection established

	err := client.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() with no connection should not return error, got %v", err)
	}
}

func TestConstants(t *testing.T) {
	if DefaultBaudRate != 115200 {
		t.Errorf("DefaultBaudRate = %d, want 115200", DefaultBaudRate)
	}

	if DefaultTimeout != 1*time.Second {
		t.Errorf("DefaultTimeout = %v, want 1s", DefaultTimeout)
	}
}

// Integration-style tests that test multiple operations together.
func TestClientCommandSequence(t *testing.T) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	// Sequence of commands
	commands := []struct {
		execute func() error
		name    string
	}{
		{func() error { return client.SetBrightness(128) }, "set brightness"},
		{func() error { return client.ShowPercentage(75) }, "show percentage"},
		{func() error { return client.SetAnimate(true) }, "enable animation"},
		{func() error { return client.ShowGradient() }, "show gradient"},
		{func() error { return client.SetAnimate(false) }, "disable animation"},
	}

	for _, cmd := range commands {
		t.Run(cmd.name, func(t *testing.T) {
			if err := cmd.execute(); err != nil {
				t.Errorf("%s failed: %v", cmd.name, err)
			}
		})
	}

	// Verify that commands were sent in correct format
	writtenData := mockPort.GetWrittenData()
	if len(writtenData) == 0 {
		t.Error("No data was written to port")
	}

	// Each command should start with magic bytes
	magicByteCount := 0

	for i := 0; i < len(writtenData)-1; i++ {
		if writtenData[i] == MagicByte1 && writtenData[i+1] == MagicByte2 {
			magicByteCount++
		}
	}

	if magicByteCount != len(commands) {
		t.Errorf("Expected %d command sequences, found %d", len(commands), magicByteCount)
	}
}

// Benchmark tests.
func BenchmarkClientSendCommand(b *testing.B) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	cmd := BrightnessCommand(128)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockPort.writeData = nil // Reset

		client.SendCommand(cmd)
	}
}

func BenchmarkClientShowPercentage(b *testing.B) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockPort.writeData = nil // Reset

		client.ShowPercentage(50)
	}
}

func BenchmarkClientDrawBitmap(b *testing.B) {
	client := NewClient()
	mockPort := NewMockPort()
	client.port = mockPort

	pixels := [39]byte{}
	for i := range pixels {
		pixels[i] = byte(i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockPort.writeData = nil // Reset

		client.DrawBitmap(pixels)
	}
}

// Tests for MultiClient

func TestNewMultiClient(t *testing.T) {
	mc := NewMultiClient()

	if mc == nil {
		t.Fatal("NewMultiClient() returned nil")
	}

	if mc.clients == nil {
		t.Error("NewMultiClient() clients map is nil")
	}

	if mc.config == nil {
		t.Error("NewMultiClient() config map is nil")
	}
}

func TestMultiClientGetClient(t *testing.T) {
	mc := NewMultiClient()
	mockClient := NewClient()
	mockPort := NewMockPort()
	mockClient.port = mockPort

	mc.mu.Lock()
	mc.clients["test-matrix"] = mockClient
	mc.mu.Unlock()

	client := mc.GetClient("test-matrix")
	if client != mockClient {
		t.Error("GetClient() returned wrong client")
	}

	nilClient := mc.GetClient("non-existent")
	if nilClient != nil {
		t.Error("GetClient() should return nil for non-existent client")
	}
}

func TestMultiClientGetClients(t *testing.T) {
	mc := NewMultiClient()
	mockClient1 := NewClient()
	mockClient2 := NewClient()

	mc.mu.Lock()
	mc.clients["matrix1"] = mockClient1
	mc.clients["matrix2"] = mockClient2
	mc.mu.Unlock()

	clients := mc.GetClients()

	if len(clients) != 2 {
		t.Errorf("GetClients() returned %d clients, want 2", len(clients))
	}

	if clients["matrix1"] != mockClient1 {
		t.Error("GetClients() returned wrong client for matrix1")
	}

	if clients["matrix2"] != mockClient2 {
		t.Error("GetClients() returned wrong client for matrix2")
	}

	// Verify it returns a copy, not the internal map
	clients["matrix3"] = NewClient()

	mc.mu.RLock()
	actualCount := len(mc.clients)
	mc.mu.RUnlock()

	if actualCount != 2 {
		t.Error("GetClients() should return a copy, not the internal map")
	}
}

func TestMultiClientGetConfig(t *testing.T) {
	mc := NewMultiClient()
	testConfig := &SingleMatrixConfig{
		Name:       "test-matrix",
		Port:       "/dev/ttyUSB0",
		Role:       "primary",
		Metrics:    []string{"cpu"},
		Brightness: 128,
	}

	mc.mu.Lock()
	mc.config["test-matrix"] = testConfig
	mc.mu.Unlock()

	config := mc.GetConfig("test-matrix")
	if config != testConfig {
		t.Error("GetConfig() returned wrong config")
	}

	nilConfig := mc.GetConfig("non-existent")
	if nilConfig != nil {
		t.Error("GetConfig() should return nil for non-existent config")
	}
}

func TestMultiClientHasMultipleMatrices(t *testing.T) {
	mc := NewMultiClient()

	// No clients
	if mc.HasMultipleMatrices() {
		t.Error("HasMultipleMatrices() should return false with no clients")
	}

	// One client
	mc.mu.Lock()
	mc.clients["matrix1"] = NewClient()
	mc.mu.Unlock()

	if mc.HasMultipleMatrices() {
		t.Error("HasMultipleMatrices() should return false with one client")
	}

	// Two clients
	mc.mu.Lock()
	mc.clients["matrix2"] = NewClient()
	mc.mu.Unlock()

	if !mc.HasMultipleMatrices() {
		t.Error("HasMultipleMatrices() should return true with two clients")
	}
}

func TestMultiClientDisconnect(t *testing.T) {
	mc := NewMultiClient()
	mockPort1 := NewMockPort()
	mockPort2 := NewMockPort()

	client1 := NewClient()
	client1.port = mockPort1
	client2 := NewClient()
	client2.port = mockPort2

	mc.mu.Lock()
	mc.clients["matrix1"] = client1
	mc.clients["matrix2"] = client2
	mc.mu.Unlock()

	err := mc.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() error = %v", err)
	}

	if !mockPort1.IsClosed() {
		t.Error("Disconnect() should close matrix1 port")
	}

	if !mockPort2.IsClosed() {
		t.Error("Disconnect() should close matrix2 port")
	}
}

// Race condition tests - these tests will fail with -race flag if mutex is not properly implemented

func TestMultiClientConcurrentReads(t *testing.T) {
	mc := NewMultiClient()

	// Set up some test data
	mc.mu.Lock()
	mc.clients["matrix1"] = NewClient()
	mc.clients["matrix2"] = NewClient()
	mc.config["matrix1"] = &SingleMatrixConfig{Name: "matrix1"}
	mc.config["matrix2"] = &SingleMatrixConfig{Name: "matrix2"}
	mc.mu.Unlock()

	// Concurrent reads should not cause data races
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = mc.GetClient("matrix1")
				_ = mc.GetClients()
				_ = mc.GetConfig("matrix1")
				_ = mc.HasMultipleMatrices()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMultiClientConcurrentReadWrite(t *testing.T) {
	mc := NewMultiClient()

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 50; i++ {
			mc.mu.Lock()
			mc.clients["test-matrix"] = NewClient()
			mc.config["test-matrix"] = &SingleMatrixConfig{Name: "test-matrix"}
			mc.mu.Unlock()
		}
		done <- true
	}()

	// Reader goroutines
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = mc.GetClient("test-matrix")
				_ = mc.GetClients()
				_ = mc.GetConfig("test-matrix")
				_ = mc.HasMultipleMatrices()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 6; i++ {
		<-done
	}
}

func TestMultiClientConcurrentGetClients(t *testing.T) {
	mc := NewMultiClient()

	mc.mu.Lock()
	mc.clients["matrix1"] = NewClient()
	mc.clients["matrix2"] = NewClient()
	mc.mu.Unlock()

	done := make(chan bool)

	// Multiple goroutines calling GetClients concurrently
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				clients := mc.GetClients()
				// Verify we get a snapshot
				if len(clients) != 2 {
					t.Errorf("Expected 2 clients, got %d", len(clients))
				}
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMultiClientGetClientsReturnsSnapshot(t *testing.T) {
	mc := NewMultiClient()

	mc.mu.Lock()
	mc.clients["matrix1"] = NewClient()
	mc.mu.Unlock()

	// Get snapshot
	snapshot1 := mc.GetClients()

	// Modify the snapshot
	snapshot1["matrix2"] = NewClient()
	snapshot1["matrix3"] = NewClient()

	// Get another snapshot
	snapshot2 := mc.GetClients()

	// Original should only have matrix1
	if len(snapshot2) != 1 {
		t.Errorf("Expected 1 client in original, got %d", len(snapshot2))
	}

	if _, exists := snapshot2["matrix1"]; !exists {
		t.Error("Expected matrix1 to exist in original")
	}

	if _, exists := snapshot2["matrix2"]; exists {
		t.Error("matrix2 should not exist in original after modifying snapshot")
	}
}
