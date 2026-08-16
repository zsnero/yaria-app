package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"yaria/pkg/appconfig"
)

// mpvUserConfigDir returns the standard user mpv config directory if it exists.
func mpvUserConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	var dir string
	switch runtime.GOOS {
	case "windows":
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			dir = filepath.Join(appdata, "mpv")
		} else {
			dir = filepath.Join(home, "AppData", "Roaming", "mpv")
		}
	case "darwin":
		dir = filepath.Join(home, ".config", "mpv")
	default:
		dir = filepath.Join(home, ".config", "mpv")
	}
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		return dir
	}
	return ""
}

// mpvApplyPlayerSettings applies curated Settings → Player options via setOpt
// (libmpv set_option_string or equivalent). Call before mpv_initialize.
// Embed-critical options (osc, input, keep-open, idle, volume-max) stay forced.
func mpvApplyPlayerSettings(setOpt func(key, val string)) {
	p := appconfig.GetPlayerSettings()

	// Always force embed-safe baseline (same as previous hardcoded defaults)
	setOpt("vo", "gpu")
	setOpt("keep-open", "yes")
	setOpt("idle", "yes")
	setOpt("osc", "no")
	setOpt("input-default-bindings", "no")
	setOpt("input-vo-keyboard", "no")
	setOpt("cursor-autohide", "always")
	setOpt("volume-max", "500")

	// Hardware decode
	hw := p.Hwdec
	if hw == "" {
		hw = "auto-safe"
	}
	setOpt("hwdec", hw)

	// Stream/torrent cache profile
	switch p.Cache {
	case "low":
		setOpt("cache", "yes")
		setOpt("demuxer-max-bytes", "32MiB")
		setOpt("demuxer-readahead-secs", "5")
		setOpt("cache-secs", "10")
	case "high":
		setOpt("cache", "yes")
		setOpt("demuxer-max-bytes", "256MiB")
		setOpt("demuxer-readahead-secs", "30")
		setOpt("cache-secs", "30")
	default: // normal — close to mpv defaults with a modest floor
		setOpt("cache", "yes")
		setOpt("demuxer-max-bytes", "96MiB")
		setOpt("demuxer-readahead-secs", "15")
		setOpt("cache-secs", "10")
	}

	if p.HqScale {
		setOpt("scale", "ewa_lanczossharp")
		setOpt("cscale", "ewa_lanczossharp")
		setOpt("dscale", "mitchell")
	}

	if p.Deinterlace {
		setOpt("deinterlace", "yes")
		setOpt("vf", "yadif")
	} else {
		setOpt("deinterlace", "no")
	}

	if p.LoadUserConfig {
		if dir := mpvUserConfigDir(); dir != "" {
			setOpt("config", "yes")
			setOpt("config-dir", dir)
		} else {
			// No dir — still allow default mpv config search
			setOpt("config", "yes")
		}
	} else {
		setOpt("config", "no")
		setOpt("load-scripts", "no")
	}
}

// mpvWindowsCLIFlags returns extra CLI flags for portable mpv.exe (after embed flags).
func mpvWindowsCLIFlags() []string {
	p := appconfig.GetPlayerSettings()
	var args []string

	hw := p.Hwdec
	if hw == "" {
		hw = "auto-safe"
	}
	args = append(args, "--hwdec="+hw)

	switch p.Cache {
	case "low":
		args = append(args,
			"--cache=yes",
			"--demuxer-max-bytes=32MiB",
			"--demuxer-readahead-secs=5",
			"--cache-secs=10",
		)
	case "high":
		args = append(args,
			"--cache=yes",
			"--demuxer-max-bytes=256MiB",
			"--demuxer-readahead-secs=30",
			"--cache-secs=60",
		)
	default:
		args = append(args,
			"--cache=yes",
			"--demuxer-max-bytes=96MiB",
			"--demuxer-readahead-secs=15",
			"--cache-secs=30",
		)
	}

	if p.HqScale {
		args = append(args,
			"--scale=ewa_lanczossharp",
			"--cscale=ewa_lanczossharp",
			"--dscale=mitchell",
		)
	}
	if p.Deinterlace {
		args = append(args, "--deinterlace=yes", "--vf=yadif")
	} else {
		args = append(args, "--deinterlace=no")
	}

	if p.LoadUserConfig {
		if dir := mpvUserConfigDir(); dir != "" {
			args = append(args, "--config-dir="+dir)
		}
	} else {
		args = append(args, "--no-config", "--load-scripts=no")
	}

	return args
}

// mpvPlayerSettingsSummary is a short debug string for logs/UI.
func mpvPlayerSettingsSummary() string {
	p := appconfig.GetPlayerSettings()
	return fmt.Sprintf("hwdec=%s cache=%s hq=%v deint=%v user_cfg=%v",
		p.Hwdec, p.Cache, p.HqScale, p.Deinterlace, p.LoadUserConfig)
}
