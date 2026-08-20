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
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var rawAssets embed.FS

// NOTE: On Linux with older WebKitGTK, you may need the build tag:
//   go build -tags webkit2_41

func main() {
	// libmpv embedding requires an X11 client window; on Wayland sessions this
	// re-execs with GDK_BACKEND=x11 (no-op otherwise). Must run before wails
	// initializes any GUI.
	ensureX11ForMpv()

	// Strip the "frontend/dist" prefix so files are served at root
	assets, err := fs.Sub(rawAssets, "frontend/dist")
	if err != nil {
		log.Fatal("failed to create sub filesystem:", err)
	}

	app := &App{}
	downloadService := NewDownloadService()
	settingsService := &SettingsService{}
	licenseService := &LicenseService{}
	playerService := &PlayerService{}
	mpvService := NewMpvService()
	codecService := &CodecService{}
	torrentDlService := NewTorrentDownloadService(nil) // StreamService linked after ProServices
	depsService := NewDepsService()
	mediaService := NewMediaService()
	remoteService := NewRemoteService()
	mediaServer := NewMediaServer()
	updaterService := NewUpdaterService()
	dlnaService := NewDLNAService()
	extensionBridge := NewExtensionBridge()
	extensionBridge.LinkDownloadService(downloadService)

	// Pro services: real implementations in pro build, stubs in free build.
	proServices := ProServices()

	// Combine all bound services.
	bindings := []interface{}{
		app,
		downloadService,
		settingsService,
		licenseService,
		playerService,
		mpvService,
		codecService,
		depsService,
		torrentDlService,
		mediaService,
		remoteService,
		mediaServer,
		dlnaService,
		updaterService,
		extensionBridge,
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
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "Yaria",
				Message: "Video & Audio Downloader",
			},
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                  false,
			DisableFramelessWindowDecorations:  false,
			WebviewUserDataPath:                "",
			WebviewBrowserPath:                 "",
		},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			downloadService.startup(ctx)
			licenseService.startup(ctx)
			playerService.startup(ctx)
			mpvService.startup(ctx)
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
			mediaService.startup(ctx)
			remoteService.LinkMediaDB(mediaService)
			remoteService.startup(ctx)
			mediaServer.LinkMediaService(mediaService)
			mediaServer.startup(ctx)
			dlnaService.LinkMediaService(mediaService)
			dlnaService.startup(ctx)
			updaterService.startup(ctx)
			extensionBridge.startup(ctx)
			ProStartup(ctx, proServices)
		},
		OnShutdown: func(ctx context.Context) {
			mpvService.shutdown()
			torrentDlService.shutdown()
			mediaService.shutdown()
			remoteService.shutdown()
			mediaServer.shutdown()
			dlnaService.shutdown()
			extensionBridge.shutdown()
			ProShutdown(ctx, proServices)
			app.shutdown(ctx)
		},
		Bind: bindings,
	})
	if err != nil {
		panic(err)
	}
}
