// Package matrix provides communication interfaces for Framework LED matrix modules.
// It handles serial communication, device discovery, and display command execution.
package matrix

import (
	"fmt"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/logging"
)

// Client manages serial communication with a single LED matrix module.
type Client struct {
	port   serial.Port
	config *serial.Mode
}

// Communication constants for LED matrix modules.
const (
	DefaultBaudRate = 115200
	DefaultTimeout  = 1 * time.Second
)

// NewClient creates a new LED matrix client with default configuration.
func NewClient() *Client {
	return &Client{
		config: &serial.Mode{
			BaudRate: DefaultBaudRate,
		},
	}
}

// DiscoverPort automatically discovers the first available Framework LED matrix port.
func (c *Client) DiscoverPort() (string, error) {
	ports, err := c.DiscoverPorts()
	if err != nil {
		return "", err
	}

	if len(ports) == 0 {
		return "", fmt.Errorf("no Framework LED matrix ports found")
	}

	return ports[0], nil
}

// DiscoverPorts returns all available Framework LED matrix ports.
func (c *Client) DiscoverPorts() ([]string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate ports: %w", err)
	}

	var frameworkPorts []string

	for _, port := range ports {
		if port.IsUSB {
			logging.Info("found USB port", "name", port.Name, "vid", port.VID, "pid", port.PID)

			// Framework LED Matrix has VID 32AC
			if port.VID == "32AC" {
				frameworkPorts = append(frameworkPorts, port.Name)
			}
		}
	}

	if len(frameworkPorts) == 0 {
		// Fallback: return first USB port if no Framework-specific ports found
		for _, port := range ports {
			if port.IsUSB {
				frameworkPorts = append(frameworkPorts, port.Name)
			}
		}
	}

	if len(frameworkPorts) == 0 {
		return nil, fmt.Errorf("no USB ports found")
	}

	logging.Info("discovered potential LED matrix ports", "count", len(frameworkPorts), "ports", frameworkPorts)

	return frameworkPorts, nil
}

// Connect establishes a connection to the LED matrix on the specified port.
// If portName is empty, it automatically discovers the first available port.
func (c *Client) Connect(portName string) error {
	if portName == "" {
		discoveredPort, err := c.DiscoverPort()
		if err != nil {
			return fmt.Errorf("failed to discover port: %w", err)
		}

		portName = discoveredPort
	}

	port, err := serial.Open(portName, c.config)
	if err != nil {
		return fmt.Errorf("failed to open port %s: %w", portName, err)
	}

	c.port = port

	logging.Info("connected to LED matrix", "port", portName)

	return nil
}

// Disconnect closes the connection to the LED matrix.
func (c *Client) Disconnect() error {
	if c.port == nil {
		return nil
	}

	err := c.port.Close()
	c.port = nil

	return err
}

// SendCommand transmits a command to the LED matrix.
func (c *Client) SendCommand(cmd Command) error {
	if c.port == nil {
		return fmt.Errorf("not connected to any port")
	}

	data := cmd.ToBytes()

	_, err := c.port.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write command: %w", err)
	}

	logging.Info("sent command", "id", fmt.Sprintf("0x%02X", cmd.ID), "data", data)

	return nil
}

// ReadResponse reads a response from the LED matrix with the specified number of expected bytes.
func (c *Client) ReadResponse(expectedBytes int) ([]byte, error) {
	if c.port == nil {
		return nil, fmt.Errorf("not connected to any port")
	}

	buffer := make([]byte, expectedBytes)

	if err := c.port.SetReadTimeout(DefaultTimeout); err != nil {
		logging.Warn("failed to set read timeout", "error", err)
	}

	n, err := c.port.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return buffer[:n], nil
}

// GetVersion retrieves the firmware version from the LED matrix.
func (c *Client) GetVersion() ([]byte, error) {
	if err := c.SendCommand(VersionCommand()); err != nil {
		return nil, err
	}

	return c.ReadResponse(3)
}

// SetBrightness sets the brightness level of the LED matrix (0-255).
func (c *Client) SetBrightness(level byte) error {
	return c.SendCommand(BrightnessCommand(level))
}

// ShowPercentage displays a percentage value (0-100) on the LED matrix.
func (c *Client) ShowPercentage(percent byte) error {
	return c.SendCommand(PercentageCommand(percent))
}

// ShowGradient displays a gradient pattern on the LED matrix.
func (c *Client) ShowGradient() error {
	return c.SendCommand(GradientCommand())
}

// ShowZigZag displays a zigzag pattern on the LED matrix.
func (c *Client) ShowZigZag() error {
	return c.SendCommand(ZigZagCommand())
}

// ShowFullBright illuminates all LEDs at maximum brightness.
func (c *Client) ShowFullBright() error {
	return c.SendCommand(FullBrightCommand())
}

// SetAnimate enables or disables animation effects on the LED matrix.
func (c *Client) SetAnimate(enable bool) error {
	return c.SendCommand(AnimateCommand(enable))
}

// DrawBitmap draws a black and white bitmap on the LED matrix using a 39-byte pixel array.
func (c *Client) DrawBitmap(pixels [39]byte) error {
	return c.SendCommand(DrawBWCommand(pixels))
}

// StageColumn stages a column of pixels for display on the LED matrix.
func (c *Client) StageColumn(col byte, pixels [34]byte) error {
	return c.SendCommand(StageColCommand(col, pixels))
}

// FlushColumns applies all staged columns to the LED matrix display.
func (c *Client) FlushColumns() error {
	return c.SendCommand(FlushColsCommand())
}

// SingleMatrixConfig represents configuration for a single matrix.
type SingleMatrixConfig struct {
	Name       string   `yaml:"name"`
	Port       string   `yaml:"port"`
	Role       string   `yaml:"role"`
	Metrics    []string `yaml:"metrics"`
	Brightness byte     `yaml:"brightness"`
}

// MultiClient manages multiple LED matrix clients.
type MultiClient struct {
	clients map[string]*Client
	config  map[string]*SingleMatrixConfig
}

// NewMultiClient creates a new MultiClient for managing multiple LED matrix connections.
func NewMultiClient() *MultiClient {
	return &MultiClient{
		clients: make(map[string]*Client),
		config:  make(map[string]*SingleMatrixConfig),
	}
}

// DiscoverAndConnect discovers available LED matrices and connects to them based on the provided configuration.
func (mc *MultiClient) DiscoverAndConnect(matrices []SingleMatrixConfig, baudRate int) error {
	client := NewClient()

	discoveredPorts, err := client.DiscoverPorts()
	if err != nil {
		return fmt.Errorf("failed to discover ports: %w", err)
	}

	logging.Info("found potential matrix ports", "found", len(discoveredPorts), "configuring", len(matrices))

	for i, matrixConfig := range matrices {
		var portToUse string

		switch {
		case matrixConfig.Port != "":
			portToUse = matrixConfig.Port
		case i < len(discoveredPorts):
			portToUse = discoveredPorts[i]
		default:
			logging.Warn("no port available for matrix, skipping", "matrix", matrixConfig.Name)

			continue
		}

		client := NewClient()
		if err := client.Connect(portToUse); err != nil {
			logging.Warn("failed to connect to matrix", "matrix", matrixConfig.Name, "port", portToUse, "error", err)

			continue
		}

		if err := client.SetBrightness(matrixConfig.Brightness); err != nil {
			logging.Warn("failed to set brightness for matrix", "matrix", matrixConfig.Name, "error", err)
		}

		mc.clients[matrixConfig.Name] = client
		configCopy := matrixConfig
		mc.config[matrixConfig.Name] = &configCopy

		logging.Info("successfully connected matrix", "matrix", matrixConfig.Name, "port", portToUse)
	}

	if len(mc.clients) == 0 {
		return fmt.Errorf("failed to connect to any LED matrices")
	}

	return nil
}

// GetClient returns the Client instance for the specified matrix name.
func (mc *MultiClient) GetClient(name string) *Client {
	return mc.clients[name]
}

// GetClients returns all connected Client instances mapped by matrix name.
func (mc *MultiClient) GetClients() map[string]*Client {
	return mc.clients
}

// GetConfig returns the configuration for the specified matrix name.
func (mc *MultiClient) GetConfig(name string) *SingleMatrixConfig {
	return mc.config[name]
}

// Disconnect closes all matrix connections and returns any errors encountered.
func (mc *MultiClient) Disconnect() error {
	var errors []error

	for name, client := range mc.clients {
		if err := client.Disconnect(); err != nil {
			errors = append(errors, fmt.Errorf("failed to disconnect %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during disconnect: %v", errors)
	}

	return nil
}

// HasMultipleMatrices returns true if more than one matrix is connected.
func (mc *MultiClient) HasMultipleMatrices() bool {
	return len(mc.clients) > 1
}
