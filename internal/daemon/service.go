package daemon

import (
	"context"
	"fmt"
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

type Service struct {
	daemon.Daemon
	config *config.Config
	// Legacy single matrix support
	matrix  *matrix.Client
	display *matrix.DisplayManager
	// Multi-matrix support
	multiClient     *matrix.MultiClient
	multiDisplay    *matrix.MultiDisplayManager
	collector       *stats.Collector
	visualizer      *visualizer.Visualizer
	multiVisualizer *visualizer.MultiVisualizer
	// Observability
	logger          *logging.Logger
	eventLogger     *logging.EventLogger
	metricsCollector *observability.MetricsCollector
	appMetrics      *observability.ApplicationMetrics
	healthMonitor   *observability.HealthMonitor
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	stopCh          chan struct{}
	usingMultiple   bool
	startTime       time.Time
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
		startTime:        time.Now(),
	}

	return service, nil
}

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

	s.matrix = matrix.NewClient()
	if err := s.matrix.Connect(s.config.Matrix.Port); err != nil {
		s.eventLogger.LogMatrix(logging.LevelError, "failed to connect to LED matrix", "single", map[string]interface{}{
			"port": s.config.Matrix.Port,
		})
		return fmt.Errorf("failed to connect to LED matrix: %w", err)
	}

	s.eventLogger.LogMatrix(logging.LevelInfo, "connected to LED matrix", "single", map[string]interface{}{
		"port": s.config.Matrix.Port,
	})

	s.display = matrix.NewDisplayManager(s.matrix)
	s.display.SetUpdateRate(s.config.Display.UpdateRate)

	if err := s.display.SetBrightness(s.config.Matrix.Brightness); err != nil {
		s.eventLogger.LogMatrix(logging.LevelWarn, "failed to set brightness", "single", map[string]interface{}{
			"brightness": s.config.Matrix.Brightness,
			"error": err.Error(),
		})
	}

	s.usingMultiple = false

	s.collector = stats.NewCollector(s.config.Stats.CollectInterval)
	s.collector.SetThresholds(stats.Thresholds{
		CPUWarning:     s.config.Stats.Thresholds.CPUWarning,
		CPUCritical:    s.config.Stats.Thresholds.CPUCritical,
		MemoryWarning:  s.config.Stats.Thresholds.MemoryWarning,
		MemoryCritical: s.config.Stats.Thresholds.MemoryCritical,
		DiskWarning:    s.config.Stats.Thresholds.DiskWarning,
		DiskCritical:   s.config.Stats.Thresholds.DiskCritical,
	})

	s.visualizer = visualizer.NewVisualizer(s.display, s.config)

	s.eventLogger.LogDaemon(logging.LevelInfo, "single matrix daemon initialized successfully", "initialize_single_complete", nil)
	return nil
}

func (s *Service) registerHealthChecks() {
	// Matrix health check
	matrixChecker := observability.NewMatrixHealthChecker("matrix", func(ctx context.Context) error {
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
	
	s.eventLogger.LogDaemon(logging.LevelInfo, "initializing multi-matrix mode", "initialize_multi", map[string]interface{}{
		"matrix_count": len(s.config.Matrix.Matrices),
	})

	// Convert config matrices to proper type
	matrices := s.convertConfigMatrices(s.config.ConvertMatrices())

	s.multiClient = matrix.NewMultiClient()
	if err := s.multiClient.DiscoverAndConnect(matrices, s.config.Matrix.BaudRate); err != nil {
		// Fallback to single matrix mode if multi-matrix setup fails
		s.eventLogger.LogMatrix(logging.LevelWarn, "multi-matrix initialization failed, falling back to single matrix", "multi", map[string]interface{}{
			"error": err.Error(),
		})
		return s.initializeSingleMatrix()
	}

	s.multiDisplay = matrix.NewMultiDisplayManager(s.multiClient, s.config.Matrix.DualMode)
	s.multiDisplay.SetUpdateRate(s.config.Display.UpdateRate)

	// Set brightness for all matrices (will use individual brightness settings from config)
	if err := s.multiDisplay.SetBrightness(s.config.Matrix.Brightness); err != nil {
		s.eventLogger.LogMatrix(logging.LevelWarn, "failed to set brightness on some matrices", "multi", map[string]interface{}{
			"brightness": s.config.Matrix.Brightness,
			"error": err.Error(),
		})
	}

	s.usingMultiple = true

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
	s.multiVisualizer = visualizer.NewMultiVisualizer(s.multiDisplay, s.config)

	s.eventLogger.LogDaemon(logging.LevelInfo, "multi-matrix daemon initialized successfully", "initialize_multi_complete", map[string]interface{}{
		"matrix_count": len(s.multiClient.GetClients()),
	})
	return nil
}

// convertConfigMatrices converts config.SingleMatrixConfig to matrix.SingleMatrixConfig
func (s *Service) convertConfigMatrices(configMatrices []config.SingleMatrixConfig) []matrix.SingleMatrixConfig {
	var matrices []matrix.SingleMatrixConfig

	for _, cm := range configMatrices {
		matrix := matrix.SingleMatrixConfig{
			Name:       cm.Name,
			Port:       cm.Port,
			Role:       cm.Role,
			Brightness: cm.Brightness,
			Metrics:    cm.Metrics,
		}
		matrices = append(matrices, matrix)
	}

	return matrices
}

func (s *Service) Start() error {
	s.eventLogger.LogDaemon(logging.LevelInfo, "starting Framework LED Matrix daemon", "start", nil)

	if err := s.Initialize(); err != nil {
		s.eventLogger.LogDaemon(logging.LevelError, "failed to initialize service", "start_error", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	s.wg.Add(1)
	go s.runStatsCollector()

	s.wg.Add(1)
	go s.runDisplayUpdater()

	s.wg.Add(1)
	go s.runRuntimeMetrics()

	s.wg.Add(1)
	go s.handleSignals()

	s.eventLogger.LogDaemon(logging.LevelInfo, "daemon started successfully", "start_complete", nil)
	return nil
}

func (s *Service) Stop() error {
	s.eventLogger.LogDaemon(logging.LevelInfo, "stopping Framework LED Matrix daemon", "stop", nil)

	close(s.stopCh)
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
			s.eventLogger.LogMatrix(logging.LevelWarn, "failed to disconnect from multi-matrices", "multi", map[string]interface{}{
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
	s.logger.Close()
	
	return nil
}

func (s *Service) Run() error {
	if err := s.Start(); err != nil {
		return err
	}

	<-s.stopCh
	return s.Stop()
}

func (s *Service) runStatsCollector() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.Stats.CollectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			timer := s.metricsCollector.StartTimer("stats_collection_duration", nil)
			stats, err := s.collector.CollectSystemStats()
			duration := timer.StopWithSuccess(err == nil)
			
			if err != nil {
				s.eventLogger.LogStats(logging.LevelWarn, "failed to collect system stats", "system", 0, map[string]interface{}{
					"error": err.Error(),
					"duration": duration.String(),
				})
			} else if stats != nil {
				// Record individual stat metrics
				s.appMetrics.RecordStatsCollection("cpu", stats.CPU.UsagePercent, duration)
				s.appMetrics.RecordStatsCollection("memory", stats.Memory.UsedPercent, duration)
				s.appMetrics.RecordStatsCollection("disk", float64(stats.Disk.ActivityRate), duration)
				s.appMetrics.RecordStatsCollection("network", float64(stats.Network.ActivityRate), duration)
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

func (s *Service) runDisplayUpdater() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.Display.UpdateRate)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			timer := s.metricsCollector.StartTimer("display_update_duration", nil)
			
			summary, err := s.collector.GetSummary()
			if err != nil {
				timer.StopWithSuccess(false)
				s.eventLogger.LogStats(logging.LevelWarn, "failed to get stats summary", "display_update", 0, map[string]interface{}{
					"error": err.Error(),
				})
				continue
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
			
			duration := timer.StopWithSuccess(updateErr == nil)
			s.appMetrics.RecordDisplayUpdate(mode, updateErr == nil, duration)
			
			if updateErr != nil {
				s.eventLogger.LogMatrix(logging.LevelWarn, "failed to update display", mode, map[string]interface{}{
					"error": updateErr.Error(),
					"duration": duration.String(),
				})
			}
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
				close(s.stopCh)
				return
			case syscall.SIGHUP:
				s.eventLogger.LogDaemon(logging.LevelInfo, "received SIGHUP, reloading configuration", "signal", map[string]interface{}{
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
		timer.StopWithSuccess(false)
		s.appMetrics.RecordConfigReload(false, timer.StopWithSuccess(false))
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

	s.visualizer.UpdateConfig(newConfig)

	duration := timer.StopWithSuccess(true)
	s.appMetrics.RecordConfigReload(true, duration)
	
	s.eventLogger.LogConfig(logging.LevelInfo, "configuration reloaded successfully", "", map[string]interface{}{
		"duration": duration.String(),
	})
	return nil
}

func (s *Service) Install() (string, error) {
	return s.Daemon.Install()
}

func (s *Service) Remove() (string, error) {
	return s.Daemon.Remove()
}

func (s *Service) Status() (string, error) {
	return s.Daemon.Status()
}

func (s *Service) StartService() (string, error) {
	return s.Daemon.Start()
}

func (s *Service) StopService() (string, error) {
	return s.Daemon.Stop()
}
