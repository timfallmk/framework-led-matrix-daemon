package observability

import (
	"testing"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/logging"
)

// Helper function to get the first metric from a map.
func getFirstMetric(metrics map[string]*Metric) *Metric {
	for _, m := range metrics {
		return m
	}

	return nil
}

func TestCopyLabels(t *testing.T) {
	tests := []struct {
		src  map[string]string
		want map[string]string
		name string
	}{
		{
			name: "nil map",
			src:  nil,
			want: nil,
		},
		{
			name: "empty map",
			src:  map[string]string{},
			want: nil,
		},
		{
			name: "single label",
			src:  map[string]string{"key1": "value1"},
			want: map[string]string{"key1": "value1"},
		},
		{
			name: "multiple labels",
			src:  map[string]string{"key1": "value1", "key2": "value2"},
			want: map[string]string{"key1": "value1", "key2": "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := copyLabels(tt.src)

			if len(result) != len(tt.want) {
				t.Errorf("copyLabels() length = %d, want %d", len(result), len(tt.want))

				return
			}

			for k, v := range tt.want {
				if result[k] != v {
					t.Errorf("copyLabels()[%s] = %v, want %v", k, result[k], v)
				}
			}

			// Test that it's a copy, not the same map
			if len(tt.src) > 0 {
				// Modify original
				tt.src["new"] = "value"

				if result["new"] == "value" {
					t.Error("copyLabels() returned reference to original map, not a copy")
				}
			}
		})
	}
}

func TestMetric(t *testing.T) {
	timestamp := time.Now()
	labels := map[string]string{"component": "test"}

	metric := Metric{
		Name:      "test_metric",
		Type:      MetricTypeCounter,
		Value:     42.0,
		Unit:      "bytes",
		Labels:    labels,
		Timestamp: timestamp,
	}

	if metric.Name != "test_metric" {
		t.Errorf("Metric.Name = %v, want test_metric", metric.Name)
	}

	if metric.Type != MetricTypeCounter {
		t.Errorf("Metric.Type = %v, want %v", metric.Type, MetricTypeCounter)
	}

	if metric.Value != 42.0 {
		t.Errorf("Metric.Value = %v, want 42.0", metric.Value)
	}

	if metric.Unit != "bytes" {
		t.Errorf("Metric.Unit = %v, want bytes", metric.Unit)
	}

	if len(metric.Labels) != 1 || metric.Labels["component"] != "test" {
		t.Errorf("Metric.Labels = %v, want map[component:test]", metric.Labels)
	}

	if !metric.Timestamp.Equal(timestamp) {
		t.Errorf("Metric.Timestamp = %v, want %v", metric.Timestamp, timestamp)
	}
}

func TestMetricType(t *testing.T) {
	tests := []struct {
		name       string
		metricType MetricType
		want       string
	}{
		{"counter", MetricTypeCounter, "counter"},
		{"gauge", MetricTypeGauge, "gauge"},
		{"histogram", MetricTypeHistogram, "histogram"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.metricType) != tt.want {
				t.Errorf("MetricType = %v, want %v", tt.metricType, tt.want)
			}
		})
	}
}

func TestNewMetricsCollector(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	if collector == nil {
		t.Fatal("NewMetricsCollector() returned nil")
	}

	defer collector.Close()

	if collector.metrics == nil {
		t.Error("NewMetricsCollector() did not initialize metrics map")
	}

	if collector.ctx == nil {
		t.Error("NewMetricsCollector() did not initialize context")
	}
}

func TestMetricsCollector_IncCounter(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	labels := map[string]string{"component": "test"}
	collector.IncCounter("test_counter", labels)

	metrics := collector.GetMetrics()
	if len(metrics) != 1 {
		t.Errorf("IncCounter() metrics count = %d, want 1", len(metrics))

		return
	}

	// Get the first (and only) metric from the map
	var metric *Metric
	for _, m := range metrics {
		metric = m

		break
	}

	if metric.Name != "test_counter" {
		t.Errorf("IncCounter() metric name = %v, want test_counter", metric.Name)
	}

	if metric.Type != MetricTypeCounter {
		t.Errorf("IncCounter() metric type = %v, want %v", metric.Type, MetricTypeCounter)
	}

	if metric.Value != 1.0 {
		t.Errorf("IncCounter() metric value = %v, want 1.0", metric.Value)
	}
}

func TestMetricsCollector_AddCounter(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	labels := map[string]string{"component": "test"}
	collector.AddCounter("test_counter", 5.0, labels)

	metrics := collector.GetMetrics()
	if len(metrics) != 1 {
		t.Errorf("AddCounter() metrics count = %d, want 1", len(metrics))

		return
	}

	metric := getFirstMetric(metrics)
	if metric.Value != 5.0 {
		t.Errorf("AddCounter() metric value = %v, want 5.0", metric.Value)
	}

	// Add more to same counter
	collector.AddCounter("test_counter", 3.0, labels)

	metrics = collector.GetMetrics()
	if len(metrics) != 1 {
		t.Errorf("AddCounter() second call metrics count = %d, want 1", len(metrics))

		return
	}

	metric = getFirstMetric(metrics)
	if metric.Value != 8.0 {
		t.Errorf("AddCounter() cumulative value = %v, want 8.0", metric.Value)
	}
}

func TestMetricsCollector_SetGauge(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	labels := map[string]string{"component": "test"}
	collector.SetGauge("test_gauge", 42.5, labels)

	metrics := collector.GetMetrics()
	if len(metrics) != 1 {
		t.Errorf("SetGauge() metrics count = %d, want 1", len(metrics))

		return
	}

	metric := getFirstMetric(metrics)
	if metric.Name != "test_gauge" {
		t.Errorf("SetGauge() metric name = %v, want test_gauge", metric.Name)
	}

	if metric.Type != MetricTypeGauge {
		t.Errorf("SetGauge() metric type = %v, want %v", metric.Type, MetricTypeGauge)
	}

	if metric.Value != 42.5 {
		t.Errorf("SetGauge() metric value = %v, want 42.5", metric.Value)
	}
}

func TestMetricsCollector_SetGaugeWithUnit(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	labels := map[string]string{"component": "test"}
	collector.SetGaugeWithUnit("test_gauge", 1024.0, labels, "bytes")

	metrics := collector.GetMetrics()
	if len(metrics) != 1 {
		t.Errorf("SetGaugeWithUnit() metrics count = %d, want 1", len(metrics))

		return
	}

	metric := getFirstMetric(metrics)
	if metric.Unit != "bytes" {
		t.Errorf("SetGaugeWithUnit() metric unit = %v, want bytes", metric.Unit)
	}

	if metric.Value != 1024.0 {
		t.Errorf("SetGaugeWithUnit() metric value = %v, want 1024.0", metric.Value)
	}
}

func TestMetricsCollector_ObserveHistogram(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	labels := map[string]string{"component": "test"}
	collector.ObserveHistogram("test_histogram", 0.5, labels)

	metrics := collector.GetMetrics()
	if len(metrics) != 1 {
		t.Errorf("ObserveHistogram() metrics count = %d, want 1", len(metrics))

		return
	}

	metric := getFirstMetric(metrics)
	if metric.Name != "test_histogram" {
		t.Errorf("ObserveHistogram() metric name = %v, want test_histogram", metric.Name)
	}

	if metric.Type != MetricTypeHistogram {
		t.Errorf("ObserveHistogram() metric type = %v, want %v", metric.Type, MetricTypeHistogram)
	}

	if metric.Value != 0.5 {
		t.Errorf("ObserveHistogram() metric value = %v, want 0.5", metric.Value)
	}
}

func TestMetricsCollector_RecordDuration(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	duration := 250 * time.Millisecond
	labels := map[string]string{"component": "test"}
	collector.RecordDuration("test_duration", duration, labels)

	metrics := collector.GetMetrics()
	if len(metrics) != 1 {
		t.Errorf("RecordDuration() metrics count = %d, want 1", len(metrics))

		return
	}

	metric := getFirstMetric(metrics)
	if metric.Type != MetricTypeHistogram {
		t.Errorf("RecordDuration() metric type = %v, want %v", metric.Type, MetricTypeHistogram)
	}

	// Duration should be converted to seconds
	expected := duration.Seconds()
	if metric.Value != expected {
		t.Errorf("RecordDuration() metric value = %v, want %v", metric.Value, expected)
	}
}

func TestMetricsCollector_GetMetrics(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	// Initially empty
	metrics := collector.GetMetrics()
	if len(metrics) != 0 {
		t.Errorf("GetMetrics() initial count = %d, want 0", len(metrics))
	}

	// Add some metrics
	labels := map[string]string{"component": "test"}
	collector.IncCounter("counter1", labels)
	collector.SetGauge("gauge1", 42.0, labels)
	collector.ObserveHistogram("histogram1", 0.5, labels)

	metrics = collector.GetMetrics()
	if len(metrics) != 3 {
		t.Errorf("GetMetrics() count after adding = %d, want 3", len(metrics))
	}
}

func TestMetricsCollector_GetMetricsByType(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	labels := map[string]string{"component": "test"}
	collector.IncCounter("counter1", labels)
	collector.IncCounter("counter2", labels)
	collector.SetGauge("gauge1", 42.0, labels)
	collector.ObserveHistogram("histogram1", 0.5, labels)

	counters := collector.GetMetricsByType(MetricTypeCounter)
	if len(counters) != 2 {
		t.Errorf("GetMetricsByType(Counter) count = %d, want 2", len(counters))
	}

	gauges := collector.GetMetricsByType(MetricTypeGauge)
	if len(gauges) != 1 {
		t.Errorf("GetMetricsByType(Gauge) count = %d, want 1", len(gauges))
	}

	histograms := collector.GetMetricsByType(MetricTypeHistogram)
	if len(histograms) != 1 {
		t.Errorf("GetMetricsByType(Histogram) count = %d, want 1", len(histograms))
	}
}

func TestMetricsCollector_Reset(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	labels := map[string]string{"component": "test"}
	collector.IncCounter("counter1", labels)
	collector.SetGauge("gauge1", 42.0, labels)

	metrics := collector.GetMetrics()
	if len(metrics) != 2 {
		t.Errorf("Metrics count before reset = %d, want 2", len(metrics))
	}

	collector.Reset()

	metrics = collector.GetMetrics()
	if len(metrics) != 0 {
		t.Errorf("Metrics count after reset = %d, want 0", len(metrics))
	}
}

func TestMetricsCollector_Close(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)

	// Should not panic
	collector.Close()

	// Context should be cancelled
	select {
	case <-collector.ctx.Done():
		// Expected
	case <-time.After(time.Second):
		t.Error("Close() did not cancel context")
	}
}

func TestNewApplicationMetrics(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	appMetrics := NewApplicationMetrics(collector)

	if appMetrics == nil {
		t.Fatal("NewApplicationMetrics() returned nil")
	}

	if appMetrics.collector != collector {
		t.Error("NewApplicationMetrics() did not set collector correctly")
	}
}

func TestApplicationMetrics_RecordMatrixOperation(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	appMetrics := NewApplicationMetrics(collector)

	appMetrics.RecordMatrixOperation("connect", "success", time.Millisecond*100, true)

	metrics := collector.GetMetrics()
	if len(metrics) < 1 {
		t.Errorf("RecordMatrixOperation() metrics count = %d, want at least 1", len(metrics))
	}
}

func TestApplicationMetrics_RecordStatsCollection(t *testing.T) {
	logger, err := logging.NewLogger(logging.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := NewMetricsCollector(logger, time.Second)
	defer collector.Close()

	appMetrics := NewApplicationMetrics(collector)

	appMetrics.RecordStatsCollection("cpu", 75.5, time.Millisecond*50)

	metrics := collector.GetMetrics()
	if len(metrics) < 1 {
		t.Errorf("RecordStatsCollection() metrics count = %d, want at least 1", len(metrics))
	}
}
