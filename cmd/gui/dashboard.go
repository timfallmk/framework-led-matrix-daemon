//go:build gui

package main

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/api"
)

// Dashboard displays real-time system metrics.
type Dashboard struct {
	cpuBar              *widget.ProgressBar
	cpuLabel            *widget.Label
	memBar              *widget.ProgressBar
	memLabel            *widget.Label
	diskLabel           *widget.Label
	netLabel            *widget.Label
	statusRect          *canvas.Rectangle
	statusLabel         *widget.Label
	matrixModeLabel     *widget.Label
	matrixInfoContainer *fyne.Container
	container           *fyne.Container
}

// NewDashboard creates a new metrics dashboard.
func NewDashboard() *Dashboard {
	d := &Dashboard{
		cpuBar:          widget.NewProgressBar(),
		cpuLabel:        widget.NewLabel("CPU: --"),
		memBar:          widget.NewProgressBar(),
		memLabel:        widget.NewLabel("Memory: --"),
		diskLabel:       widget.NewLabel("Disk I/O: --"),
		netLabel:        widget.NewLabel("Network: --"),
		statusRect:      canvas.NewRectangle(color.NRGBA{R: 128, G: 128, B: 128, A: 255}),
		statusLabel:     widget.NewLabel("Status: Unknown"),
		matrixModeLabel: widget.NewLabel("Matrix: single"),
	}

	d.matrixInfoContainer = container.NewVBox(d.matrixModeLabel)
	d.statusRect.SetMinSize(fyne.NewSize(20, 20))
	d.statusRect.CornerRadius = 10

	cpuSection := container.NewVBox(
		widget.NewLabelWithStyle("CPU Usage", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.cpuBar,
		d.cpuLabel,
	)

	memSection := container.NewVBox(
		widget.NewLabelWithStyle("Memory Usage", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.memBar,
		d.memLabel,
	)

	ioSection := container.NewVBox(
		widget.NewLabelWithStyle("I/O Activity", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.diskLabel,
		d.netLabel,
	)

	statusSection := container.NewHBox(
		d.statusRect,
		d.statusLabel,
	)

	matrixSection := container.NewVBox(
		widget.NewLabelWithStyle("Matrix Info", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.matrixInfoContainer,
	)

	d.container = container.NewVBox(
		widget.NewLabelWithStyle("System Metrics", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		cpuSection,
		widget.NewSeparator(),
		memSection,
		widget.NewSeparator(),
		ioSection,
		widget.NewSeparator(),
		statusSection,
		widget.NewSeparator(),
		matrixSection,
	)

	return d
}

// Container returns the dashboard's Fyne container.
func (d *Dashboard) Container() *fyne.Container {
	return d.container
}

// Update refreshes the dashboard with new metrics data.
func (d *Dashboard) Update(m *api.MetricsResult) {
	if m == nil {
		return
	}

	d.cpuBar.SetValue(m.CPUUsage / 100.0)
	d.cpuLabel.SetText(fmt.Sprintf("CPU: %.1f%%", m.CPUUsage))

	d.memBar.SetValue(m.MemoryUsage / 100.0)
	d.memLabel.SetText(fmt.Sprintf("Memory: %.1f%%", m.MemoryUsage))

	d.diskLabel.SetText(fmt.Sprintf("Disk I/O: %.1f KB/s", m.DiskActivity/1024.0))
	d.netLabel.SetText(fmt.Sprintf("Network: %.1f KB/s", m.NetworkActivity/1024.0))

	switch m.Status {
	case "normal":
		d.statusRect.FillColor = color.NRGBA{R: 0, G: 200, B: 0, A: 255}
		d.statusLabel.SetText("Status: Normal")
	case "warning":
		d.statusRect.FillColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
		d.statusLabel.SetText("Status: Warning")
	case "critical":
		d.statusRect.FillColor = color.NRGBA{R: 255, G: 0, B: 0, A: 255}
		d.statusLabel.SetText("Status: Critical")
	default:
		d.statusRect.FillColor = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
		d.statusLabel.SetText("Status: Unknown")
	}

	d.statusRect.Refresh()
}

// UpdateMatrixInfo refreshes the matrix information display.
func (d *Dashboard) UpdateMatrixInfo(status *api.StatusResult) {
if status == nil {
return
}

mode := status.MatrixMode
if mode == "" {
mode = "single"
}
d.matrixModeLabel.SetText(fmt.Sprintf("Matrix Mode: %s", mode))

// Rebuild per-matrix info
d.matrixInfoContainer.RemoveAll()
d.matrixInfoContainer.Add(d.matrixModeLabel)

if len(status.Matrices) > 0 {
for _, m := range status.Matrices {
info := fmt.Sprintf("  %s (%s)", m.Name, m.Role)
if len(m.Metrics) > 0 {
info += " — metrics: "
for i, metric := range m.Metrics {
if i > 0 {
info += ", "
}
info += metric
}
}
d.matrixInfoContainer.Add(widget.NewLabel(info))
}
}

d.matrixInfoContainer.Refresh()
}
