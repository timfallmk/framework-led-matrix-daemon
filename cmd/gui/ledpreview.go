//go:build gui

package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

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

var defaultCellColor = color.NRGBA{R: 20, G: 20, B: 30, A: 255}

// MatrixGrid encapsulates a single 34x9 LED matrix grid.
type MatrixGrid struct {
	cells     [matrixRows][matrixCols]*canvas.Rectangle
	label     *widget.Label
	container *fyne.Container
}

func newMatrixGrid(name string) *MatrixGrid {
	mg := &MatrixGrid{
		label: widget.NewLabel(name),
	}

	grid := container.NewWithoutLayout()

	for row := 0; row < matrixRows; row++ {
		for col := 0; col < matrixCols; col++ {
			rect := canvas.NewRectangle(defaultCellColor)
			rect.SetMinSize(fyne.NewSize(cellSize, cellSize))
			rect.CornerRadius = 2
			rect.Move(fyne.NewPos(
				float32(col*(cellSize+cellGap)),
				float32(row*(cellSize+cellGap)),
			))
			rect.Resize(fyne.NewSize(cellSize, cellSize))

			mg.cells[row][col] = rect
			grid.Add(rect)
		}
	}

	gridWidth := float32(matrixCols*(cellSize+cellGap) - cellGap)
	gridHeight := float32(matrixRows*(cellSize+cellGap) - cellGap)
	grid.Resize(fyne.NewSize(gridWidth, gridHeight))

	bg := canvas.NewRectangle(color.NRGBA{R: 10, G: 10, B: 20, A: 255})
	bg.SetMinSize(fyne.NewSize(gridWidth+20, gridHeight+20))

	gridWrapper := container.NewCenter(
		container.NewStack(bg, container.NewPadded(grid)),
	)

	mg.container = container.NewVBox(
		container.NewCenter(mg.label),
		gridWrapper,
	)

	return mg
}

func (mg *MatrixGrid) renderPercentageBar(percentage float64) {
	filledCols := int(math.Round(percentage / 100.0 * float64(matrixCols)))

	for col := 0; col < matrixCols; col++ {
		for row := 0; row < matrixRows; row++ {
			if col < filledCols {
				ratio := float64(col) / float64(matrixCols)
				r := uint8(math.Min(255, ratio*2*255))
				g := uint8(math.Min(255, (1-ratio)*2*255))
				mg.cells[row][col].FillColor = color.NRGBA{R: r, G: g, B: 20, A: 255}
			} else {
				mg.cells[row][col].FillColor = defaultCellColor
			}

			mg.cells[row][col].Refresh()
		}
	}
}

func (mg *MatrixGrid) clear() {
	for row := 0; row < matrixRows; row++ {
		for col := 0; col < matrixCols; col++ {
			mg.cells[row][col].FillColor = defaultCellColor
			mg.cells[row][col].Refresh()
		}
	}
}

// LEDPreview renders a visual preview of the LED matrix state.
type LEDPreview struct {
	primary     *MatrixGrid
	secondary   *MatrixGrid
	modeLabel   *widget.Label
	brightLabel *widget.Label
	gridArea    *fyne.Container
	container   *fyne.Container
	dualMode    bool
	matrixMode  string
}

// NewLEDPreview creates a new LED matrix preview widget.
func NewLEDPreview() *LEDPreview {
	l := &LEDPreview{
		modeLabel:   widget.NewLabel("Mode: --"),
		brightLabel: widget.NewLabel("Brightness: --"),
		primary:     newMatrixGrid("Primary"),
	}

	l.gridArea = container.NewVBox(l.primary.container)

	l.container = container.NewVBox(
		widget.NewLabelWithStyle("LED Matrix Preview", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		l.gridArea,
		widget.NewSeparator(),
		container.NewHBox(l.modeLabel, widget.NewLabel(" | "), l.brightLabel),
	)

	return l
}

// Container returns the LED preview's Fyne container.
func (l *LEDPreview) Container() *fyne.Container {
	return l.container
}

// SetDualMode enables or disables dual matrix display.
func (l *LEDPreview) SetDualMode(isDual bool, mode string) {
	if isDual == l.dualMode && mode == l.matrixMode {
		return
	}

	l.dualMode = isDual
	l.matrixMode = mode

	l.gridArea.RemoveAll()

	if isDual {
		if l.secondary == nil {
			l.secondary = newMatrixGrid("Secondary")
		}

		// Label grids based on mode
		switch strings.ToLower(mode) {
		case "split":
			l.primary.label.SetText("Primary (CPU)")
			l.secondary.label.SetText("Secondary (Memory)")
		case "mirror":
			l.primary.label.SetText("Primary (Mirrored)")
			l.secondary.label.SetText("Secondary (Mirrored)")
		default:
			l.primary.label.SetText("Primary")
			l.secondary.label.SetText("Secondary")
		}

		l.gridArea.Add(container.NewGridWithColumns(2,
			l.primary.container,
			l.secondary.container,
		))
	} else {
		l.primary.label.SetText("Matrix")
		l.gridArea.Add(l.primary.container)
		if l.secondary != nil {
			l.secondary.clear()
		}
	}

	l.gridArea.Refresh()
}

// UpdateFromMetrics generates LED patterns based on current metrics.
func (l *LEDPreview) UpdateFromMetrics(m *api.MetricsResult) {
	if m == nil {
		return
	}

	l.modeLabel.SetText(fmt.Sprintf("Mode: %s", m.Status))

	if l.dualMode && l.secondary != nil {
		switch strings.ToLower(l.matrixMode) {
		case "split":
			l.primary.renderPercentageBar(m.CPUUsage)
			l.secondary.renderPercentageBar(m.MemoryUsage)
		case "mirror":
			l.primary.renderPercentageBar(m.CPUUsage)
			l.secondary.renderPercentageBar(m.CPUUsage)
		default:
			l.primary.renderPercentageBar(m.CPUUsage)
			l.secondary.renderPercentageBar(m.MemoryUsage)
		}
	} else {
		l.primary.renderPercentageBar(m.CPUUsage)
	}
}

// SetBrightnessDisplay updates the brightness label.
func (l *LEDPreview) SetBrightnessDisplay(brightness int) {
	l.brightLabel.SetText(fmt.Sprintf("Brightness: %d/255", brightness))
}
