//go:build gui

package main

import (
"fmt"
"strconv"
"strings"

"fyne.io/fyne/v2"
"fyne.io/fyne/v2/container"
"fyne.io/fyne/v2/widget"

"github.com/timfallmk/framework-led-matrix-daemon/internal/api"
)

// Settings provides configuration editing controls.
type Settings struct {
client           *api.Client
modeSelect       *widget.Select
metricSelect     *widget.Select
brightnessSlider *widget.Slider
brightnessLabel  *widget.Label
logLevelSelect   *widget.Select
statusLabel      *widget.Label

// Dual matrix controls
dualModeSelect      *widget.Select
matrixInfoContainer *fyne.Container

container *fyne.Container

// Track whether the user has manually changed each setting.
// Once a user edits a value, we stop overwriting it from daemon polls.
userEditedMode       bool
userEditedMetric     bool
userEditedBrightness bool
userEditedDualMode   bool
}

// NewSettings creates a new settings editor.
func NewSettings(client *api.Client) *Settings {
s := &Settings{
client:      client,
statusLabel: widget.NewLabel(""),
}

// Display mode
s.modeSelect = widget.NewSelect(
[]string{"percentage", "gradient", "activity", "status"},
func(mode string) {
s.userEditedMode = true
if err := client.SetDisplayMode(mode); err != nil {
s.statusLabel.SetText("Error: " + err.Error())
} else {
s.statusLabel.SetText("Mode updated to " + mode)
}
},
)
s.modeSelect.PlaceHolder = "Select mode"

// Primary metric
s.metricSelect = widget.NewSelect(
[]string{"cpu", "memory", "disk", "network"},
func(metric string) {
s.userEditedMetric = true
if err := client.SetPrimaryMetric(metric); err != nil {
s.statusLabel.SetText("Error: " + err.Error())
} else {
s.statusLabel.SetText("Metric updated to " + metric)
}
},
)
s.metricSelect.PlaceHolder = "Select metric"

// Brightness
s.brightnessLabel = widget.NewLabel("Brightness: 100")
s.brightnessSlider = widget.NewSlider(0, 255)
s.brightnessSlider.Step = 1
s.brightnessSlider.Value = 100
s.brightnessSlider.OnChanged = func(val float64) {
level := int(val)
s.brightnessLabel.SetText("Brightness: " + strconv.Itoa(level))
}
s.brightnessSlider.OnChangeEnded = func(val float64) {
level := int(val)
s.userEditedBrightness = true
if err := client.SetBrightness(level); err != nil {
s.statusLabel.SetText("Error: " + err.Error())
} else {
s.statusLabel.SetText("Brightness updated to " + strconv.Itoa(level))
}
}

// Log level
s.logLevelSelect = widget.NewSelect(
[]string{"debug", "info", "warn", "error"},
func(level string) {
s.statusLabel.SetText("Log level change requires daemon restart")
},
)
s.logLevelSelect.PlaceHolder = "Select log level"

// Dual matrix mode
s.dualModeSelect = widget.NewSelect(
[]string{"single", "mirror", "split", "extended", "independent"},
func(mode string) {
s.userEditedDualMode = true
if err := client.SetDualMode(mode); err != nil {
s.statusLabel.SetText("Error: " + err.Error())
} else {
s.statusLabel.SetText("Matrix mode updated to " + mode)
}
},
)
s.dualModeSelect.PlaceHolder = "Select matrix mode"

s.matrixInfoContainer = container.NewVBox()

displaySection := container.NewVBox(
widget.NewLabelWithStyle("Display Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
widget.NewLabel("Display Mode:"),
s.modeSelect,
widget.NewLabel("Primary Metric:"),
s.metricSelect,
)

matrixSection := container.NewVBox(
widget.NewLabelWithStyle("Matrix Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
s.brightnessLabel,
s.brightnessSlider,
)

dualMatrixSection := container.NewVBox(
widget.NewLabelWithStyle("Dual Matrix", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
widget.NewLabel("Matrix Mode:"),
s.dualModeSelect,
s.matrixInfoContainer,
)

loggingSection := container.NewVBox(
widget.NewLabelWithStyle("Logging", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
widget.NewLabel("Log Level:"),
s.logLevelSelect,
)

s.container = container.NewVBox(
widget.NewLabelWithStyle("Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
widget.NewSeparator(),
displaySection,
widget.NewSeparator(),
matrixSection,
widget.NewSeparator(),
dualMatrixSection,
widget.NewSeparator(),
loggingSection,
widget.NewSeparator(),
s.statusLabel,
)

return s
}

// Container returns the settings' Fyne container.
func (s *Settings) Container() *fyne.Container {
return s.container
}

// UpdateFromStatus refreshes the settings display with current daemon values.
// It skips fields the user has explicitly edited to avoid overwriting active changes.
func (s *Settings) UpdateFromStatus(status *api.StatusResult) {
if status == nil {
return
}

if !s.userEditedMode {
s.modeSelect.SetSelected(status.DisplayMode)
s.userEditedMode = false // SetSelected fires callback, reset the flag
}

if !s.userEditedMetric {
s.metricSelect.SetSelected(status.PrimaryMetric)
s.userEditedMetric = false
}

if !s.userEditedBrightness {
s.brightnessSlider.SetValue(float64(status.Brightness))
s.brightnessLabel.SetText("Brightness: " + strconv.Itoa(status.Brightness))
}

if !s.userEditedDualMode {
mode := status.MatrixMode
if mode == "" {
mode = "single"
}
s.dualModeSelect.SetSelected(mode)
s.userEditedDualMode = false
}
}

// UpdateMatrixInfo refreshes per-matrix details in the settings view.
func (s *Settings) UpdateMatrixInfo(status *api.StatusResult) {
if status == nil {
return
}

s.matrixInfoContainer.RemoveAll()

if len(status.Matrices) == 0 {
s.matrixInfoContainer.Add(widget.NewLabel("No per-matrix info available"))
s.matrixInfoContainer.Refresh()
return
}

for _, m := range status.Matrices {
name := m.Name
if name == "" {
name = "unnamed"
}

var metricsStr string
if len(m.Metrics) > 0 {
metricsStr = strings.Join(m.Metrics, ", ")
} else {
metricsStr = "all"
}

info := fmt.Sprintf("%s (%s) — brightness: %d, metrics: %s",
name, m.Role, m.Brightness, metricsStr)
s.matrixInfoContainer.Add(widget.NewLabel(info))
}

s.matrixInfoContainer.Refresh()
}
