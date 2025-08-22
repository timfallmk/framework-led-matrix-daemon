package observability

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/logging"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

// Metric represents a single metric data point
type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Unit      string            `json:"unit,omitempty"`
}

// MetricsCollector collects and manages application metrics
type MetricsCollector struct {
	logger        *logging.MetricsLogger
	eventLogger   *logging.EventLogger
	metrics       map[string]*Metric
	mu            sync.RWMutex
	flushInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger *logging.Logger, flushInterval time.Duration) *MetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())

	mc := &MetricsCollector{
		logger:        logging.NewMetricsLogger(logger),
		eventLogger:   logging.NewEventLogger(logger),
		metrics:       make(map[string]*Metric),
		flushInterval: flushInterval,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start metrics flushing goroutine
	mc.wg.Add(1)
	go mc.flushLoop()

	return mc
}

// IncCounter increments a counter metric
func (mc *MetricsCollector) IncCounter(name string, labels map[string]string) {
	mc.AddCounter(name, 1, labels)
}

// AddCounter adds a value to a counter metric
func (mc *MetricsCollector) AddCounter(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.metricKey(name, labels)
	if metric, exists := mc.metrics[key]; exists {
		metric.Value += value
		metric.Timestamp = time.Now()
	} else {
		mc.metrics[key] = &Metric{
			Name:      name,
			Type:      MetricTypeCounter,
			Value:     value,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// SetGauge sets a gauge metric value
func (mc *MetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	mc.SetGaugeWithUnit(name, value, labels, "")
}

// SetGaugeWithUnit sets a gauge metric value with a unit
func (mc *MetricsCollector) SetGaugeWithUnit(name string, value float64, labels map[string]string, unit string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.metricKey(name, labels)
	mc.metrics[key] = &Metric{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
		Unit:      unit,
	}
}

// ObserveHistogram adds an observation to a histogram metric
func (mc *MetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.metricKey(name, labels)
	mc.metrics[key] = &Metric{
		Name:      name,
		Type:      MetricTypeHistogram,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
}

// RecordDuration records a duration as a histogram metric
func (mc *MetricsCollector) RecordDuration(name string, duration time.Duration, labels map[string]string) {
	mc.ObserveHistogram(name, duration.Seconds(), labels)
}

// GetMetrics returns a snapshot of all current metrics
func (mc *MetricsCollector) GetMetrics() map[string]*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	snapshot := make(map[string]*Metric)
	for k, v := range mc.metrics {
		// Create a copy of the metric
		snapshot[k] = &Metric{
			Name:      v.Name,
			Type:      v.Type,
			Value:     v.Value,
			Labels:    v.Labels,
			Timestamp: v.Timestamp,
			Unit:      v.Unit,
		}
	}
	return snapshot
}

// GetMetricsByType returns metrics filtered by type
func (mc *MetricsCollector) GetMetricsByType(metricType MetricType) []*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var filtered []*Metric
	for _, metric := range mc.metrics {
		if metric.Type == metricType {
			filtered = append(filtered, &Metric{
				Name:      metric.Name,
				Type:      metric.Type,
				Value:     metric.Value,
				Labels:    metric.Labels,
				Timestamp: metric.Timestamp,
				Unit:      metric.Unit,
			})
		}
	}
	return filtered
}

// Reset clears all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics = make(map[string]*Metric)
}

// Close stops the metrics collector
func (mc *MetricsCollector) Close() {
	mc.cancel()
	mc.wg.Wait()
	mc.eventLogger.Close()
}

func (mc *MetricsCollector) metricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	var b strings.Builder
	b.WriteString(name)

	// To ensure deterministic keys, sort the label keys
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		b.WriteString(",")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(labels[k])
	}

	return b.String()
}

func (mc *MetricsCollector) flushLoop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			mc.flushMetrics()
			return
		case <-ticker.C:
			mc.flushMetrics()
		}
	}
}

func (mc *MetricsCollector) flushMetrics() {
	// Get a list of metrics to flush without holding the lock during logging
	mc.mu.RLock()
	metricsToFlush := make([]*Metric, 0, len(mc.metrics))
	for _, metric := range mc.metrics {
		// Create a copy to avoid holding references to the original
		metricsToFlush = append(metricsToFlush, &Metric{
			Name:      metric.Name,
			Type:      metric.Type,
			Value:     metric.Value,
			Labels:    metric.Labels,
			Timestamp: metric.Timestamp,
			Unit:      metric.Unit,
		})
	}
	mc.mu.RUnlock()

	for _, metric := range metricsToFlush {
		switch metric.Type {
		case MetricTypeCounter:
			mc.logger.LogCounter(metric.Name, int64(metric.Value), metric.Labels)
		case MetricTypeGauge:
			mc.logger.LogGauge(metric.Name, metric.Value, metric.Labels)
		case MetricTypeHistogram:
			mc.logger.LogHistogram(metric.Name, metric.Value, metric.Labels)
		}
	}
}

// ApplicationMetrics provides high-level application metrics
type ApplicationMetrics struct {
	collector *MetricsCollector
}

// NewApplicationMetrics creates application-specific metrics
func NewApplicationMetrics(collector *MetricsCollector) *ApplicationMetrics {
	return &ApplicationMetrics{
		collector: collector,
	}
}

// RecordMatrixOperation records matrix operation metrics
func (am *ApplicationMetrics) RecordMatrixOperation(operation string, matrixID string, duration time.Duration, success bool) {
	labels := map[string]string{
		"operation": operation,
		"matrix_id": matrixID,
		"success":   "true",
	}

	if !success {
		labels["success"] = "false"
	}

	// Count operations
	am.collector.IncCounter("matrix_operations_total", labels)

	// Record duration
	am.collector.RecordDuration("matrix_operation_duration_seconds", duration, labels)
}

// RecordStatsCollection records statistics collection metrics
func (am *ApplicationMetrics) RecordStatsCollection(statsType string, value float64, duration time.Duration) {
	labels := map[string]string{
		"stats_type": statsType,
	}

	// Record the stat value itself
	am.collector.SetGauge("system_stats_"+statsType, value, nil)

	// Record collection duration
	am.collector.RecordDuration("stats_collection_duration_seconds", duration, labels)

	// Count collections
	am.collector.IncCounter("stats_collections_total", labels)
}

// RecordConfigReload records configuration reload metrics
func (am *ApplicationMetrics) RecordConfigReload(success bool, duration time.Duration) {
	labels := map[string]string{
		"success": "true",
	}

	if !success {
		labels["success"] = "false"
	}

	am.collector.IncCounter("config_reloads_total", labels)
	am.collector.RecordDuration("config_reload_duration_seconds", duration, labels)
}

// RecordDaemonUptime records daemon uptime
func (am *ApplicationMetrics) RecordDaemonUptime(uptime time.Duration) {
	am.collector.SetGaugeWithUnit("daemon_uptime_seconds", uptime.Seconds(), nil, "seconds")
}

// RecordMemoryUsage records memory usage metrics
func (am *ApplicationMetrics) RecordMemoryUsage(heapAlloc, heapSys, heapInuse uint64) {
	am.collector.SetGaugeWithUnit("memory_heap_alloc_bytes", float64(heapAlloc), nil, "bytes")
	am.collector.SetGaugeWithUnit("memory_heap_sys_bytes", float64(heapSys), nil, "bytes")
	am.collector.SetGaugeWithUnit("memory_heap_inuse_bytes", float64(heapInuse), nil, "bytes")
}

// RecordGoroutines records the number of goroutines
func (am *ApplicationMetrics) RecordGoroutines(count int) {
	am.collector.SetGauge("goroutines_count", float64(count), nil)
}

// RecordDisplayUpdate records display update metrics
func (am *ApplicationMetrics) RecordDisplayUpdate(mode string, success bool, duration time.Duration) {
	labels := map[string]string{
		"mode":    mode,
		"success": "true",
	}

	if !success {
		labels["success"] = "false"
	}

	am.collector.IncCounter("display_updates_total", labels)
	am.collector.RecordDuration("display_update_duration_seconds", duration, labels)
}

// Health check metrics
func (am *ApplicationMetrics) RecordHealthCheck(component string, healthy bool, duration time.Duration) {
	labels := map[string]string{
		"component": component,
		"healthy":   "true",
	}

	if !healthy {
		labels["healthy"] = "false"
	}

	am.collector.IncCounter("health_checks_total", labels)
	am.collector.RecordDuration("health_check_duration_seconds", duration, labels)

	// Set health status gauge
	healthValue := 1.0
	if !healthy {
		healthValue = 0.0
	}
	am.collector.SetGauge("component_health", healthValue, map[string]string{"component": component})
}

// Timer provides convenient timing functionality
type Timer struct {
	startTime time.Time
	name      string
	labels    map[string]string
	collector *MetricsCollector
}

// StartTimer creates and starts a new timer
func (mc *MetricsCollector) StartTimer(name string, labels map[string]string) *Timer {
	return &Timer{
		startTime: time.Now(),
		name:      name,
		labels:    labels,
		collector: mc,
	}
}

// Stop stops the timer and records the duration
func (t *Timer) Stop() time.Duration {
	duration := time.Since(t.startTime)
	t.collector.RecordDuration(t.name, duration, t.labels)
	return duration
}

// StopWithSuccess stops the timer and records success/failure
func (t *Timer) StopWithSuccess(success bool) time.Duration {
	duration := time.Since(t.startTime)

	labels := t.labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["success"] = "true"
	if !success {
		labels["success"] = "false"
	}

	t.collector.RecordDuration(t.name, duration, labels)
	return duration
}
