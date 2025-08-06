package matrix

import (
	"fmt"
	"log"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

type Client struct {
	port   serial.Port
	config *serial.Mode
}

const (
	DefaultBaudRate = 115200
	DefaultTimeout  = 1 * time.Second
)

func NewClient() *Client {
	return &Client{
		config: &serial.Mode{
			BaudRate: DefaultBaudRate,
		},
	}
}

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

func (c *Client) DiscoverPorts() ([]string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate ports: %w", err)
	}

	var frameworkPorts []string

	for _, port := range ports {
		if port.IsUSB {
			log.Printf("Found USB port: %s (VID: %s, PID: %s)",
				port.Name, port.VID, port.PID)

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

	log.Printf("Discovered %d potential LED matrix port(s): %v", len(frameworkPorts), frameworkPorts)
	return frameworkPorts, nil
}

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
	log.Printf("Connected to LED Matrix on port: %s", portName)
	return nil
}

func (c *Client) Disconnect() error {
	if c.port == nil {
		return nil
	}

	err := c.port.Close()
	c.port = nil
	return err
}

func (c *Client) SendCommand(cmd Command) error {
	if c.port == nil {
		return fmt.Errorf("not connected to any port")
	}

	data := cmd.ToBytes()
	_, err := c.port.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write command: %w", err)
	}

	log.Printf("Sent command: ID=0x%02X, Data=%v", cmd.ID, data)
	return nil
}

func (c *Client) ReadResponse(expectedBytes int) ([]byte, error) {
	if c.port == nil {
		return nil, fmt.Errorf("not connected to any port")
	}

	buffer := make([]byte, expectedBytes)
	c.port.SetReadTimeout(DefaultTimeout)

	n, err := c.port.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return buffer[:n], nil
}

func (c *Client) GetVersion() ([]byte, error) {
	if err := c.SendCommand(VersionCommand()); err != nil {
		return nil, err
	}

	return c.ReadResponse(3)
}

func (c *Client) SetBrightness(level byte) error {
	return c.SendCommand(BrightnessCommand(level))
}

func (c *Client) ShowPercentage(percent byte) error {
	return c.SendCommand(PercentageCommand(percent))
}

func (c *Client) ShowGradient() error {
	return c.SendCommand(GradientCommand())
}

func (c *Client) ShowZigZag() error {
	return c.SendCommand(ZigZagCommand())
}

func (c *Client) ShowFullBright() error {
	return c.SendCommand(FullBrightCommand())
}

func (c *Client) SetAnimate(enable bool) error {
	return c.SendCommand(AnimateCommand(enable))
}

func (c *Client) DrawBitmap(pixels [39]byte) error {
	return c.SendCommand(DrawBWCommand(pixels))
}

func (c *Client) StageColumn(col byte, pixels [34]byte) error {
	return c.SendCommand(StageColCommand(col, pixels))
}

func (c *Client) FlushColumns() error {
	return c.SendCommand(FlushColsCommand())
}

// SingleMatrixConfig represents configuration for a single matrix
type SingleMatrixConfig struct {
	Name       string   `yaml:"name"`       // "primary", "secondary", or custom name
	Port       string   `yaml:"port"`       // Specific port or auto-discover if empty
	Role       string   `yaml:"role"`       // "primary", "secondary"
	Brightness byte     `yaml:"brightness"` // Individual brightness control
	Metrics    []string `yaml:"metrics"`    // Which metrics this matrix displays
}

// MultiClient manages multiple LED matrix clients
type MultiClient struct {
	clients map[string]*Client
	config  map[string]*SingleMatrixConfig
}

func NewMultiClient() *MultiClient {
	return &MultiClient{
		clients: make(map[string]*Client),
		config:  make(map[string]*SingleMatrixConfig),
	}
}

func (mc *MultiClient) DiscoverAndConnect(matrices []SingleMatrixConfig, baudRate int) error {
	client := NewClient()
	discoveredPorts, err := client.DiscoverPorts()
	if err != nil {
		return fmt.Errorf("failed to discover ports: %w", err)
	}

	log.Printf("Found %d potential matrix ports, configuring %d matrices",
		len(discoveredPorts), len(matrices))

	for i, matrixConfig := range matrices {
		var portToUse string

		if matrixConfig.Port != "" {
			portToUse = matrixConfig.Port
		} else if i < len(discoveredPorts) {
			portToUse = discoveredPorts[i]
		} else {
			log.Printf("Warning: No port available for matrix %s, skipping", matrixConfig.Name)
			continue
		}

		client := NewClient()
		if err := client.Connect(portToUse); err != nil {
			log.Printf("Warning: Failed to connect to matrix %s on port %s: %v",
				matrixConfig.Name, portToUse, err)
			continue
		}

		if err := client.SetBrightness(matrixConfig.Brightness); err != nil {
			log.Printf("Warning: Failed to set brightness for matrix %s: %v", matrixConfig.Name, err)
		}

		mc.clients[matrixConfig.Name] = client
		configCopy := matrixConfig
		mc.config[matrixConfig.Name] = &configCopy

		log.Printf("Successfully connected matrix %s on port %s", matrixConfig.Name, portToUse)
	}

	if len(mc.clients) == 0 {
		return fmt.Errorf("failed to connect to any LED matrices")
	}

	return nil
}

func (mc *MultiClient) GetClient(name string) *Client {
	return mc.clients[name]
}

func (mc *MultiClient) GetClients() map[string]*Client {
	return mc.clients
}

func (mc *MultiClient) GetConfig(name string) *SingleMatrixConfig {
	return mc.config[name]
}

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

func (mc *MultiClient) HasMultipleMatrices() bool {
	return len(mc.clients) > 1
}
