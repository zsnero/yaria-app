package main

import (
	"context"
	"embed"
	"io/fs"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/src
var rawAssets embed.FS

// NOTE: On Linux with older WebKitGTK, you may need the build tag:
//   go build -tags webkit2_41

func main() {
	// Strip the "frontend/src" prefix so files are served at root
	assets, err := fs.Sub(rawAssets, "frontend/src")
	if err != nil {
		log.Fatal("failed to create sub filesystem:", err)
	}

	app := &App{}
	downloadService := NewDownloadService()
	settingsService := &SettingsService{}
	licenseService := &LicenseService{}
	playerService := &PlayerService{}
	codecService := &CodecService{}
	torrentDlService := NewTorrentDownloadService(nil) // StreamService linked after ProServices
	depsService := NewDepsService()

	// Pro services: real implementations in pro build, stubs in free build.
	proServices := ProServices()

	// Combine all bound services.
	bindings := []interface{}{
		app,
		downloadService,
		settingsService,
		licenseService,
		playerService,
		codecService,
		depsService,
		torrentDlService,
	}
	bindings = append(bindings, proServices...)

	err = wails.Run(&options.App{
		Title:     "Yaria",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Frameless:   false,
		StartHidden: false,
		Linux: &linux.Options{
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
		},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			downloadService.startup(ctx)
			licenseService.startup(ctx)
			playerService.startup(ctx)
			codecService.startup(ctx)
			depsService.startup(ctx)
			// Link torrent download service to the streaming service for client reuse
			for _, svc := range proServices {
				if ss, ok := svc.(*StreamService); ok {
					torrentDlService.streamService = ss
					break
				}
			}
			torrentDlService.startup(ctx)
			ProStartup(ctx, proServices)
		},
		OnShutdown: func(ctx context.Context) {
			torrentDlService.shutdown()
			ProShutdown(ctx, proServices)
			app.shutdown(ctx)
		},
		Bind: bindings,
	})
	if err != nil {
		panic(err)
	}
}
