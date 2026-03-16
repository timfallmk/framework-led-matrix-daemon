package api

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
)

// mockDisplayController implements DisplayController for testing.
type mockDisplayController struct {
	mode       string
	metric     string
	brightness byte
}

func (m *mockDisplayController) SetDisplayMode(mode string) error {
	m.mode = mode

	return nil
}

func (m *mockDisplayController) SetBrightness(level byte) error {
	m.brightness = level

	return nil
}

func (m *mockDisplayController) SetPrimaryMetric(metric string) error {
	m.metric = metric

	return nil
}

func (m *mockDisplayController) GetDisplayState() map[string]interface{} {
	return map[string]interface{}{
		"mode":       m.mode,
		"brightness": m.brightness,
		"metric":     m.metric,
	}
}

func (m *mockDisplayController) IsMultiMatrix() bool {
	return false
}

// waitForSocket polls until the Unix socket at path is connectable or 5 seconds elapses.
func waitForSocket(t *testing.T, path string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		dialer := net.Dialer{Timeout: 20 * time.Millisecond}

		conn, err := dialer.DialContext(context.Background(), "unix", path)
		if err == nil {
			_ = conn.Close()

			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("socket %s not available after 5s", path)
}

func TestServerStartStop(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test-api-server.sock")

	cfg := config.DefaultConfig()
	display := &mockDisplayController{}

	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Config:     cfg,
		Display:    display,
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() {
		errCh <- server.Serve(ctx)
	}()

	waitForSocket(t, socketPath)

	// Stop server
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("server returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop in time")
	}
}

func TestServerClientRoundTrip(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test-api-roundtrip.sock")

	cfg := config.DefaultConfig()
	display := &mockDisplayController{mode: "percentage", brightness: 100, metric: "cpu"}

	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Config:     cfg,
		Display:    display,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Serve(ctx)

	waitForSocket(t, socketPath)

	client := NewClient(socketPath)
	if err := client.Connect(); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	t.Run("StatusGet", func(t *testing.T) {
		status, err := client.GetStatus()
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}

		if status.DisplayMode != "percentage" {
			t.Errorf("expected mode 'percentage', got %q", status.DisplayMode)
		}

		if !status.Connected {
			t.Error("expected connected=true")
		}
	})

	t.Run("ConfigGet", func(t *testing.T) {
		resp, err := client.Call(MethodConfigGet, nil)
		if err != nil {
			t.Fatalf("failed to get config: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected error: %s", resp.Error.Message)
		}

		var cfgResult map[string]interface{}
		if err := json.Unmarshal(resp.Result, &cfgResult); err != nil {
			t.Fatalf("failed to parse config result: %v", err)
		}
	})

	t.Run("DisplaySetMode", func(t *testing.T) {
		if err := client.SetDisplayMode(DisplayModeGradient); err != nil {
			t.Fatalf("failed to set mode: %v", err)
		}

		if display.mode != DisplayModeGradient {
			t.Errorf("expected mode 'gradient', got %q", display.mode)
		}
	})

	t.Run("DisplaySetBrightness", func(t *testing.T) {
		if err := client.SetBrightness(200); err != nil {
			t.Fatalf("failed to set brightness: %v", err)
		}

		if display.brightness != 200 {
			t.Errorf("expected brightness 200, got %d", display.brightness)
		}
	})

	t.Run("DisplaySetMetric", func(t *testing.T) {
		if err := client.SetPrimaryMetric("memory"); err != nil {
			t.Fatalf("failed to set metric: %v", err)
		}

		if display.metric != "memory" {
			t.Errorf("expected metric 'memory', got %q", display.metric)
		}
	})

	t.Run("MatrixGetState", func(t *testing.T) {
		resp, err := client.Call(MethodMatrixGetState, nil)
		if err != nil {
			t.Fatalf("failed to get matrix state: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected error: %s", resp.Error.Message)
		}

		var state map[string]interface{}
		if err := json.Unmarshal(resp.Result, &state); err != nil {
			t.Fatalf("failed to parse state: %v", err)
		}
	})

	t.Run("UnknownMethod", func(t *testing.T) {
		resp, err := client.Call("unknown.method", nil)
		if err != nil {
			t.Fatalf("failed to call: %v", err)
		}

		if resp.Error == nil {
			t.Fatal("expected error for unknown method")
		}

		if resp.Error.Code != ErrCodeInvalidMethod {
			t.Errorf("expected error code %d, got %d", ErrCodeInvalidMethod, resp.Error.Code)
		}
	})
}

func TestClientNotConnected(t *testing.T) {
	client := NewClient("/nonexistent/socket.sock")

	_, err := client.Call(MethodStatusGet, nil)
	if err == nil {
		t.Fatal("expected error when not connected")
	}
}

func TestClientReconnect(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test-api-reconnect.sock")

	cfg := config.DefaultConfig()
	display := &mockDisplayController{}

	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Config:     cfg,
		Display:    display,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Serve(ctx)

	waitForSocket(t, socketPath)

	client := NewClient(socketPath)
	if err := client.Connect(); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Close and reconnect
	client.Close()

	if client.IsConnected() {
		t.Fatal("expected disconnected after Close")
	}

	if err := client.Reconnect(); err != nil {
		t.Fatalf("failed to reconnect: %v", err)
	}

	if !client.IsConnected() {
		t.Fatal("expected connected after Reconnect")
	}

	// Verify connection works
	status, err := client.GetStatus()
	if err != nil {
		t.Fatalf("failed to get status after reconnect: %v", err)
	}

	if status.MatrixMode != MatrixModeSingle {
		t.Errorf("expected matrix mode %q, got %q", MatrixModeSingle, status.MatrixMode)
	}

	client.Close()
}

func TestDefaultSocketPath(t *testing.T) {
	client := NewClient("")
	if client.socketPath != DefaultSocketPath {
		t.Errorf("expected default socket path %q, got %q", DefaultSocketPath, client.socketPath)
	}
}
