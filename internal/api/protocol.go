// Package api provides a Unix domain socket-based API for communication between
// the GUI application and the Framework LED Matrix daemon.
package api

import "encoding/json"

// API method constants.
const (
	MethodMetricsGet       = "metrics.get"
	MethodMetricsSubscribe = "metrics.subscribe"
	MethodConfigGet        = "config.get"
	MethodConfigUpdate     = "config.update"
	MethodDisplaySetMode   = "display.set_mode"
	MethodDisplaySetBright = "display.set_brightness"
	MethodDisplaySetMetric = "display.set_metric"
	MethodHealthGet        = "health.get"
	MethodStatusGet        = "status.get"
	MethodMatrixGetState   = "matrix.get_state"
)

// Request represents a JSON-RPC-style request from a client.
type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     string          `json:"id"`
}

// Response represents a JSON-RPC-style response to a client.
type Response struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ErrorInfo      `json:"error,omitempty"`
	ID     string          `json:"id"`
}

// ErrorInfo contains error details in a response.
type ErrorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error code constants.
const (
	ErrCodeInvalidMethod = -32601
	ErrCodeInvalidParams = -32602
	ErrCodeInternal      = -32603
)

// MetricsResult contains a snapshot of system metrics.
type MetricsResult struct {
	CPUUsage        float64 `json:"cpu_usage"`
	MemoryUsage     float64 `json:"memory_usage"`
	DiskActivity    float64 `json:"disk_activity"`
	NetworkActivity float64 `json:"network_activity"`
	Status          string  `json:"status"`
	Timestamp       string  `json:"timestamp"`
}

// StatusResult contains daemon status information.
type StatusResult struct {
	Uptime        string `json:"uptime"`
	DisplayMode   string `json:"display_mode"`
	PrimaryMetric string `json:"primary_metric"`
	Brightness    int    `json:"brightness"`
	MatrixMode    string `json:"matrix_mode"`
	Connected     bool   `json:"connected"`
}

// HealthCheckResult represents a single health check entry.
type HealthCheckResult struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
	LastChecked string `json:"last_checked"`
	Duration    string `json:"duration"`
}

// SetModeParams contains parameters for display.set_mode.
type SetModeParams struct {
	Mode string `json:"mode"`
}

// SetBrightnessParams contains parameters for display.set_brightness.
type SetBrightnessParams struct {
	Brightness int `json:"brightness"`
}

// SetMetricParams contains parameters for display.set_metric.
type SetMetricParams struct {
	Metric string `json:"metric"`
}

// SubscribeParams contains parameters for metrics.subscribe.
type SubscribeParams struct {
	IntervalMs int `json:"interval_ms,omitempty"`
}
