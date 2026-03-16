//go:build gui

package main

import (
	"fmt"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/api"
)

const (
	matrixCols = 34
	matrixRows = 9
	cellSize   = 14
	cellGap    = 2
)

// LEDPreview renders a visual preview of the LED matrix state.
type LEDPreview struct {
	cells       [matrixRows][matrixCols]*canvas.Rectangle
	modeLabel   *widget.Label
	brightLabel *widget.Label
	container   *fyne.Container
}

// NewLEDPreview creates a new LED matrix preview widget.
func NewLEDPreview() *LEDPreview {
	l := &LEDPreview{
		modeLabel:   widget.NewLabel("Mode: --"),
		brightLabel: widget.NewLabel("Brightness: --"),
	}

	grid := container.NewWithoutLayout()

	for row := 0; row < matrixRows; row++ {
		for col := 0; col < matrixCols; col++ {
			rect := canvas.NewRectangle(color.NRGBA{R: 20, G: 20, B: 30, A: 255})
			rect.SetMinSize(fyne.NewSize(cellSize, cellSize))
			rect.CornerRadius = 2
			rect.Move(fyne.NewPos(
				float32(col*(cellSize+cellGap)),
				float32(row*(cellSize+cellGap)),
			))
			rect.Resize(fyne.NewSize(cellSize, cellSize))

			l.cells[row][col] = rect
			grid.Add(rect)
		}
	}

	gridWidth := float32(matrixCols*(cellSize+cellGap) - cellGap)
	gridHeight := float32(matrixRows*(cellSize+cellGap) - cellGap)
	grid.Resize(fyne.NewSize(gridWidth, gridHeight))

	// Background behind the grid
	bg := canvas.NewRectangle(color.NRGBA{R: 10, G: 10, B: 20, A: 255})
	bg.SetMinSize(fyne.NewSize(gridWidth+20, gridHeight+20))

	gridWrapper := container.NewCenter(
		container.NewStack(bg, container.NewPadded(grid)),
	)

	l.container = container.NewVBox(
		widget.NewLabelWithStyle("LED Matrix Preview", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		gridWrapper,
		widget.NewSeparator(),
		container.NewHBox(l.modeLabel, widget.NewLabel(" | "), l.brightLabel),
	)

	return l
}

// Container returns the LED preview's Fyne container.
func (l *LEDPreview) Container() *fyne.Container {
	return l.container
}

// UpdateFromMetrics generates an LED pattern based on current metrics.
func (l *LEDPreview) UpdateFromMetrics(m *api.MetricsResult) {
	if m == nil {
		return
	}

	l.modeLabel.SetText(fmt.Sprintf("Mode: %s", m.Status))

	// Generate a percentage bar pattern based on CPU usage
	l.renderPercentageBar(m.CPUUsage)
}

// renderPercentageBar fills columns from left to right proportional to the value.
func (l *LEDPreview) renderPercentageBar(percentage float64) {
	filledCols := int(math.Round(percentage / 100.0 * float64(matrixCols)))

	for col := 0; col < matrixCols; col++ {
		for row := 0; row < matrixRows; row++ {
			if col < filledCols {
				// Gradient from green to red based on percentage
				ratio := float64(col) / float64(matrixCols)
				r := uint8(math.Min(255, ratio*2*255))
				g := uint8(math.Min(255, (1-ratio)*2*255))
				l.cells[row][col].FillColor = color.NRGBA{R: r, G: g, B: 20, A: 255}
			} else {
				l.cells[row][col].FillColor = color.NRGBA{R: 20, G: 20, B: 30, A: 255}
			}

			l.cells[row][col].Refresh()
		}
	}
}

// SetBrightnessDisplay updates the brightness label.
func (l *LEDPreview) SetBrightnessDisplay(brightness int) {
	l.brightLabel.SetText(fmt.Sprintf("Brightness: %d/255", brightness))
}
