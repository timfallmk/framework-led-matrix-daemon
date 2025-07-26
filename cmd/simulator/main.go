package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/stats"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/visualizer"
)

const (
	LEDWidth  = 34
	LEDHeight = 9
)

func main() {
	var (
		configPath = flag.String("config", "", "Path to configuration file")
		mode       = flag.String("mode", "percentage", "Display mode: percentage, gradient, activity, status")
		metric     = flag.String("metric", "cpu", "Primary metric: cpu, memory, disk, network")
		duration   = flag.Duration("duration", 30*time.Second, "How long to run simulation")
		interval   = flag.Duration("interval", 2*time.Second, "Update interval")
	)
	flag.Parse()

	fmt.Println("🔥 Framework LED Matrix Simulator")
	fmt.Println("=================================")
	fmt.Printf("Mode: %s | Metric: %s | Duration: %v\n\n", *mode, *metric, *duration)

	// Load configuration
	cfg := config.DefaultConfig()
	if *configPath != "" {
		var err error
		cfg, err = config.LoadConfig(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	}

	// Override with command line options
	cfg.Display.Mode = *mode
	cfg.Display.PrimaryMetric = *metric

	// Initialize components
	collector := stats.NewCollector()
	visualizer := visualizer.New(&MockDisplayManager{}, cfg)

	fmt.Println("Starting simulation...")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	start := time.Now()
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Collect real system stats
			systemStats, err := collector.CollectSystemStats()
			if err != nil {
				log.Printf("Error collecting stats: %v", err)
				continue
			}

			// Update display
			err = visualizer.UpdateDisplay(systemStats)
			if err != nil {
				log.Printf("Error updating display: %v", err)
				continue
			}

			// Print current state
			printSimulatedDisplay(systemStats, cfg)

		default:
			if time.Since(start) > *duration {
				fmt.Println("\nSimulation completed!")
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// MockDisplayManager simulates the LED matrix display
type MockDisplayManager struct {
	currentPattern []byte
	brightness     byte
	lastUpdate     time.Time
}

func (m *MockDisplayManager) UpdatePercentage(key string, percent float64) error {
	m.currentPattern = createProgressBar(percent)
	m.lastUpdate = time.Now()
	return nil
}

func (m *MockDisplayManager) ShowActivity(active bool) error {
	if active {
		m.currentPattern = createZigZagPattern()
	} else {
		m.currentPattern = createGradientPattern()
	}
	m.lastUpdate = time.Now()
	return nil
}

func (m *MockDisplayManager) ShowStatus(status string) error {
	switch status {
	case "normal":
		m.currentPattern = createGradientPattern()
	case "warning":
		m.currentPattern = createZigZagPattern()
	case "critical":
		m.currentPattern = createSolidPattern()
	}
	m.lastUpdate = time.Now()
	return nil
}

func (m *MockDisplayManager) SetBrightness(level byte) error {
	m.brightness = level
	return nil
}

func (m *MockDisplayManager) GetCurrentState() map[string]interface{} {
	return map[string]interface{}{
		"brightness":    m.brightness,
		"last_update":   m.lastUpdate,
		"pattern_size":  len(m.currentPattern),
	}
}

// Helper functions to create patterns
func createProgressBar(percentage float64) []byte {
	pattern := make([]byte, LEDWidth*LEDHeight)
	totalPixels := LEDWidth * LEDHeight
	pixelsToFill := int((percentage / 100.0) * float64(totalPixels))
	
	for i := 0; i < pixelsToFill && i < len(pattern); i++ {
		pattern[i] = 1
	}
	return pattern
}

func createZigZagPattern() []byte {
	pattern := make([]byte, LEDWidth*LEDHeight)
	for i := range pattern {
		if (i/LEDWidth+i)%2 == 0 {
			pattern[i] = 1
		}
	}
	return pattern
}

func createGradientPattern() []byte {
	pattern := make([]byte, LEDWidth*LEDHeight)
	center := LEDWidth / 2
	for row := 0; row < LEDHeight; row++ {
		for col := 0; col < LEDWidth; col++ {
			distance := abs(col - center)
			intensity := 1.0 - float64(distance)/float64(center)
			if intensity > 0.3 {
				pattern[row*LEDWidth+col] = 1
			}
		}
	}
	return pattern
}

func createSolidPattern() []byte {
	pattern := make([]byte, LEDWidth*LEDHeight)
	for i := range pattern {
		pattern[i] = 1
	}
	return pattern
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Print ASCII representation of the LED matrix
func printSimulatedDisplay(stats *stats.SystemStats, cfg *config.Config) {
	fmt.Printf("\r\033[2J\033[H") // Clear screen

	fmt.Printf("⏰ %s | Mode: %s | Metric: %s\n", 
		time.Now().Format("15:04:05"), cfg.Display.Mode, cfg.Display.PrimaryMetric)
	
	fmt.Printf("📊 CPU: %.1f%% | Memory: %.1f%% | Disk: %.1f MB/s | Network: %.1f MB/s\n\n",
		stats.CPU.UsagePercent, stats.Memory.UsagePercent, 
		stats.Disk.ReadRate+stats.Disk.WriteRate, 
		stats.Network.ReceiveRate+stats.Network.SendRate)

	// Simulate the LED matrix display
	fmt.Println("🔲 LED Matrix Simulation (34x9):")
	fmt.Println("┌──────────────────────────────────┐")
	
	var pattern []byte
	switch cfg.Display.Mode {
	case "percentage":
		var percentage float64
		switch cfg.Display.PrimaryMetric {
		case "cpu":
			percentage = stats.CPU.UsagePercent
		case "memory":
			percentage = stats.Memory.UsagePercent
		case "disk":
			percentage = (stats.Disk.ReadRate + stats.Disk.WriteRate) / 10.0 // Scale for display
		case "network":
			percentage = (stats.Network.ReceiveRate + stats.Network.SendRate) / 10.0 // Scale for display
		}
		pattern = createProgressBar(percentage)
		
	case "activity":
		isActive := stats.CPU.UsagePercent > 30 || 
					(stats.Disk.ReadRate+stats.Disk.WriteRate) > 1.0 ||
					(stats.Network.ReceiveRate+stats.Network.SendRate) > 1.0
		if isActive {
			pattern = createZigZagPattern()
		} else {
			pattern = createGradientPattern()
		}
		
	case "status":
		status := "normal"
		if stats.CPU.UsagePercent > 80 || stats.Memory.UsagePercent > 90 {
			status = "critical"
		} else if stats.CPU.UsagePercent > 60 || stats.Memory.UsagePercent > 75 {
			status = "warning"
		}
		
		switch status {
		case "normal":
			pattern = createGradientPattern()
		case "warning":
			pattern = createZigZagPattern()
		case "critical":
			pattern = createSolidPattern()
		}
		
	default: // gradient
		pattern = createGradientPattern()
	}
	
	// Render the pattern
	for row := 0; row < LEDHeight; row++ {
		fmt.Print("│")
		for col := 0; col < LEDWidth; col++ {
			if pattern[row*LEDWidth+col] == 1 {
				fmt.Print("█")
			} else {
				fmt.Print("░")
			}
		}
		fmt.Println("│")
	}
	
	fmt.Println("└──────────────────────────────────┘")
	fmt.Printf("\n💡 Brightness: %d/255 | Updates: %s\n", 
		cfg.Matrix.Brightness, cfg.Display.UpdateRate)
}