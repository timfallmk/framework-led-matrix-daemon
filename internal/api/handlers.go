package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
)

// handleMetricsGet returns a one-shot snapshot of current system metrics.
func (s *Server) handleMetricsGet(req Request) Response {
	if s.collector == nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: "collector not available"},
		}
	}

	summary, err := s.collector.GetSummary()
	if err != nil || summary == nil {
		msg := "no metrics available yet"
		if err != nil {
			msg = err.Error()
		}

		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: msg},
		}
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
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	return Response{ID: req.ID, Result: data}
}

// handleConfigGet returns the current daemon configuration as JSON.
func (s *Server) handleConfigGet(req Request) Response {
	cfg := s.getConfig()
	if cfg == nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: "config not available"},
		}
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	return Response{ID: req.ID, Result: data}
}

// handleConfigUpdate applies a partial config update sent by the client.
func (s *Server) handleConfigUpdate(req Request) Response {
	if req.Params == nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInvalidParams, Message: "params required"},
		}
	}

	// Parse partial config update as a map
	var updates map[string]json.RawMessage
	if err := json.Unmarshal(req.Params, &updates); err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid params: %v", err)},
		}
	}

	cfg := s.getConfig()
	if cfg == nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: "config not available"},
		}
	}

	// Re-marshal the full config, apply updates on top, then unmarshal back
	fullData, err := json.Marshal(cfg)
	if err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	var merged map[string]json.RawMessage
	if err := json.Unmarshal(fullData, &merged); err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	for k, v := range updates {
		merged[k] = v
	}

	mergedData, err := json.Marshal(merged)
	if err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	var newCfg config.Config
	if err := json.Unmarshal(mergedData, &newCfg); err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid config: %v", err)},
		}
	}

	if err := newCfg.Validate(); err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("validation failed: %v", err)},
		}
	}

	s.configMu.Lock()
	s.config = &newCfg
	s.configMu.Unlock()

	result, err := json.Marshal(map[string]string{"status": "ok"})
	if err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	return Response{ID: req.ID, Result: result}
}

// handleDisplaySetMode changes the daemon's active display mode.
func (s *Server) handleDisplaySetMode(req Request) Response {
	var params SetModeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid params: %v", err)},
		}
	}

	if s.display == nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: "display controller not available"},
		}
	}

	if err := s.display.SetDisplayMode(params.Mode); err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	// Also update config
	cfg := s.getConfig()
	if cfg != nil {
		cfg.Display.Mode = params.Mode
	}

	result, err := json.Marshal(map[string]string{"status": "ok"})
	if err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	return Response{ID: req.ID, Result: result}
}

// handleDisplaySetBrightness updates the LED matrix brightness level.
func (s *Server) handleDisplaySetBrightness(req Request) Response {
	var params SetBrightnessParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid params: %v", err)},
		}
	}

	if params.Brightness < 0 || params.Brightness > 255 {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInvalidParams, Message: "brightness must be 0-255"},
		}
	}

	if s.display == nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: "display controller not available"},
		}
	}

	if err := s.display.SetBrightness(byte(params.Brightness)); err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	result, err := json.Marshal(map[string]string{"status": "ok"})
	if err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	return Response{ID: req.ID, Result: result}
}

// handleDisplaySetMetric sets the primary metric used for the display.
func (s *Server) handleDisplaySetMetric(req Request) Response {
	var params SetMetricParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid params: %v", err)},
		}
	}

	if s.display == nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: "display controller not available"},
		}
	}

	if err := s.display.SetPrimaryMetric(params.Metric); err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	// Also update config
	cfg := s.getConfig()
	if cfg != nil {
		cfg.Display.PrimaryMetric = params.Metric
	}

	result, err := json.Marshal(map[string]string{"status": "ok"})
	if err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	return Response{ID: req.ID, Result: result}
}

// handleHealthGet returns the results of all registered health checks.
func (s *Server) handleHealthGet(req Request) Response {
	if s.health == nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: "health monitor not available"},
		}
	}

	checks := s.health.GetHealth()
	results := make([]HealthCheckResult, 0, len(checks))

	for _, check := range checks {
		results = append(results, HealthCheckResult{
			Name:        check.Name,
			Status:      string(check.Status),
			Message:     check.Message,
			Error:       check.Error,
			LastChecked: check.LastChecked.Format(time.RFC3339),
			Duration:    check.Duration.String(),
		})
	}

	data, err := json.Marshal(results)
	if err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	return Response{ID: req.ID, Result: data}
}

// handleStatusGet returns current daemon status including uptime, display mode, metric, and brightness.
func (s *Server) handleStatusGet(req Request) Response {
	cfg := s.getConfig()

	result := StatusResult{
		Uptime:    time.Since(s.startTime).String(),
		Connected: s.display != nil,
	}

	if cfg != nil {
		result.DisplayMode = cfg.Display.Mode
		result.PrimaryMetric = cfg.Display.PrimaryMetric
		result.Brightness = int(cfg.Matrix.Brightness)

		if cfg.Matrix.DualMode != "" {
			result.MatrixMode = cfg.Matrix.DualMode
		} else {
			result.MatrixMode = "single"
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	return Response{ID: req.ID, Result: data}
}

// handleMatrixGetState returns the raw display state from the active DisplayController.
func (s *Server) handleMatrixGetState(req Request) Response {
	if s.display == nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: "display controller not available"},
		}
	}

	state := s.display.GetDisplayState()

	data, err := json.Marshal(state)
	if err != nil {
		return Response{
			ID:    req.ID,
			Error: &ErrorInfo{Code: ErrCodeInternal, Message: err.Error()},
		}
	}

	return Response{ID: req.ID, Result: data}
}
