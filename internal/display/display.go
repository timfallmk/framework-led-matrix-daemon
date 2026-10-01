// Package display provides the MatrixDisplay interface and a global registry
// for pluggable LED matrix panel renderers. Both static patterns and animated
// or metric-driven displays implement the same interface.
//
// Adding a new display:
//
//  1. Create a file in this package (or any imported package).
//  2. Implement MatrixDisplay.
//  3. Call display.Register("my-name", func() MatrixDisplay { return &MyDisplay{} })
//     from an init() function.
//  4. Reference "my-name" in configs/config.yaml under display.panels.
package display

import (
	"fmt"
	"sort"
	"sync"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/stats"
)

// MatrixDisplay produces pixel frames for one 9×34 LED matrix panel.
// Render is called at the configured update_rate. Implementations that need
// a different internal rate (e.g. Game of Life at 350 ms) should track their
// own timer inside Render. The stats argument is nil-safe; non-metric displays
// can ignore it.
type MatrixDisplay interface {
	Render(s *stats.StatsSummary) [9][34]byte
}

// Factory constructs a fresh, independent MatrixDisplay instance.
type Factory func() MatrixDisplay

var (
	mu       sync.RWMutex
	registry = map[string]Factory{}
)

// Register makes a display available under name. Intended to be called from
// package-level init() functions so displays self-register on import.
// Panics on empty name, nil factory, or duplicate registration.
func Register(name string, f Factory) {
	if name == "" {
		panic("display.Register: name must not be empty")
	}
	if f == nil {
		panic("display.Register: factory must not be nil")
	}
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[name]; exists {
		panic("display.Register: display " + name + " already registered")
	}
	registry[name] = f
}

// New returns a fresh instance of the named display, or an error if the name
// has not been registered. The factory is called outside the registry lock to
// avoid holding the lock during potentially expensive construction.
func New(name string) (MatrixDisplay, error) {
	mu.RLock()
	f, ok := registry[name]
	if !ok {
		available := registered()
		mu.RUnlock()
		return nil, fmt.Errorf("display %q not registered (available: %v)", name, available)
	}
	mu.RUnlock()
	return f(), nil
}

// Registered returns all currently registered display names in sorted order.
func Registered() []string {
	mu.RLock()
	defer mu.RUnlock()
	return registered()
}

func registered() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
