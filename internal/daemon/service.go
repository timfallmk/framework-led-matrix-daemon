package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/takama/daemon"
	"github.com/timfa/framework-led-matrix-daemon/internal/config"
	"github.com/timfa/framework-led-matrix-daemon/internal/matrix"
	"github.com/timfa/framework-led-matrix-daemon/internal/stats"
	"github.com/timfa/framework-led-matrix-daemon/internal/visualizer"
)

type Service struct {
	daemon.Daemon
	config     *config.Config
	matrix     *matrix.Client
	display    *matrix.DisplayManager
	collector  *stats.Collector
	visualizer *visualizer.Visualizer
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	stopCh     chan struct{}
}

func NewService(cfg *config.Config) (*Service, error) {
	d, err := daemon.New(cfg.Daemon.Name, cfg.Daemon.Description, daemon.SystemDaemon, "run")
	if err != nil {
		return nil, fmt.Errorf("failed to create daemon: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		Daemon: d,
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
		stopCh: make(chan struct{}),
	}

	return service, nil
}

func (s *Service) Initialize() error {
	log.Printf("Initializing Framework LED Matrix daemon...")

	s.matrix = matrix.NewClient()
	if err := s.matrix.Connect(s.config.Matrix.Port); err != nil {
		return fmt.Errorf("failed to connect to LED matrix: %w", err)
	}

	s.display = matrix.NewDisplayManager(s.matrix)
	s.display.SetUpdateRate(s.config.Display.UpdateRate)

	if err := s.display.SetBrightness(s.config.Matrix.Brightness); err != nil {
		log.Printf("Warning: failed to set brightness: %v", err)
	}

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

	log.Printf("Daemon initialized successfully")
	return nil
}

func (s *Service) Start() error {
	log.Printf("Starting Framework LED Matrix daemon...")

	if err := s.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	s.wg.Add(1)
	go s.runStatsCollector()

	s.wg.Add(1)
	go s.runDisplayUpdater()

	s.wg.Add(1)
	go s.handleSignals()

	log.Printf("Daemon started successfully")
	return nil
}

func (s *Service) Stop() error {
	log.Printf("Stopping Framework LED Matrix daemon...")

	close(s.stopCh)
	s.cancel()

	s.wg.Wait()

	if s.matrix != nil {
		if err := s.matrix.Disconnect(); err != nil {
			log.Printf("Warning: failed to disconnect from matrix: %v", err)
		}
	}

	log.Printf("Daemon stopped successfully")
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
			if _, err := s.collector.CollectSystemStats(); err != nil {
				log.Printf("Warning: failed to collect system stats: %v", err)
			}
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
			summary, err := s.collector.GetSummary()
			if err != nil {
				log.Printf("Warning: failed to get stats summary: %v", err)
				continue
			}

			if err := s.visualizer.UpdateDisplay(summary); err != nil {
				log.Printf("Warning: failed to update display: %v", err)
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
				log.Printf("Received signal %v, shutting down...", sig)
				close(s.stopCh)
				return
			case syscall.SIGHUP:
				log.Printf("Received SIGHUP, reloading configuration...")
				if err := s.reloadConfig(); err != nil {
					log.Printf("Warning: failed to reload config: %v", err)
				}
			}
		}
	}
}

func (s *Service) reloadConfig() error {
	newConfig, err := config.LoadConfig("")
	if err != nil {
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

	log.Printf("Configuration reloaded successfully")
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
