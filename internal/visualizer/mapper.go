// Package visualizer provides visualization components that convert system metrics to LED patterns.
// It supports both single and multi-matrix configurations with various display modes including
// percentage, gradient, activity, and status visualization modes.
package visualizer

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/config"
	"github.com/timfallmk/framework-led-matrix-daemon/internal/stats"
)

// DisplayManagerInterface defines the interface for display managers.
type DisplayManagerInterface interface {
	UpdatePercentage(key string, percent float64) error
	ShowActivity(active bool) error
	ShowStatus(status string) error
	SetBrightness(level byte) error
	GetCurrentState() map[string]interface{}
	SetUpdateRate(rate time.Duration)
}

// MultiDisplayManagerInterface defines the interface for multi-display managers.
type MultiDisplayManagerInterface interface {
	UpdateMetric(metricName string, value float64, stats map[string]float64) error
	UpdateActivity(active bool) error
	UpdateStatus(status string) error
	SetBrightness(level byte) error
	SetUpdateRate(rate time.Duration)
	HasMultipleDisplays() bool
}

// Visualizer converts system metrics into visual patterns for single LED matrix displays.
type Visualizer struct {
	display    DisplayManagerInterface
	config     *config.Config
	lastUpdate time.Time
}

// MultiVisualizer converts system metrics into visual patterns for multiple LED matrix displays.
type MultiVisualizer struct {
	multiDisplay MultiDisplayManagerInterface
	config       *config.Config
	lastUpdate   time.Time
}

// NewVisualizer creates a new Visualizer with the specified display manager and configuration.
func NewVisualizer(display DisplayManagerInterface, cfg *config.Config) *Visualizer {
	return &Visualizer{
		display: display,
		config:  cfg,
	}
}

// NewMultiVisualizer creates a new MultiVisualizer with the specified multi-display manager and configuration.
func NewMultiVisualizer(multiDisplay MultiDisplayManagerInterface, cfg *config.Config) *MultiVisualizer {
	return &MultiVisualizer{
		multiDisplay: multiDisplay,
		config:       cfg,
	}
}

// UpdateDisplay updates the LED matrix display based on the current system statistics and configured display mode.
func (v *Visualizer) UpdateDisplay(summary *stats.StatsSummary) error {
	if time.Since(v.lastUpdate) < v.config.Display.UpdateRate {
		return nil
	}

	switch v.config.Display.Mode {
	case "percentage":
		return v.updatePercentageMode(summary)
	case "gradient":
		return v.updateGradientMode(summary)
	case "activity":
		return v.updateActivityMode(summary)
	case "status":
		return v.updateStatusMode(summary)
	case "custom":
		return v.updateCustomMode(summary)
	default:
		return fmt.Errorf("unknown display mode: %s", v.config.Display.Mode)
	}
}

func (v *Visualizer) updatePercentageMode(summary *stats.StatsSummary) error {
	var value float64

	switch v.config.Display.PrimaryMetric {
	case "cpu":
		value = summary.CPUUsage
	case "memory":
		value = summary.MemoryUsage
	case "disk":
		value = v.normalizeActivity(summary.DiskActivity)
	case "network":
		value = v.normalizeActivity(summary.NetworkActivity)
	default:
		value = summary.CPUUsage
	}

	if err := v.display.UpdatePercentage(v.config.Display.PrimaryMetric, value); err != nil {
		return fmt.Errorf("failed to update percentage display: %w", err)
	}

	v.lastUpdate = time.Now()

	return nil
}

func (v *Visualizer) updateGradientMode(_ *stats.StatsSummary) error {
	if err := v.display.ShowStatus("normal"); err != nil {
		return fmt.Errorf("failed to show gradient: %w", err)
	}

	v.lastUpdate = time.Now()

	return nil
}

func (v *Visualizer) updateActivityMode(summary *stats.StatsSummary) error {
	isActive := v.isSystemActive(summary)

	if err := v.display.ShowActivity(isActive); err != nil {
		return fmt.Errorf("failed to update activity display: %w", err)
	}

	v.lastUpdate = time.Now()

	return nil
}

func (v *Visualizer) updateStatusMode(summary *stats.StatsSummary) error {
	status := summary.Status.String()

	if err := v.display.ShowStatus(status); err != nil {
		return fmt.Errorf("failed to update status display: %w", err)
	}

	v.lastUpdate = time.Now()

	return nil
}

func (v *Visualizer) updateCustomMode(summary *stats.StatsSummary) error {
	return fmt.Errorf("custom mode not yet implemented")
}

func (v *Visualizer) normalizeActivity(activity float64) float64 {
	const maxActivityRate = 10 * 1024 * 1024

	normalized := (activity / maxActivityRate) * 100
	if normalized > 100 {
		normalized = 100
	}

	return normalized
}

func (v *Visualizer) shouldAnimate(summary *stats.StatsSummary) bool {
	if !v.config.Display.EnableAnimation {
		return false
	}

	thresholds := v.config.Stats.Thresholds

	if summary.CPUUsage > thresholds.CPUWarning ||
		summary.MemoryUsage > thresholds.MemoryWarning {
		return true
	}

	if summary.DiskActivity > 1024*1024 || summary.NetworkActivity > 1024*1024 {
		return true
	}

	return false
}

func (v *Visualizer) isSystemActive(summary *stats.StatsSummary) bool {
	activityThreshold := 1024.0

	if summary.DiskActivity > activityThreshold ||
		summary.NetworkActivity > activityThreshold {
		return true
	}

	if summary.CPUUsage > 10.0 {
		return true
	}

	return false
}

// CreateCustomPattern creates a custom LED pattern from normalized float data with specified dimensions.
func (v *Visualizer) CreateCustomPattern(width, height int, data []float64) ([39]byte, error) {
	var pixels [39]byte

	if len(data) != width*height {
		return pixels, fmt.Errorf("data length mismatch: expected %d, got %d", width*height, len(data))
	}

	for i, value := range data {
		if i >= len(pixels) {
			break
		}

		normalized := math.Max(0, math.Min(1, value))
		pixels[i] = byte(normalized * 255)
	}

	return pixels, nil
}

// DrawCustomBitmap displays a custom bitmap pattern on the LED matrix (not yet implemented).
func (v *Visualizer) DrawCustomBitmap(pixels [39]byte) error {
	return fmt.Errorf("custom bitmap drawing not yet implemented")
}

// CreateProgressBar creates a progress bar pattern with the specified percentage and width.
func (v *Visualizer) CreateProgressBar(percent float64, width int) []byte {
	bar := make([]byte, width)
	filled := int((percent / 100.0) * float64(width))

	for i := 0; i < width; i++ {
		if i < filled {
			bar[i] = 255
		} else {
			bar[i] = 0
		}
	}

	return bar
}

// SetBrightness sets the LED matrix brightness level.
func (v *Visualizer) SetBrightness(level byte) error {
	return v.display.SetBrightness(level)
}

// GetCurrentState returns the current display state.
func (v *Visualizer) GetCurrentState() map[string]interface{} {
	return v.display.GetCurrentState()
}

// UpdateDisplay updates multiple LED matrix displays based on system statistics and dual mode configuration.
func (mv *MultiVisualizer) UpdateDisplay(summary *stats.StatsSummary) error {
	if time.Since(mv.lastUpdate) < mv.config.Display.UpdateRate {
		return nil
	}

	switch mv.config.Display.Mode {
	case "percentage":
		return mv.updatePercentageMode(summary)
	case "gradient":
		return mv.updateGradientMode(summary)
	case "activity":
		return mv.updateActivityMode(summary)
	case "status":
		return mv.updateStatusMode(summary)
	default:
		return mv.updatePercentageMode(summary)
	}
}

func (mv *MultiVisualizer) updatePercentageMode(summary *stats.StatsSummary) error {
	// Create stats map for all metrics
	statsMap := map[string]float64{
		"cpu":     summary.CPUUsage,
		"memory":  summary.MemoryUsage,
		"disk":    mv.normalizeActivity(summary.DiskActivity),
		"network": mv.normalizeActivity(summary.NetworkActivity),
	}

	// Update each configured metric
	var lastErr error

	for metric, value := range statsMap {
		if err := mv.multiDisplay.UpdateMetric(metric, value, statsMap); err != nil {
			lastErr = err
			log.Printf("Error updating metric %s: %v", metric, err)
		}
	}

	mv.lastUpdate = time.Now()

	return lastErr
}

func (mv *MultiVisualizer) updateGradientMode(_ *stats.StatsSummary) error {
	if err := mv.multiDisplay.UpdateStatus("normal"); err != nil {
		return fmt.Errorf("failed to show gradient: %w", err)
	}

	mv.lastUpdate = time.Now()

	return nil
}

func (mv *MultiVisualizer) updateActivityMode(summary *stats.StatsSummary) error {
	active := mv.isSystemActive(summary)

	if err := mv.multiDisplay.UpdateActivity(active); err != nil {
		return fmt.Errorf("failed to update activity: %w", err)
	}

	mv.lastUpdate = time.Now()

	return nil
}

func (mv *MultiVisualizer) updateStatusMode(summary *stats.StatsSummary) error {
	status := mv.determineSystemStatus(summary)

	if err := mv.multiDisplay.UpdateStatus(status); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	mv.lastUpdate = time.Now()

	return nil
}

func (mv *MultiVisualizer) normalizeActivity(activity float64) float64 {
	if activity <= 0 {
		return 0.0
	}

	maxActivityMB := 100.0 * 1024 * 1024
	normalized := (activity / maxActivityMB) * 100.0

	if normalized > 100.0 {
		return 100.0
	}

	return normalized
}

func (mv *MultiVisualizer) isSystemActive(summary *stats.StatsSummary) bool {
	activityThreshold := 1024.0

	if summary.DiskActivity > activityThreshold ||
		summary.NetworkActivity > activityThreshold {
		return true
	}

	if summary.CPUUsage > 10.0 {
		return true
	}

	return false
}

func (mv *MultiVisualizer) determineSystemStatus(summary *stats.StatsSummary) string {
	thresholds := mv.config.Stats.Thresholds

	if summary.CPUUsage > thresholds.CPUCritical ||
		summary.MemoryUsage > thresholds.MemoryCritical {
		return "critical"
	}

	if summary.CPUUsage > thresholds.CPUWarning ||
		summary.MemoryUsage > thresholds.MemoryWarning {
		return "warning"
	}

	return "normal"
}

// UpdateConfig updates the visualizer configuration and applies new settings including update rate and brightness.
func (v *Visualizer) UpdateConfig(cfg *config.Config) {
	v.config = cfg
	v.display.SetUpdateRate(cfg.Display.UpdateRate)

	if cfg.Matrix.Brightness != 0 {
		if err := v.SetBrightness(cfg.Matrix.Brightness); err != nil {
			log.Printf("Warning: failed to set brightness: %v", err)
		}
	}
}
