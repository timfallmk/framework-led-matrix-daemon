//go:build gui

package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/api"
)

// HealthView displays health check results.
type HealthView struct {
	list      *fyne.Container
	container *fyne.Container
}

// NewHealthView creates a new health status display.
func NewHealthView() *HealthView {
	h := &HealthView{
		list: container.NewVBox(),
	}

	h.container = container.NewVBox(
		widget.NewLabelWithStyle("Component Health", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		h.list,
	)

	return h
}

// Container returns the health view's Fyne container.
func (h *HealthView) Container() *fyne.Container {
	return h.container
}

// Update refreshes the health display with new check results.
func (h *HealthView) Update(checks []api.HealthCheckResult) {
	h.list.RemoveAll()

	if len(checks) == 0 {
		h.list.Add(widget.NewLabel("No health checks available"))
		return
	}

	for _, check := range checks {
		statusColor := statusToColor(check.Status)
		indicator := canvas.NewRectangle(statusColor)
		indicator.SetMinSize(fyne.NewSize(12, 12))
		indicator.CornerRadius = 6

		nameLabel := widget.NewLabelWithStyle(check.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

		detailText := "Status: " + check.Status
		if check.Duration != "" {
			detailText += " | Duration: " + check.Duration
		}

		if check.Error != "" {
			detailText += " | Error: " + check.Error
		}

		detailLabel := widget.NewLabel(detailText)
		detailLabel.Wrapping = fyne.TextWrapWord

		row := container.NewHBox(indicator, container.NewVBox(nameLabel, detailLabel))
		h.list.Add(row)
		h.list.Add(widget.NewSeparator())
	}
}

func statusToColor(status string) color.Color {
	switch status {
	case "healthy":
		return color.NRGBA{R: 0, G: 200, B: 0, A: 255}
	case "unhealthy":
		return color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	case "starting":
		return color.NRGBA{R: 255, G: 165, B: 0, A: 255}
	default:
		return color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	}
}
