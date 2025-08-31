package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/daemon"
)

const (
	name = "framework-led-daemon"
)

var (
	// These are set by the build system via -ldflags.
	version   = "dev"     // Set via -X main.version=...
	buildTime = "unknown" // Set via -X main.buildTime=...
)

var (
	configPath    = flag.String("config", "", "Path to configuration file")
	showVersion   = flag.Bool("version", false, "Show version information")
	showHelp      = flag.Bool("help", false, "Show help information")
	logLevel      = flag.String("log-level", "", "Set log level (debug, info, warn, error)")
	matrixPort    = flag.String("port", "", "Serial port for LED matrix")
	brightness    = flag.Int("brightness", -1, "LED brightness (0-255)")
	displayMode   = flag.String("mode", "", "Display mode (percentage, gradient, activity, status)")
	primaryMetric = flag.String("metric", "", "Primary metric to display (cpu, memory, disk, network)")
)

func main() {
	flag.Parse()

	if *showHelp {
		showUsage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("%s version %s\n", name, version)
		fmt.Printf("Build time: %s\n", buildTime)
		os.Exit(0)
	}

	cfg, err := loadConfiguration()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	applyCommandLineOverrides(cfg)

	if flag.NArg() < 1 {
		showUsage()
		os.Exit(1)
	}

	args := flag.Args()
	command := args[0]

	service, err := daemon.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	switch command {
	case "run":
		if err := service.Run(); err != nil {
			log.Fatalf("Failed to run service: %v", err)
		}
	case "install":
		status, err := service.Install()
		if err != nil {
			log.Fatalf("Failed to install service: %v", err)
		}

		fmt.Println(status)
	case "remove", "uninstall":
		status, err := service.Remove()
		if err != nil {
			log.Fatalf("Failed to remove service: %v", err)
		}

		fmt.Println(status)
	case "start":
		status, err := service.StartService()
		if err != nil {
			log.Fatalf("Failed to start service: %v", err)
		}

		fmt.Println(status)
	case "stop":
		status, err := service.StopService()
		if err != nil {
			log.Fatalf("Failed to stop service: %v", err)
		}

		fmt.Println(status)
	case "status":
		status, err := service.Status()
		if err != nil {
			log.Fatalf("Failed to get service status: %v", err)
		}

		fmt.Println(status)
	case "config":
		showConfiguration(cfg)
	case "test":
		if err := testConnection(cfg); err != nil {
			log.Fatalf("Connection test failed: %v", err)
		}

		fmt.Println("Connection test successful!")
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		showUsage()
		os.Exit(1)
	}
}

func loadConfiguration() (*config.Config, error) {
	if *configPath != "" {
		return config.LoadConfig(*configPath)
	}

	configFile, err := config.FindConfig()
	if err != nil {
		log.Printf("No configuration file found, using defaults")

		return config.DefaultConfig(), nil //nolint:nilerr
	}

	return config.LoadConfig(configFile)
}

func applyCommandLineOverrides(cfg *config.Config) {
	if *matrixPort != "" {
		cfg.Matrix.Port = *matrixPort
	}

	if *brightness >= 0 && *brightness <= 255 {
		cfg.Matrix.Brightness = byte(*brightness)
	}

	if *displayMode != "" {
		cfg.Display.Mode = *displayMode
	}

	if *primaryMetric != "" {
		cfg.Display.PrimaryMetric = *primaryMetric
	}

	if *logLevel != "" {
		cfg.Logging.Level = *logLevel
	}
}

func showUsage() {
	fmt.Printf(`%s - Framework LED Matrix System Statistics Display

USAGE:
    %s [OPTIONS] <COMMAND>

COMMANDS:
    run                 Run the daemon in foreground mode
    install             Install the daemon as a system service
    remove, uninstall   Remove the daemon service
    start               Start the installed daemon service
    stop                Stop the running daemon service
    status              Show the daemon service status
    config              Show current configuration
    test                Test connection to LED matrix

OPTIONS:
    -config string      Path to configuration file
    -port string        Serial port for LED matrix
    -brightness int     LED brightness (0-255)
    -mode string        Display mode (percentage, gradient, activity, status)
    -metric string      Primary metric to display (cpu, memory, disk, network)
    -log-level string   Set log level (debug, info, warn, error)
    -version           Show version information
    -help              Show this help message

EXAMPLES:
    %s run                                    # Run in foreground
    %s -config /etc/daemon.yaml run         # Run with custom config
    %s -port /dev/ttyACM0 -brightness 128 run # Run with overrides
    %s install                               # Install as system service
    %s start                                 # Start system service
    %s test                                  # Test LED matrix connection

CONFIGURATION:
    The daemon looks for configuration files in the following order:
    1. Path specified by -config flag
    2. $XDG_CONFIG_HOME/framework-led-daemon/config.yaml
    3. $HOME/.config/framework-led-daemon/config.yaml
    4. /etc/framework-led-daemon/config.yaml
    5. ./configs/config.yaml

`, name, name, name, name, name, name, name, name)
}

func showConfiguration(cfg *config.Config) {
	fmt.Printf("Current Configuration:\n")
	fmt.Printf("  Matrix:\n")
	fmt.Printf("    Port: %s\n", cfg.Matrix.Port)
	fmt.Printf("    Baud Rate: %d\n", cfg.Matrix.BaudRate)
	fmt.Printf("    Auto Discover: %t\n", cfg.Matrix.AutoDiscover)
	fmt.Printf("    Brightness: %d\n", cfg.Matrix.Brightness)
	fmt.Printf("  Display:\n")
	fmt.Printf("    Mode: %s\n", cfg.Display.Mode)
	fmt.Printf("    Primary Metric: %s\n", cfg.Display.PrimaryMetric)
	fmt.Printf("    Update Rate: %s\n", cfg.Display.UpdateRate)
	fmt.Printf("    Show Activity: %t\n", cfg.Display.ShowActivity)
	fmt.Printf("  Stats:\n")
	fmt.Printf("    Collect Interval: %s\n", cfg.Stats.CollectInterval)
	fmt.Printf("    CPU Enabled: %t\n", cfg.Stats.EnableCPU)
	fmt.Printf("    Memory Enabled: %t\n", cfg.Stats.EnableMemory)
	fmt.Printf("    Disk Enabled: %t\n", cfg.Stats.EnableDisk)
	fmt.Printf("    Network Enabled: %t\n", cfg.Stats.EnableNetwork)
	fmt.Printf("  Thresholds:\n")
	fmt.Printf("    CPU Warning/Critical: %.1f%% / %.1f%%\n",
		cfg.Stats.Thresholds.CPUWarning, cfg.Stats.Thresholds.CPUCritical)
	fmt.Printf("    Memory Warning/Critical: %.1f%% / %.1f%%\n",
		cfg.Stats.Thresholds.MemoryWarning, cfg.Stats.Thresholds.MemoryCritical)
}

func testConnection(cfg *config.Config) error {
	log.Printf("Testing connection to LED matrix...")

	service, err := daemon.NewService(cfg)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := service.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	log.Printf("Connection test completed successfully")

	return nil
}
