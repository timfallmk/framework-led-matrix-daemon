// Package daemon provides the core service management and orchestration for the Framework LED Matrix daemon.
// It coordinates system statistics collection, LED matrix display updates, and service lifecycle management.
package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/takama/daemon"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/logging"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/matrix"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/observability"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/stats"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/visualizer"
)

// Service represents the main daemon service that orchestrates LED matrix display operations.
// It manages system statistics collection, display updates, and service lifecycle.
type Service struct {
	startTime time.Time
	daemon.Daemon
	ctx              context.Context
	eventLogger      *logging.EventLogger
	appMetrics       *observability.ApplicationMetrics
	multiDisplay     *matrix.MultiDisplayManager
	collector        *stats.Collector
	visualizer       *visualizer.Visualizer
	multiVisualizer  *visualizer.MultiVisualizer
	logger           *logging.Logger
	display          *matrix.DisplayManager
	metricsCollector *observability.MetricsCollector
	multiClient      *matrix.MultiClient
	healthMonitor    *observability.HealthMonitor
	matrix           *matrix.Client
	cancel           context.CancelFunc
	config           *config.Config
	stopCh           chan struct{}
	wg               sync.WaitGroup
	stopOnce         sync.Once
	mu               sync.RWMutex // Protects matrix, multiClient, display, multiDisplay
	usingMultiple    bool
}

// NewService creates and returns a configured Service instance.
//
// It builds the underlying system daemon, initializes structured logging (and sets it
// as the global logger), and constructs observability components (metrics collector,
// application metrics, and health monitor). The returned Service is populated with a
// cancellable background context, an event logger, and channels used to control
// lifecycle. The provided cfg must contain the daemon, logging, and observability
// settings to initialize these components.
//
// Returns an error if creating the system daemon or the structured logger fails.
func NewService(cfg *config.Config) (*Service, error) {
	d, err := daemon.New(cfg.Daemon.Name, cfg.Daemon.Description, daemon.SystemDaemon, "run")
	if err != nil {
		return nil, fmt.Errorf("failed to create daemon: %w", err)
	}

	// Initialize structured logging
	logConfig := logging.Config{
		Level:     logging.LogLevel(cfg.Logging.Level),
		Format:    logging.LogFormat(cfg.Logging.Format),
		Output:    cfg.Logging.Output,
		AddSource: cfg.Logging.AddSource,
	}

	logger, err := logging.NewLogger(logConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Set as global logger
	logging.SetGlobalLogger(logger)

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize observability components
	metricsCollector := observability.NewMetricsCollector(logger, 30*time.Second)
	appMetrics := observability.NewApplicationMetrics(metricsCollector)
	healthMonitor := observability.NewHealthMonitor(logger, appMetrics, 15*time.Second)

	service := &Service{
		Daemon:           d,
		config:           cfg,
		logger:           logger,
		eventLogger:      logging.NewEventLogger(logger),
		metricsCollector: metricsCollector,
		appMetrics:       appMetrics,
		healthMonitor:    healthMonitor,
		ctx:              ctx,
		cancel:           cancel,
		stopCh:           make(chan struct{}),
	}

	return service, nil
}

// Initialize sets up the service components including LED matrix connections,
// system statistics collection, and display management.
func (s *Service) Initialize() error {
	s.eventLogger.LogDaemon(logging.LevelInfo, "initializing Framework LED Matrix daemon", "initialize", nil)

	// Register health checks
	s.registerHealthChecks()

	// Start health monitoring
	s.healthMonitor.Start()

	// Determine if we should use multi-matrix mode
	if len(s.config.Matrix.Matrices) > 0 && s.config.Matrix.DualMode != "" {
		return s.initializeMultiMatrix()
	} else {
		return s.initializeSingleMatrix()
	}
}

func (s *Service) initializeSingleMatrix() error {
	timer := s.metricsCollector.StartTimer("matrix_initialization_duration", map[string]string{"mode": "single"})
	defer timer.Stop()

	s.eventLogger.LogDaemon(logging.LevelInfo, "initializing single matrix mode", "initialize_single", nil)

	client := matrix.NewClient()
	if err := client.Connect(s.config.Matrix.Port); err != nil {
		s.eventLogger.LogMatrix(logging.LevelError, "failed to connect to LED matrix", "single", map[string]interface{}{
			"port": s.config.Matrix.Port,
		})

		return fmt.Errorf("failed to connect to LED matrix: %w", err)
	}

	s.eventLogger.LogMatrix(logging.LevelInfo, "connected to LED matrix", "single", map[string]interface{}{
		"port": s.config.Matrix.Port,
	})

	display := matrix.NewDisplayManager(client)
	display.SetUpdateRate(s.config.Display.UpdateRate)

	if err := display.SetBrightness(s.config.Matrix.Brightness); err != nil {
		s.eventLogger.LogMatrix(logging.LevelWarn, "failed to set brightness", "single", map[string]interface{}{
			"brightness": s.config.Matrix.Brightness,
			"error":      err.Error(),
		})
	}

	// Safely assign to shared fields protected by mutex
	s.mu.Lock()
	s.matrix = client
	s.display = display
	s.usingMultiple = false
	s.mu.Unlock()

	s.collector = stats.NewCollector(s.config.Stats.CollectInterval)
	s.collector.SetThresholds(stats.Thresholds{
		CPUWarning:     s.config.Stats.Thresholds.CPUWarning,
		CPUCritical:    s.config.Stats.Thresholds.CPUCritical,
		MemoryWarning:  s.config.Stats.Thresholds.MemoryWarning,
		MemoryCritical: s.config.Stats.Thresholds.MemoryCritical,
		DiskWarning:    s.config.Stats.Thresholds.DiskWarning,
		DiskCritical:   s.config.Stats.Thresholds.DiskCritical,
	})

	s.visualizer = visualizer.NewVisualizer(display, s.config)

	s.eventLogger.LogDaemon(logging.LevelInfo, "single matrix daemon initialized successfully",
		"initialize_single_complete", nil)

	return nil
}

func (s *Service) registerHealthChecks() {
	// Matrix health check
	matrixChecker := observability.NewMatrixHealthChecker("matrix", func(ctx context.Context) error {
		s.mu.RLock()
		defer s.mu.RUnlock()
		
		if s.usingMultiple && s.multiClient != nil {
			return nil // Multi-client health check would be implemented
		} else if s.matrix != nil {
			return nil // Single matrix health check would be implemented
		}

		return fmt.Errorf("no matrix client initialized")
	})
	s.healthMonitor.RegisterChecker(matrixChecker)

	// Stats collection health check
	statsChecker := observability.NewStatsHealthChecker("stats", func(ctx context.Context) error {
		if s.collector == nil {
			return fmt.Errorf("stats collector not initialized")
		}
		// Try to collect stats to verify it's working
		_, err := s.collector.CollectSystemStats()

		return err
	})
	s.healthMonitor.RegisterChecker(statsChecker)

	// Config health check
	configChecker := observability.NewConfigHealthChecker("config", func(ctx context.Context) error {
		if s.config == nil {
			return fmt.Errorf("configuration not loaded")
		}

		return s.config.Validate()
	})
	s.healthMonitor.RegisterChecker(configChecker)

	// Memory health check (warn if using more than 100MB)
	memoryChecker := observability.NewMemoryHealthChecker("memory", 100*1024*1024)
	s.healthMonitor.RegisterChecker(memoryChecker)
}

func (s *Service) initializeMultiMatrix() error {
	timer := s.metricsCollector.StartTimer("matrix_initialization_duration", map[string]string{"mode": "multi"})
	defer timer.Stop()

	s.eventLogger.LogDaemon(logging.LevelInfo, "initializing multi-matrix mode",
		"initialize_multi", map[string]interface{}{
			"matrix_count": len(s.config.Matrix.Matrices),
		})

	// Convert config matrices to proper type
	matrices := s.convertConfigMatrices(s.config.ConvertMatrices())

	multiClient := matrix.NewMultiClient()
	if err := multiClient.DiscoverAndConnect(matrices, s.config.Matrix.BaudRate); err != nil {
		// Fallback to single matrix mode if multi-matrix setup fails
		s.eventLogger.LogMatrix(logging.LevelWarn,
			"multi-matrix initialization failed, falling back to single matrix", "multi", map[string]interface{}{
				"error": err.Error(),
			})

		return s.initializeSingleMatrix()
	}

	multiDisplay := matrix.NewMultiDisplayManager(multiClient, s.config.Matrix.DualMode)
	multiDisplay.SetUpdateRate(s.config.Display.UpdateRate)

	// Set brightness for all matrices (will use individual brightness settings from config)
	if err := multiDisplay.SetBrightness(s.config.Matrix.Brightness); err != nil {
		s.eventLogger.LogMatrix(logging.LevelWarn, "failed to set brightness on some matrices",
			"multi", map[string]interface{}{
				"brightness": s.config.Matrix.Brightness,
				"error":      err.Error(),
			})
	}

	// Safely assign to shared fields protected by mutex
	s.mu.Lock()
	s.multiClient = multiClient
	s.multiDisplay = multiDisplay
	s.usingMultiple = true
	s.mu.Unlock()

	s.collector = stats.NewCollector(s.config.Stats.CollectInterval)
	s.collector.SetThresholds(stats.Thresholds{
		CPUWarning:     s.config.Stats.Thresholds.CPUWarning,
		CPUCritical:    s.config.Stats.Thresholds.CPUCritical,
		MemoryWarning:  s.config.Stats.Thresholds.MemoryWarning,
		MemoryCritical: s.config.Stats.Thresholds.MemoryCritical,
		DiskWarning:    s.config.Stats.Thresholds.DiskWarning,
		DiskCritical:   s.config.Stats.Thresholds.DiskCritical,
	})

	// Create multi-visualizer for dual matrix mode
	s.multiVisualizer = visualizer.NewMultiVisualizer(multiDisplay, s.config)

	s.eventLogger.LogDaemon(logging.LevelInfo, "multi-matrix daemon initialized successfully",
		"initialize_multi_complete", map[string]interface{}{
			"matrix_count": len(multiClient.GetClients()),
		})

	return nil
}

// convertConfigMatrices converts config.SingleMatrixConfig to matrix.SingleMatrixConfig.
func (s *Service) convertConfigMatrices(configMatrices []config.SingleMatrixConfig) []matrix.SingleMatrixConfig {
	matrices := make([]matrix.SingleMatrixConfig, 0, len(configMatrices))

	for _, cm := range configMatrices {
		matrixConfig := matrix.SingleMatrixConfig{
			Name:       cm.Name,
			Port:       cm.Port,
			Role:       cm.Role,
			Brightness: cm.Brightness,
			Metrics:    cm.Metrics,
		}
		matrices = append(matrices, matrixConfig)
	}

	return matrices
}

// Start begins the daemon service operation, starting statistics collection and display updates.
func (s *Service) Start() error {
	s.eventLogger.LogDaemon(logging.LevelInfo, "starting Framework LED Matrix daemon", "start", nil)

	if err := s.Initialize(); err != nil {
		s.eventLogger.LogDaemon(logging.LevelError, "failed to initialize service", "start_error", map[string]interface{}{
			"error": err.Error(),
		})

		return fmt.Errorf("failed to initialize service: %w", err)
	}

	s.wg.Add(1)

	go s.runSystemLoop()

	s.wg.Add(1)

	go s.runRuntimeMetrics()

	s.wg.Add(1)

	go s.handleSignals()

	// Initialize start time only after successful startup
	s.startTime = time.Now()

	s.eventLogger.LogDaemon(logging.LevelInfo, "daemon started successfully", "start_complete", nil)

	return nil
}

// Stop gracefully shuts down the service by canceling contexts, waiting for goroutines,
// clearing displays, and releasing resources.
func (s *Service) Stop() error {
	s.eventLogger.LogDaemon(logging.LevelInfo, "stopping Framework LED Matrix daemon", "stop", nil)

	s.stopOnce.Do(func() { close(s.stopCh) })
	s.cancel()

	s.wg.Wait()

	// Stop observability components
	s.healthMonitor.Stop()
	s.metricsCollector.Close()

	// Clear displays before disconnecting
	if s.usingMultiple && s.multiDisplay != nil {
		if err := s.multiDisplay.UpdateStatus("off"); err != nil {
			s.eventLogger.LogMatrix(logging.LevelWarn, "failed to clear multi-displays", "multi", map[string]interface{}{
				"error": err.Error(),
			})
		}
	} else if s.display != nil {
		if err := s.display.ShowStatus("off"); err != nil {
			s.eventLogger.LogMatrix(logging.LevelWarn, "failed to clear display", "single", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Disconnect from matrices
	if s.usingMultiple && s.multiClient != nil {
		if err := s.multiClient.Disconnect(); err != nil {
			s.eventLogger.LogMatrix(logging.LevelWarn, "failed to disconnect from multi-matrices",
				"multi", map[string]interface{}{
					"error": err.Error(),
				})
		}
	} else if s.matrix != nil {
		if err := s.matrix.Disconnect(); err != nil {
			s.eventLogger.LogMatrix(logging.LevelWarn, "failed to disconnect from matrix", "single", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Record final uptime metric
	uptime := time.Since(s.startTime)
	s.appMetrics.RecordDaemonUptime(uptime)

	s.eventLogger.LogDaemon(logging.LevelInfo, "daemon stopped successfully", "stop_complete", map[string]interface{}{
		"uptime": uptime.String(),
	})

	// Close logging resources
	s.eventLogger.Close()

	if err := s.logger.Close(); err != nil {
		log.Printf("Warning: failed to close logger: %v", err)
	}

	return nil
}

// Run starts the service and blocks until it receives a stop signal, then shuts down gracefully.
func (s *Service) Run() error {
	if err := s.Start(); err != nil {
		return err
	}

	<-s.stopCh

	return s.Stop()
}

func (s *Service) runSystemLoop() {
	defer s.wg.Done()

	// Use the faster of the two configured intervals for the combined loop
	interval := s.config.Display.UpdateRate
	if s.config.Stats.CollectInterval < interval {
		interval = s.config.Stats.CollectInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			// Collect stats first
			statsTimer := s.metricsCollector.StartTimer("stats_collection_duration", nil)
			collectedStats, err := s.collector.CollectSystemStats()

			statsDuration := statsTimer.StopWithSuccess(err == nil)
			if err != nil {
				s.eventLogger.LogStats(logging.LevelWarn, "failed to collect system stats", "system", 0, map[string]interface{}{
					"error":    err.Error(),
					"duration": statsDuration.String(),
				})

				continue
			}

			if collectedStats != nil {
				// Record individual stat metrics
				s.appMetrics.RecordStatsCollection("cpu", collectedStats.CPU.UsagePercent, statsDuration)
				s.appMetrics.RecordStatsCollection("memory", collectedStats.Memory.UsedPercent, statsDuration)
				s.appMetrics.RecordStatsCollection("disk", float64(collectedStats.Disk.ActivityRate), statsDuration)
				s.appMetrics.RecordStatsCollection("network", float64(collectedStats.Network.ActivityRate), statsDuration)

				// Immediately update display with fresh stats
				displayTimer := s.metricsCollector.StartTimer("display_update_duration", nil)

				// Create summary directly from collected stats to avoid double collection
				summary := &stats.StatsSummary{
					CPUUsage:        collectedStats.CPU.UsagePercent,
					MemoryUsage:     collectedStats.Memory.UsedPercent,
					DiskActivity:    collectedStats.Disk.ActivityRate,
					NetworkActivity: collectedStats.Network.ActivityRate,
					Timestamp:       collectedStats.Timestamp,
				}

				// Determine status based on thresholds
				thresholds := s.collector.GetThresholds()
				switch {
				case summary.CPUUsage >= thresholds.CPUCritical || summary.MemoryUsage >= thresholds.MemoryCritical:
					summary.Status = stats.StatusCritical
				case summary.CPUUsage >= thresholds.CPUWarning || summary.MemoryUsage >= thresholds.MemoryWarning:
					summary.Status = stats.StatusWarning
				default:
					summary.Status = stats.StatusNormal
				}

				// Use appropriate visualizer based on mode
				var updateErr error

				mode := "single"
				if s.usingMultiple && s.multiVisualizer != nil {
					mode = "multi"
					updateErr = s.multiVisualizer.UpdateDisplay(summary)
				} else if s.visualizer != nil {
					updateErr = s.visualizer.UpdateDisplay(summary)
				}

				displayDuration := displayTimer.StopWithSuccess(updateErr == nil)
				s.appMetrics.RecordDisplayUpdate(mode, updateErr == nil, displayDuration)

				if updateErr != nil {
					s.eventLogger.LogMatrix(logging.LevelWarn, "failed to update display", mode, map[string]interface{}{
						"error":    updateErr.Error(),
						"duration": displayDuration.String(),
					})
				}
			}
		}
	}
}

func (s *Service) runRuntimeMetrics() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			// Record memory metrics
			s.appMetrics.RecordMemoryUsage(memStats.HeapAlloc, memStats.HeapSys, memStats.HeapInuse)

			// Record goroutine count
			s.appMetrics.RecordGoroutines(runtime.NumGoroutine())

			// Record uptime
			uptime := time.Since(s.startTime)
			s.appMetrics.RecordDaemonUptime(uptime)
		}
	}
}

func (s *Service) handleSignals() {
	defer s.wg.Done()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		select {
		case <-s.ctx.Done():
			return
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				s.eventLogger.LogDaemon(logging.LevelInfo, "received shutdown signal", "signal", map[string]interface{}{
					"signal": sig.String(),
				})
				s.stopOnce.Do(func() { close(s.stopCh) })

				return
			case syscall.SIGHUP:
				s.eventLogger.LogDaemon(logging.LevelInfo, "received SIGHUP, reloading configuration",
					"signal", map[string]interface{}{
						"signal": sig.String(),
					})

				if err := s.reloadConfig(); err != nil {
					s.eventLogger.LogConfig(logging.LevelWarn, "failed to reload config", "", map[string]interface{}{
						"error": err.Error(),
					})
				}
			}
		}
	}
}

func (s *Service) reloadConfig() error {
	timer := s.metricsCollector.StartTimer("config_reload_duration", nil)

	newConfig, err := config.LoadConfig("")
	if err != nil {
		duration := timer.StopWithSuccess(false)
		s.appMetrics.RecordConfigReload(false, duration)

		return fmt.Errorf("failed to load config: %w", err)
	}

	s.config = newConfig

	s.collector.SetThresholds(stats.Thresholds{
		CPUWarning:     newConfig.Stats.Thresholds.CPUWarning,
		CPUCritical:    newConfig.Stats.Thresholds.CPUCritical,
		MemoryWarning:  newConfig.Stats.Thresholds.MemoryWarning,
		MemoryCritical: newConfig.Stats.Thresholds.MemoryCritical,
		DiskWarning:    newConfig.Stats.Thresholds.DiskWarning,
		DiskCritical:   newConfig.Stats.Thresholds.DiskCritical,
	})

	if s.visualizer != nil {
		s.visualizer.UpdateConfig(newConfig)
	}

	duration := timer.StopWithSuccess(true)
	s.appMetrics.RecordConfigReload(true, duration)

	s.eventLogger.LogConfig(logging.LevelInfo, "configuration reloaded successfully", "", map[string]interface{}{
		"duration": duration.String(),
	})

	return nil
}

// Install installs the service as a system daemon.
func (s *Service) Install() (string, error) {
	return s.Daemon.Install()
}

// Remove removes the service from the system.
func (s *Service) Remove() (string, error) {
	return s.Daemon.Remove()
}

// Status returns the current status of the system service.
func (s *Service) Status() (string, error) {
	return s.Daemon.Status()
}

// StartService starts the system service.
func (s *Service) StartService() (string, error) {
	return s.Daemon.Start()
}

// StopService stops the system service.
func (s *Service) StopService() (string, error) {
	return s.Daemon.Stop()
}
