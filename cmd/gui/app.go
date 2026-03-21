//go:build gui

package main

import (
	"context"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/api"
)

// GUIApp is the main GUI application.
type GUIApp struct {
	app        fyne.App
	window     fyne.Window
	client     *api.Client
	dashboard  *Dashboard
	ledPreview *LEDPreview
	settings   *Settings
	health     *HealthView
	statusBar  *widget.Label
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewGUIApp creates a new GUI application.
func NewGUIApp(fyneApp fyne.App, client *api.Client) *GUIApp {
	ctx, cancel := context.WithCancel(context.Background())

	g := &GUIApp{
		app:    fyneApp,
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}

	g.window = fyneApp.NewWindow("Framework LED Matrix")
	g.window.Resize(fyne.NewSize(800, 600))
	g.window.SetOnClosed(func() {
		g.cancel()
		g.client.Close()
	})

	g.statusBar = widget.NewLabel("Disconnected")

	g.dashboard = NewDashboard()
	g.ledPreview = NewLEDPreview()
	g.settings = NewSettings(client)
	g.health = NewHealthView()

	tabs := container.NewAppTabs(
		container.NewTabItem("Dashboard", g.dashboard.Container()),
		container.NewTabItem("LED Preview", g.ledPreview.Container()),
		container.NewTabItem("Settings", g.settings.Container()),
		container.NewTabItem("Health", g.health.Container()),
	)

	content := container.NewBorder(nil, g.statusBar, nil, nil, tabs)
	g.window.SetContent(content)

	return g
}

// Run starts the GUI application.
func (g *GUIApp) Run() {
	go g.connectionLoop()
	g.window.ShowAndRun()
}

func (g *GUIApp) connectionLoop() {
	const (
		minBackoff = 1 * time.Second
		maxBackoff = 30 * time.Second
	)

	backoff := minBackoff

	for {
		select {
		case <-g.ctx.Done():
			return
		default:
		}

		if err := g.client.Connect(); err != nil {
			fyne.Do(func() {
				g.statusBar.SetText("Disconnected - daemon not running")
			})

			// Exponential backoff with jitter
			jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
			sleep := backoff + jitter

			select {
			case <-g.ctx.Done():
				return
			case <-time.After(sleep):
			}

			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}

			continue
		}

		// Successfully connected; reset backoff
		backoff = minBackoff
		fyne.Do(func() {
			g.statusBar.SetText("Connected")
		})
		g.pollLoop()

		// If we get here, we got disconnected
		fyne.Do(func() {
			g.statusBar.SetText("Disconnected - reconnecting...")
		})
		g.client.Close()
	}
}

func (g *GUIApp) pollLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Initial fetch
	g.fetchAndUpdate()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			if err := g.fetchAndUpdate(); err != nil {
				return // Disconnected
			}
		}
	}
}

func (g *GUIApp) fetchAndUpdate() error {
	// Fetch metrics
	metrics, err := g.client.GetMetrics()
	if err != nil {
		return err
	}

	// Fetch status
	status, err := g.client.GetStatus()
	if err != nil {
		return err
	}

	// Fetch health
	health, err := g.client.GetHealth()
	if err != nil {
		return err
	}

	// Apply all UI updates on the Fyne main thread
	fyne.Do(func() {
		g.dashboard.Update(metrics)
		g.dashboard.UpdateMatrixInfo(status)

		matrixMode := status.MatrixMode
		if matrixMode == "" {
			matrixMode = "single"
		}
		isDual := matrixMode != "single"
		g.ledPreview.SetDualMode(isDual, matrixMode)
		g.ledPreview.UpdateFromMetrics(metrics)

		g.statusBar.SetText("Connected | Mode: " + status.DisplayMode +
			" | Metric: " + status.PrimaryMetric +
			" | Matrix: " + matrixMode)
		g.settings.UpdateFromStatus(status)
		g.settings.UpdateMatrixInfo(status)
		g.ledPreview.SetBrightnessDisplay(status.Brightness)

		g.health.Update(health)
	})

	return nil
}
