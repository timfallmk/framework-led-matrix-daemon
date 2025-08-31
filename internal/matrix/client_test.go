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
