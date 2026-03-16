//go:build gui

// Package main provides the GUI application for the Framework LED Matrix daemon.
// It connects to the running daemon via a Unix domain socket and provides
// real-time metrics visualization, LED matrix preview, configuration editing,
// and health monitoring.
package main

import (
	"flag"
	"fmt"
	"os"

	"fyne.io/fyne/v2/app"

	apiPkg "github.com/timfallmk/framework-led-matrix-daemon/internal/api"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	socketPath := flag.String("socket", apiPkg.DefaultSocketPath, "daemon API socket path")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("framework-led-gui %s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	client := apiPkg.NewClient(*socketPath)

	fyneApp := app.NewWithID("com.framework.led-matrix-gui")
	guiApp := NewGUIApp(fyneApp, client)
	guiApp.Run()
}
