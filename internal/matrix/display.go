package matrix

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ClientInterface defines the interface that DisplayManager needs from a client.
type ClientInterface interface {
	ShowPercentage(percent byte) error
	ShowZigZag() error
	ShowGradient() error
	ShowFullBright() error
	SetBrightness(level byte) error
}

// DisplayManager manages display operations for a single LED matrix with rate limiting and state tracking.
type DisplayManager struct {
	lastUpdate   time.Time
	client       ClientInterface
	currentState map[string]interface{}
	updateRate   time.Duration
	mu           sync.RWMutex
}

// NewDisplayManager creates a new DisplayManager with the specified client and default update rate.
func NewDisplayManager(client ClientInterface) *DisplayManager {
	return &DisplayManager{
		client:       client,
		updateRate:   time.Second,
		currentState: make(map[string]interface{}),
	}
}

// SetUpdateRate sets the minimum time interval between display updates to prevent flickering.
func (dm *DisplayManager) SetUpdateRate(rate time.Duration) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.updateRate = rate
}

func (dm *DisplayManager) shouldUpdate() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return time.Since(dm.lastUpdate) >= dm.updateRate
}

func (dm *DisplayManager) markUpdatedUnsafe() {
	dm.lastUpdate = time.Now()
}

// UpdatePercentage updates the matrix display with a percentage value if sufficient time has passed
// and the value changed significantly.
func (dm *DisplayManager) UpdatePercentage(key string, percent float64) error {
	if !dm.shouldUpdate() {
		return nil
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	if lastPercent, exists := dm.currentState[key]; exists {
		if lastPercentFloat, ok := lastPercent.(float64); ok {
			if abs(lastPercentFloat-percent) < 1.0 {
				return nil
			}
		}
	}

	percentByte := byte(percent)
	if percentByte > 100 {
		percentByte = 100
	}

	if err := dm.client.ShowPercentage(percentByte); err != nil {
		return fmt.Errorf("failed to update percentage display: %w", err)
	}

	dm.currentState[key] = percent
	dm.markUpdatedUnsafe()
	log.Printf("Updated %s percentage display: %.1f%%", key, percent)

	return nil
}

// ShowActivity displays activity indication using zigzag pattern for active state or gradient for inactive state.
func (dm *DisplayManager) ShowActivity(active bool) error {
	if !dm.shouldUpdate() {
		return nil
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	var err error
	if active {
		err = dm.client.ShowZigZag()
		dm.currentState["activity"] = true
	} else {
		err = dm.client.ShowGradient()
		dm.currentState["activity"] = false
	}

	if err != nil {
		return fmt.Errorf("failed to update activity display: %w", err)
	}

	dm.markUpdatedUnsafe()
	log.Printf("Updated activity display: %v", active)

	return nil
}

// ShowStatus displays system status using different LED patterns (normal=gradient, warning=zigzag,
// critical=full bright, off=no brightness).
func (dm *DisplayManager) ShowStatus(status string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	var err error

	switch status {
	case "normal":
		err = dm.client.ShowGradient()
	case "warning":
		err = dm.client.ShowZigZag()
	case "critical":
		err = dm.client.ShowFullBright()
	case "off":
		err = dm.client.SetBrightness(0)
	default:
		return fmt.Errorf("unknown status: %s", status)
	}

	if err != nil {
		return fmt.Errorf("failed to update status display: %w", err)
	}

	dm.currentState["status"] = status
	dm.markUpdatedUnsafe()
	log.Printf("Updated status display: %s", status)

	return nil
}

// SetBrightness sets the LED matrix brightness level from 0-255.
func (dm *DisplayManager) SetBrightness(level byte) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if err := dm.client.SetBrightness(level); err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}

	dm.currentState["brightness"] = level
	dm.markUpdatedUnsafe()
	log.Printf("Set brightness: %d", level)

	return nil
}

// GetCurrentState returns a copy of the current display state including all tracked values.
func (dm *DisplayManager) GetCurrentState() map[string]interface{} {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	state := make(map[string]interface{})
	for k, v := range dm.currentState {
		state[k] = v
	}

	return state
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}

	return x
}

// MultiDisplayManager manages multiple DisplayManagers for dual matrix support.
type MultiDisplayManager struct {
	displays    map[string]*DisplayManager
	multiClient *MultiClient
	dualMode    string
	mu          sync.RWMutex
}

// NewMultiDisplayManager creates a new MultiDisplayManager with the specified multi-client and dual mode configuration.
func NewMultiDisplayManager(multiClient *MultiClient, dualMode string) *MultiDisplayManager {
	mdm := &MultiDisplayManager{
		displays:    make(map[string]*DisplayManager),
		multiClient: multiClient,
		dualMode:    dualMode,
	}

	for name, client := range multiClient.GetClients() {
		mdm.displays[name] = NewDisplayManager(client)
	}

	return mdm
}

// UpdateMetric updates displays with metric data according to the configured dual mode
// (mirror, split, extended, or independent).
func (mdm *MultiDisplayManager) UpdateMetric(metricName string, value float64, stats map[string]float64) error {
	mdm.mu.RLock()
	defer mdm.mu.RUnlock()

	switch mdm.dualMode {
	case "mirror":
		return mdm.updateMirrorMode(metricName, value)
	case "split":
		return mdm.updateSplitMode(metricName, value)
	case "extended":
		return mdm.updateExtendedMode(metricName, value)
	case "independent":
		return mdm.updateIndependentMode(metricName, value)
	default:
		return mdm.updateSplitMode(metricName, value) // Default to split mode
	}
}

func (mdm *MultiDisplayManager) updateMirrorMode(metricName string, value float64) error {
	// Show the same content on all matrices
	var lastErr error

	for _, display := range mdm.displays {
		if err := display.UpdatePercentage(metricName, value); err != nil {
			lastErr = err
			log.Printf("Error updating mirror display: %v", err)
		}
	}

	return lastErr
}

func (mdm *MultiDisplayManager) updateSplitMode(metricName string, value float64) error {
	// Each matrix shows different metrics based on configuration
	var lastErr error

	for name, display := range mdm.displays {
		matrixConfig := mdm.multiClient.GetConfig(name)
		if matrixConfig == nil {
			continue
		}

		// Check if this matrix should display the current metric
		shouldDisplay := false

		for _, assignedMetric := range matrixConfig.Metrics {
			if assignedMetric == metricName {
				shouldDisplay = true

				break
			}
		}

		if shouldDisplay {
			if err := display.UpdatePercentage(metricName, value); err != nil {
				lastErr = err
				log.Printf("Error updating split display %s with %s: %v", name, metricName, err)
			} else {
				log.Printf("Updated matrix %s with %s: %.1f%%", name, metricName, value)
			}
		} else {
			// If this matrix has no assigned metrics, show primary metric
			if len(matrixConfig.Metrics) == 0 && metricName == "cpu" {
				if err := display.UpdatePercentage(metricName, value); err != nil {
					lastErr = err
					log.Printf("Error updating fallback display %s with %s: %v", name, metricName, err)
				}
			}
		}
	}

	return lastErr
}

func (mdm *MultiDisplayManager) updateExtendedMode(metricName string, value float64) error {
	// Show a wider visualization across both matrices
	// For now, treat it like split mode but could be enhanced later
	return mdm.updateSplitMode(metricName, value)
}

func (mdm *MultiDisplayManager) updateIndependentMode(metricName string, value float64) error {
	// Each matrix operates completely independently
	// This would require more sophisticated configuration
	return mdm.updateSplitMode(metricName, value)
}

// UpdateActivity updates activity indication on all managed displays.
func (mdm *MultiDisplayManager) UpdateActivity(active bool) error {
	mdm.mu.RLock()
	defer mdm.mu.RUnlock()

	var lastErr error

	for name, display := range mdm.displays {
		if err := display.ShowActivity(active); err != nil {
			lastErr = err
			log.Printf("Error updating activity on display %s: %v", name, err)
		}
	}

	return lastErr
}

// UpdateStatus updates status display on all managed displays.
func (mdm *MultiDisplayManager) UpdateStatus(status string) error {
	mdm.mu.RLock()
	defer mdm.mu.RUnlock()

	var lastErr error

	for name, display := range mdm.displays {
		if err := display.ShowStatus(status); err != nil {
			lastErr = err
			log.Printf("Error updating status on display %s: %v", name, err)
		}
	}

	return lastErr
}

// SetBrightness sets brightness level on all managed displays.
func (mdm *MultiDisplayManager) SetBrightness(level byte) error {
	mdm.mu.RLock()
	defer mdm.mu.RUnlock()

	var lastErr error

	for name, display := range mdm.displays {
		if err := display.SetBrightness(level); err != nil {
			lastErr = err
			log.Printf("Error setting brightness on display %s: %v", name, err)
		}
	}

	return lastErr
}

// SetUpdateRate sets the update rate on all managed displays.
func (mdm *MultiDisplayManager) SetUpdateRate(rate time.Duration) {
	mdm.mu.RLock()
	defer mdm.mu.RUnlock()

	for _, display := range mdm.displays {
		display.SetUpdateRate(rate)
	}
}

// GetDisplayManager returns the DisplayManager for the specified matrix name.
func (mdm *MultiDisplayManager) GetDisplayManager(name string) *DisplayManager {
	mdm.mu.RLock()
	defer mdm.mu.RUnlock()

	return mdm.displays[name]
}

// HasMultipleDisplays returns true if managing more than one display.
func (mdm *MultiDisplayManager) HasMultipleDisplays() bool {
	mdm.mu.RLock()
	defer mdm.mu.RUnlock()

	return len(mdm.displays) > 1
}
