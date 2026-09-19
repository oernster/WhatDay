// Command installer is WhatDay's setup program: a Wails app carrying the
// built application as an embedded zip payload. It installs, updates, goes
// back, repairs and uninstalls, all per user with no administrator rights.
// Ported from PigeonPost's installer.
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// payload is the application, zipped by build.ps1; the committed file is an
// empty zip so the tree builds without a full build.
//
//go:embed payload.zip
var payload []byte

// appVersion is set from VERSION by build.ps1 through -ldflags.
var appVersion = "0.0.0-dev"

// The window is fixed in size; it fits the tallest screen with the 126 px mark.
const (
	windowTitle = "WhatDay Setup"
	windowW     = 640
	windowH     = 560
)

// background is the page's own background colour, so no white flash shows
// before the page paints.
var background = &options.RGBA{R: 0x16, G: 0x18, B: 0x1D, A: 1}

func main() {
	app := NewApp(payload, appVersion)
	_ = wails.Run(&options.App{
		Title:            windowTitle,
		Width:            windowW,
		Height:           windowH,
		DisableResize:    true,
		BackgroundColour: background,
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
	})
}
