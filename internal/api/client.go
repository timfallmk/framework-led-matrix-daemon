package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Client connects to the daemon API server over a Unix domain socket.
type Client struct {
	conn       net.Conn
	scanner    *bufio.Scanner
	socketPath string
	mu         sync.Mutex
	requestID  atomic.Uint64
}

// NewClient creates a new API client for the given socket path.
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}

	return &Client{
		socketPath: socketPath,
	}
}

// Connect establishes a connection to the daemon API server.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := net.Dialer{Timeout: 5 * time.Second}

	conn, err := dialer.DialContext(context.Background(), "unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon at %s: %w", c.socketPath, err)
	}

	c.conn = conn
	c.scanner = bufio.NewScanner(conn)
	c.scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	return nil
}

// Close closes the connection to the daemon.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.scanner = nil

		return err
	}

	return nil
}

// IsConnected returns true if the client has an active connection.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn != nil
}

// Call sends a request and waits for the response.
func (c *Client) Call(method string, params interface{}) (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	id := fmt.Sprintf("%d", c.requestID.Add(1))

	req := Request{
		Method: method,
		ID:     id,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}

		req.Params = data
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	reqData = append(reqData, '\n')
	if _, err := c.conn.Write(reqData); err != nil {
		c.conn = nil
		c.scanner = nil

		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	if !c.scanner.Scan() {
		err := c.scanner.Err()
		c.conn = nil
		c.scanner = nil

		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		return nil, fmt.Errorf("connection closed")
	}

	var resp Response
	if err := json.Unmarshal(c.scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// Subscribe sends a subscription request and calls the callback for each streamed response.
// It blocks until ctx is cancelled, the connection is closed, or an error occurs.
func (c *Client) Subscribe(ctx context.Context, method string, params interface{}, callback func(*Response)) error {
	c.mu.Lock()

	if c.conn == nil {
		c.mu.Unlock()

		return fmt.Errorf("not connected")
	}

	id := fmt.Sprintf("%d", c.requestID.Add(1))

	req := Request{
		Method: method,
		ID:     id,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			c.mu.Unlock()

			return fmt.Errorf("failed to marshal params: %w", err)
		}

		req.Params = data
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		c.mu.Unlock()

		return fmt.Errorf("failed to marshal request: %w", err)
	}

	reqData = append(reqData, '\n')
	if _, err := c.conn.Write(reqData); err != nil {
		c.conn = nil
		c.scanner = nil
		c.mu.Unlock()

		return fmt.Errorf("failed to send request: %w", err)
	}

	conn := c.conn
	scanner := c.scanner
	c.mu.Unlock()

	// Read the initial ack response
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read subscribe ack: %w", err)
		}

		return fmt.Errorf("connection closed before ack")
	}

	// Verify the ack response
	var ackResp Response
	if err := json.Unmarshal(scanner.Bytes(), &ackResp); err != nil {
		return fmt.Errorf("failed to parse subscribe ack: %w", err)
	}

	if ackResp.Error != nil {
		return fmt.Errorf("subscribe rejected: %s", ackResp.Error.Message)
	}

	// Stream responses
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return fmt.Errorf("failed to set read deadline: %w", err)
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("subscription read error: %w", err)
			}

			return fmt.Errorf("connection closed")
		}

		var resp Response
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			continue
		}

		callback(&resp)
	}
}

// GetMetrics retrieves a one-shot metrics snapshot.
func (c *Client) GetMetrics() (*MetricsResult, error) {
	resp, err := c.Call(MethodMetricsGet, nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("API error: %s", resp.Error.Message)
	}

	var result MetricsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}

	return &result, nil
}

// GetStatus retrieves daemon status information.
func (c *Client) GetStatus() (*StatusResult, error) {
	resp, err := c.Call(MethodStatusGet, nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("API error: %s", resp.Error.Message)
	}

	var result StatusResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	return &result, nil
}

// GetHealth retrieves health check results.
func (c *Client) GetHealth() ([]HealthCheckResult, error) {
	resp, err := c.Call(MethodHealthGet, nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("API error: %s", resp.Error.Message)
	}

	var result []HealthCheckResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse health: %w", err)
	}

	return result, nil
}

// SetDisplayMode changes the display mode on the daemon.
func (c *Client) SetDisplayMode(mode string) error {
	resp, err := c.Call(MethodDisplaySetMode, SetModeParams{Mode: mode})
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("API error: %s", resp.Error.Message)
	}

	return nil
}

// SetBrightness changes the LED brightness on the daemon.
func (c *Client) SetBrightness(level int) error {
	resp, err := c.Call(MethodDisplaySetBright, SetBrightnessParams{Brightness: level})
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("API error: %s", resp.Error.Message)
	}

	return nil
}

// SetPrimaryMetric changes the primary metric on the daemon.
func (c *Client) SetPrimaryMetric(metric string) error {
	resp, err := c.Call(MethodDisplaySetMetric, SetMetricParams{Metric: metric})
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("API error: %s", resp.Error.Message)
	}

	return nil
}

// Reconnect attempts to re-establish the connection.
func (c *Client) Reconnect() error {
	_ = c.Close() //nolint:errcheck // best-effort close before reconnect

	return c.Connect()
}

// SetDualMode changes the dual-matrix mode on the daemon.
func (c *Client) SetDualMode(mode string) error {
	resp, err := c.Call(MethodMatrixSetDualMode, SetDualModeParams{Mode: mode})
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("API error: %s", resp.Error.Message)
	}

	return nil
}
