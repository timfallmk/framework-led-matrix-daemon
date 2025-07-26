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
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return "", fmt.Errorf("failed to enumerate ports: %w", err)
	}

	for _, port := range ports {
		if port.IsUSB {
			log.Printf("Found USB port: %s (VID: %s, PID: %s)",
				port.Name, port.VID, port.PID)
		}
	}

	if len(ports) == 0 {
		return "", fmt.Errorf("no serial ports found")
	}

	for _, port := range ports {
		if port.IsUSB {
			return port.Name, nil
		}
	}

	return ports[0].Name, nil
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
