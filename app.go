package main

import (
	"context"

	"yaria/pkg/appconfig"
)

// App is the main application struct, providing Wails lifecycle hooks.
type App struct {
	ctx context.Context
}

// startup is called when the app starts. It initializes the shared config.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	appconfig.Init()
}

// shutdown is called when the app is closing. Placeholder for cleanup.
func (a *App) shutdown(ctx context.Context) {
	// nothing to clean up yet
}
