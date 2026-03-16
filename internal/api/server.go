package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/observability"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/stats"
)

// DefaultSocketPath is the default Unix domain socket path for the API server.
const DefaultSocketPath = "/tmp/framework-led-daemon.sock"

// DisplayController provides methods the API server uses to control the display.
type DisplayController interface {
	SetDisplayMode(mode string) error
	SetBrightness(level byte) error
	SetPrimaryMetric(metric string) error
	GetDisplayState() map[string]interface{}
	IsMultiMatrix() bool
}

// ServerConfig holds the configuration for the API server.
type ServerConfig struct {
	SocketPath string
	Collector  *stats.Collector
	Config     *config.Config
	Health     *observability.HealthMonitor
	Display    DisplayController
}

// Server is a Unix domain socket API server that exposes daemon state and controls.
type Server struct {
	config     *config.Config
	collector  *stats.Collector
	health     *observability.HealthMonitor
	display    DisplayController
	listener   net.Listener
	socketPath string
	startTime  time.Time
	mu         sync.RWMutex
	configMu   sync.RWMutex
}

// NewServer creates a new API server with the given configuration.
func NewServer(cfg ServerConfig) *Server {
	socketPath := cfg.SocketPath
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}

	return &Server{
		socketPath: socketPath,
		collector:  cfg.Collector,
		config:     cfg.Config,
		health:     cfg.Health,
		display:    cfg.Display,
		startTime:  time.Now(),
	}
}

// Serve starts the API server and listens for connections until the context is cancelled.
func (s *Server) Serve(ctx context.Context) error {
	// Remove stale socket file only if it is actually a Unix socket
	if info, err := os.Lstat(s.socketPath); err == nil {
		if info.Mode()&os.ModeSocket != 0 {
			if removeErr := os.Remove(s.socketPath); removeErr != nil && !os.IsNotExist(removeErr) {
				return fmt.Errorf("failed to remove stale socket: %w", removeErr)
			}
		}
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.socketPath, err)
	}

	// Set permissions so non-root users can connect
	if err := os.Chmod(s.socketPath, 0o666); err != nil {
		_ = listener.Close()

		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	var wg sync.WaitGroup

	// Close listener when context is cancelled
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				wg.Wait()
				return nil
			default:
				continue
			}
		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			s.handleConnection(ctx, conn)
		}()
	}
}

// Close shuts down the API server and removes the socket file.
func (s *Server) Close() error {
	s.mu.Lock()
	listener := s.listener
	s.mu.Unlock()

	if listener != nil {
		_ = listener.Close()
	}

	err := os.Remove(s.socketPath)
	if os.IsNotExist(err) {
		return nil
	}

	return err
}

// UpdateConfig updates the config reference held by the server.
func (s *Server) UpdateConfig(cfg *config.Config) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.config = cfg
}

// getConfig returns the current config under a read lock.
func (s *Server) getConfig() *config.Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.config
}

// handleConnection reads JSON requests from conn until the connection is closed or ctx is cancelled.
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	// Allow up to 1MB messages
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := Response{
				Error: &ErrorInfo{Code: ErrCodeInvalidParams, Message: "invalid JSON"},
			}
			s.writeResponse(conn, resp)

			continue
		}

		resp := s.handleRequest(ctx, conn, req)
		s.writeResponse(conn, resp)
	}
}

// writeResponse marshals resp to JSON, appends a newline, and writes all bytes to conn.
// It loops until all bytes are written and discards the response silently on error.
func (s *Server) writeResponse(conn net.Conn, resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}

	data = append(data, '\n')
	conn.Write(data)
}

// handleRequest routes req to the appropriate handler and returns the response.
// For metrics.subscribe, the stream is handled in a blocking call and an empty ack is returned.
func (s *Server) handleRequest(ctx context.Context, conn net.Conn, req Request) Response {
	switch req.Method {
	case MethodMetricsGet:
		return s.handleMetricsGet(req)
	case MethodMetricsSubscribe:
		s.handleMetricsSubscribe(ctx, conn, req)
		// Subscribe doesn't return a single response; it streams
		return Response{ID: req.ID}
	case MethodConfigGet:
		return s.handleConfigGet(req)
	case MethodConfigUpdate:
		return s.handleConfigUpdate(req)
	case MethodDisplaySetMode:
		return s.handleDisplaySetMode(req)
	case MethodDisplaySetBright:
		return s.handleDisplaySetBrightness(req)
	case MethodDisplaySetMetric:
		return s.handleDisplaySetMetric(req)
	case MethodHealthGet:
		return s.handleHealthGet(req)
	case MethodStatusGet:
		return s.handleStatusGet(req)
	case MethodMatrixGetState:
		return s.handleMatrixGetState(req)
	default:
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInvalidMethod, Message: fmt.Sprintf("unknown method: %s", req.Method)},
		}
	}
}

// handleMetricsSubscribe streams periodic metrics snapshots to conn until ctx is cancelled or conn is closed.
func (s *Server) handleMetricsSubscribe(ctx context.Context, conn net.Conn, req Request) {
	var params SubscribeParams
	if req.Params != nil {
		json.Unmarshal(req.Params, &params)
	}

	interval := time.Duration(params.IntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = 2 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.collector == nil {
				continue
			}

			summary, err := s.collector.GetSummary()
			if err != nil || summary == nil {
				continue
			}

			result := MetricsResult{
				CPUUsage:        summary.CPUUsage,
				MemoryUsage:     summary.MemoryUsage,
				DiskActivity:    summary.DiskActivity,
				NetworkActivity: summary.NetworkActivity,
				Status:          summary.Status.String(),
				Timestamp:       summary.Timestamp.Format(time.RFC3339),
			}

			data, err := json.Marshal(result)
			if err != nil {
				continue
			}

			resp := Response{ID: req.ID, Result: data}

			respData, err := json.Marshal(resp)
			if err != nil {
				continue
			}

			respData = append(respData, '\n')
			if _, err := conn.Write(respData); err != nil {
				return // Client disconnected
			}
		}
	}
}
