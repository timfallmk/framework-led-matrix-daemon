package api

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
)

func TestDeepMergeJSON(t *testing.T) {
	t.Run("merge two flat objects", func(t *testing.T) {
		base := json.RawMessage(`{"a":"1","b":"2"}`)
		patch := json.RawMessage(`{"b":"3","c":"4"}`)

		result, err := deepMergeJSON(base, patch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var m map[string]string
		if err := json.Unmarshal(result, &m); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		if m["a"] != "1" {
			t.Errorf("expected a=1, got %q", m["a"])
		}

		if m["b"] != "3" {
			t.Errorf("expected b=3 (patched), got %q", m["b"])
		}

		if m["c"] != "4" {
			t.Errorf("expected c=4 (new key), got %q", m["c"])
		}
	})

	t.Run("merge nested objects", func(t *testing.T) {
		base := json.RawMessage(`{"outer":{"a":"1","b":"2"}}`)
		patch := json.RawMessage(`{"outer":{"b":"9","c":"3"}}`)

		result, err := deepMergeJSON(base, patch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var m map[string]map[string]string
		if err := json.Unmarshal(result, &m); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		outer := m["outer"]
		if outer["a"] != "1" {
			t.Errorf("expected outer.a=1, got %q", outer["a"])
		}

		if outer["b"] != "9" {
			t.Errorf("expected outer.b=9 (patched), got %q", outer["b"])
		}

		if outer["c"] != "3" {
			t.Errorf("expected outer.c=3 (new), got %q", outer["c"])
		}
	})

	t.Run("patch scalar replaces base object", func(t *testing.T) {
		base := json.RawMessage(`{"a":"1"}`)
		patch := json.RawMessage(`"scalar"`)

		result, err := deepMergeJSON(base, patch)
		if err == nil {
			t.Fatal("expected error when patch is not an object")
		}

		// When patch is not an object, deepMergeJSON returns patch as-is
		var s string
		if err := json.Unmarshal(result, &s); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		if s != "scalar" {
			t.Errorf("expected 'scalar', got %q", s)
		}
	})

	t.Run("base scalar returns patch", func(t *testing.T) {
		base := json.RawMessage(`"scalar"`)
		patch := json.RawMessage(`{"a":"1"}`)

		result, err := deepMergeJSON(base, patch)
		if err == nil {
			t.Fatal("expected error when base is not an object")
		}

		// Returns patch when base is not an object
		var m map[string]string
		if err := json.Unmarshal(result, &m); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		if m["a"] != "1" {
			t.Errorf("expected a=1, got %q", m["a"])
		}
	})

	t.Run("empty objects", func(t *testing.T) {
		base := json.RawMessage(`{}`)
		patch := json.RawMessage(`{"x":"y"}`)

		result, err := deepMergeJSON(base, patch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var m map[string]string
		if err := json.Unmarshal(result, &m); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		if m["x"] != "y" {
			t.Errorf("expected x=y, got %q", m["x"])
		}
	})

	t.Run("patch with nested new key over non-object", func(t *testing.T) {
		base := json.RawMessage(`{"a":"1"}`)
		patch := json.RawMessage(`{"a":{"nested":"value"}}`)

		result, err := deepMergeJSON(base, patch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var m map[string]json.RawMessage
		if err := json.Unmarshal(result, &m); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		var nested map[string]string
		if err := json.Unmarshal(m["a"], &nested); err != nil {
			t.Fatalf("failed to unmarshal nested: %v", err)
		}

		if nested["nested"] != "value" {
			t.Errorf("expected nested=value, got %q", nested["nested"])
		}
	})
}

// setupTestServer creates a server with a client connected to it for handler testing.
func setupTestServer(t *testing.T, scfg ServerConfig) (*Server, *Client) {
	t.Helper()

	socketPath := filepath.Join(t.TempDir(), "test-handler.sock")
	scfg.SocketPath = socketPath

	server := NewServer(scfg)

	ctx, cancel := context.WithCancel(context.Background())

	go func() { _ = server.Serve(ctx) }()

	waitForSocket(t, socketPath)

	client := NewClient(socketPath)
	if err := client.Connect(); err != nil {
		cancel()
		t.Fatalf("failed to connect: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
		cancel()
	})

	return server, client
}

func TestHandleMetricsGetNoCollector(t *testing.T) {
	cfg := config.DefaultConfig()
	_, client := setupTestServer(t, ServerConfig{
		Config:  cfg,
		Display: &mockDisplayController{},
	})

	resp, err := client.Call(MethodMetricsGet, nil)
	if err != nil {
		t.Fatalf("failed to call: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error when collector is nil")
	}

	if resp.Error.Code != ErrCodeInternal {
		t.Errorf("expected error code %d, got %d", ErrCodeInternal, resp.Error.Code)
	}
}

func TestClientGetMetricsError(t *testing.T) {
	cfg := config.DefaultConfig()
	_, client := setupTestServer(t, ServerConfig{
		Config:  cfg,
		Display: &mockDisplayController{},
	})

	_, err := client.GetMetrics()
	if err == nil {
		t.Fatal("expected error when collector is nil")
	}
}

func TestHandleHealthGetNoMonitor(t *testing.T) {
	cfg := config.DefaultConfig()
	_, client := setupTestServer(t, ServerConfig{
		Config:  cfg,
		Display: &mockDisplayController{},
	})

	resp, err := client.Call(MethodHealthGet, nil)
	if err != nil {
		t.Fatalf("failed to call: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error when health monitor is nil")
	}

	if resp.Error.Code != ErrCodeInternal {
		t.Errorf("expected error code %d, got %d", ErrCodeInternal, resp.Error.Code)
	}
}

func TestClientGetHealthError(t *testing.T) {
	cfg := config.DefaultConfig()
	_, client := setupTestServer(t, ServerConfig{
		Config:  cfg,
		Display: &mockDisplayController{},
	})

	_, err := client.GetHealth()
	if err == nil {
		t.Fatal("expected error when health monitor is nil")
	}
}

func TestHandleConfigUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	_, client := setupTestServer(t, ServerConfig{
		Config:  cfg,
		Display: &mockDisplayController{},
	})

	t.Run("valid update", func(t *testing.T) {
		params := map[string]interface{}{
			"display": map[string]interface{}{
				"mode": DisplayModeGradient,
			},
		}

		resp, err := client.Call(MethodConfigUpdate, params)
		if err != nil {
			t.Fatalf("failed to call: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected error: %s", resp.Error.Message)
		}
	})

	t.Run("nil params", func(t *testing.T) {
		resp, err := client.Call(MethodConfigUpdate, nil)
		if err != nil {
			t.Fatalf("failed to call: %v", err)
		}

		if resp.Error == nil {
			t.Fatal("expected error for nil params")
		}

		if resp.Error.Code != ErrCodeInvalidParams {
			t.Errorf("expected error code %d, got %d", ErrCodeInvalidParams, resp.Error.Code)
		}
	})
}

func TestHandleConfigUpdateWithCallback(t *testing.T) {
	cfg := config.DefaultConfig()

	var callbackCalled bool

	server, client := setupTestServer(t, ServerConfig{
		Config:  cfg,
		Display: &mockDisplayController{},
	})

	server.ConfigUpdateFunc = func(c *config.Config) {
		callbackCalled = true
	}

	params := map[string]interface{}{
		"display": map[string]interface{}{
			"mode": DisplayModeGradient,
		},
	}

	resp, err := client.Call(MethodConfigUpdate, params)
	if err != nil {
		t.Fatalf("failed to call: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	if !callbackCalled {
		t.Error("expected ConfigUpdateFunc to be called")
	}
}

func TestHandleMatrixSetDualMode(t *testing.T) {
	cfg := config.DefaultConfig()
	_, client := setupTestServer(t, ServerConfig{
		Config:  cfg,
		Display: &mockDisplayController{},
	})

	t.Run("valid modes", func(t *testing.T) {
		modes := []string{"mirror", "split", "extended", "independent", "single"}
		for _, mode := range modes {
			resp, err := client.Call(MethodMatrixSetDualMode, SetDualModeParams{Mode: mode})
			if err != nil {
				t.Fatalf("failed to call for mode %s: %v", mode, err)
			}

			if resp.Error != nil {
				t.Errorf("unexpected error for mode %s: %s", mode, resp.Error.Message)
			}
		}
	})

	t.Run("invalid mode", func(t *testing.T) {
		resp, err := client.Call(MethodMatrixSetDualMode, SetDualModeParams{Mode: "bogus"})
		if err != nil {
			t.Fatalf("failed to call: %v", err)
		}

		if resp.Error == nil {
			t.Fatal("expected error for invalid mode")
		}

		if resp.Error.Code != ErrCodeInvalidParams {
			t.Errorf("expected error code %d, got %d", ErrCodeInvalidParams, resp.Error.Code)
		}
	})
}

func TestClientSetDualMode(t *testing.T) {
	cfg := config.DefaultConfig()
	_, client := setupTestServer(t, ServerConfig{
		Config:  cfg,
		Display: &mockDisplayController{},
	})

	if err := client.SetDualMode("mirror"); err != nil {
		t.Fatalf("SetDualMode failed: %v", err)
	}

	// Invalid mode should return error
	if err := client.SetDualMode("bogus"); err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestServerClose(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test-close.sock")

	cfg := config.DefaultConfig()

	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Config:     cfg,
		Display:    &mockDisplayController{},
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() {
		errCh <- server.Serve(ctx)
	}()

	waitForSocket(t, socketPath)

	// Close should work without error
	if err := server.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	cancel()

	select {
	case <-errCh:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop in time")
	}

	// Close again when socket already removed should be fine
	if err := server.Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
}

func TestServerUpdateConfig(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test-update-config.sock")

	cfg := config.DefaultConfig()

	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Config:     cfg,
		Display:    &mockDisplayController{},
	})

	newCfg := config.DefaultConfig()
	newCfg.Display.Mode = DisplayModeGradient

	server.UpdateConfig(newCfg)

	got := server.getConfig()
	if got.Display.Mode != DisplayModeGradient {
		t.Errorf("expected mode 'gradient', got %q", got.Display.Mode)
	}
}

func TestHandleMatrixSetDualModeUpdatesConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	server, client := setupTestServer(t, ServerConfig{
		Config:  cfg,
		Display: &mockDisplayController{},
	})

	// Set to mirror mode
	resp, err := client.Call(MethodMatrixSetDualMode, SetDualModeParams{Mode: "mirror"})
	if err != nil {
		t.Fatalf("failed to call: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	got := server.getConfig()
	if got.Matrix.DualMode != "mirror" {
		t.Errorf("expected DualMode 'mirror', got %q", got.Matrix.DualMode)
	}

	// Set back to single — should clear DualMode
	resp, err = client.Call(MethodMatrixSetDualMode, SetDualModeParams{Mode: "single"})
	if err != nil {
		t.Fatalf("failed to call: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	got = server.getConfig()
	if got.Matrix.DualMode != "" {
		t.Errorf("expected DualMode empty after single, got %q", got.Matrix.DualMode)
	}
}
