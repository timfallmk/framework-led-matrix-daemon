//go:build gui

package main

import (
	"strconv"

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
	container        *fyne.Container

	// Track whether the user has manually changed each setting.
	// Once a user edits a value, we stop overwriting it from daemon polls.
	userEditedMode       bool
	userEditedMetric     bool
	userEditedBrightness bool
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
}
