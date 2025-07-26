package matrix

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ClientInterface defines the interface that DisplayManager needs from a client
type ClientInterface interface {
	ShowPercentage(percent byte) error
	ShowZigZag() error
	ShowGradient() error
	ShowFullBright() error
	SetBrightness(level byte) error
}

type DisplayManager struct {
	client       ClientInterface
	mu           sync.RWMutex
	lastUpdate   time.Time
	updateRate   time.Duration
	currentState map[string]interface{}
}

func NewDisplayManager(client ClientInterface) *DisplayManager {
	return &DisplayManager{
		client:       client,
		updateRate:   time.Second,
		currentState: make(map[string]interface{}),
	}
}

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

func (dm *DisplayManager) markUpdated() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.lastUpdate = time.Now()
}

func (dm *DisplayManager) markUpdatedUnsafe() {
	dm.lastUpdate = time.Now()
}

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
