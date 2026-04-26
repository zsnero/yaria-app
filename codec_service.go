package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// CodecService checks and installs GStreamer codec dependencies.
type CodecService struct {
	ctx context.Context
}

func (c *CodecService) startup(ctx context.Context) {
	c.ctx = ctx
}

type codecStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Package   string `json:"package"`
}

// CheckCodecs checks which GStreamer plugin packages are installed.
func (c *CodecService) CheckCodecs() map[string]interface{} {
	if runtime.GOOS != "linux" {
		return map[string]interface{}{"supported": false, "message": "codec check only available on Linux"}
	}

	required := []struct {
		name    string
		testLib string // GStreamer element to check
		pkg     map[string]string // distro -> package name
	}{
		{
			"GStreamer Good (base codecs)",
			"vp8dec",
			map[string]string{"arch": "gst-plugins-good", "debian": "gstreamer1.0-plugins-good", "fedora": "gstreamer1-plugins-good"},
		},
		{
			"GStreamer LibAV (FFmpeg codecs)",
			"avdec_h264",
			map[string]string{"arch": "gst-libav", "debian": "gstreamer1.0-libav", "fedora": "gstreamer1-libav"},
		},
		{
			"GStreamer Ugly (MPEG/AC3/DTS)",
			"x264enc",
			map[string]string{"arch": "gst-plugins-ugly", "debian": "gstreamer1.0-plugins-ugly", "fedora": "gstreamer1-plugins-ugly"},
		},
		{
			"GStreamer Bad (HEVC/AV1/AAC)",
			"av1dec",
			map[string]string{"arch": "gst-plugins-bad", "debian": "gstreamer1.0-plugins-bad", "fedora": "gstreamer1-plugins-bad"},
		},
	}

	distro := detectDistro()
	var codecs []codecStatus
	allInstalled := true
	var missingPkgs []string

	for _, r := range required {
		installed := checkGstElement(r.testLib)
		pkg := r.pkg[distro]
		if pkg == "" {
			pkg = r.pkg["debian"] // fallback
		}
		codecs = append(codecs, codecStatus{
			Name:      r.name,
			Installed: installed,
			Package:   pkg,
		})
		if !installed {
			allInstalled = false
			missingPkgs = append(missingPkgs, pkg)
		}
	}

	// Build install command
	var installCmd string
	if len(missingPkgs) > 0 {
		switch distro {
		case "arch":
			installCmd = "sudo pacman -S --noconfirm " + strings.Join(missingPkgs, " ")
		case "debian":
			installCmd = "sudo apt install -y " + strings.Join(missingPkgs, " ")
		case "fedora":
			installCmd = "sudo dnf install -y " + strings.Join(missingPkgs, " ")
		default:
			installCmd = "# Install: " + strings.Join(missingPkgs, " ")
		}
	}

	return map[string]interface{}{
		"codecs":        codecs,
		"all_installed": allInstalled,
		"distro":        distro,
		"install_cmd":   installCmd,
		"missing":       missingPkgs,
	}
}

// InstallCodecs attempts to install missing GStreamer packages.
// Returns the command output. Requires pkexec for privilege escalation.
func (c *CodecService) InstallCodecs() map[string]interface{} {
	if runtime.GOOS != "linux" {
		return map[string]interface{}{"error": "only supported on Linux"}
	}

	distro := detectDistro()
	check := c.CheckCodecs()
	missing, ok := check["missing"].([]string)
	if !ok || len(missing) == 0 {
		return map[string]interface{}{"status": "all codecs already installed"}
	}

	// Use pkexec for graphical sudo prompt
	var cmd *exec.Cmd
	switch distro {
	case "arch":
		args := append([]string{"pacman", "-S", "--noconfirm"}, missing...)
		cmd = exec.Command("pkexec", args...)
	case "debian":
		args := append([]string{"apt", "install", "-y"}, missing...)
		cmd = exec.Command("pkexec", args...)
	case "fedora":
		args := append([]string{"dnf", "install", "-y"}, missing...)
		cmd = exec.Command("pkexec", args...)
	default:
		return map[string]interface{}{"error": "unsupported distro: " + distro, "install_cmd": check["install_cmd"]}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return map[string]interface{}{
			"error":  fmt.Sprintf("installation failed: %v", err),
			"output": string(output),
		}
	}

	return map[string]interface{}{
		"status": "installed",
		"output": string(output),
	}
}

// checkGstElement tests if a GStreamer element exists using gst-inspect-1.0
func checkGstElement(element string) bool {
	cmd := exec.Command("gst-inspect-1.0", element)
	return cmd.Run() == nil
}

// detectDistro determines the Linux distribution family
func detectDistro() string {
	// Check package managers
	if _, err := exec.LookPath("pacman"); err == nil {
		return "arch"
	}
	if _, err := exec.LookPath("apt"); err == nil {
		return "debian"
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		return "fedora"
	}
	return "unknown"
}
