//go:build linux

package main

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// mountedStorageDevices returns local mounted storage for the in-app file
// picker: removable drives (external HDD, USB, optical) plus manually mounted
// partitions. Uses /proc/self/mounts so it works without root or extra tools.
func mountedStorageDevices() []map[string]interface{} {
	home, _ := os.UserHomeDir()
	uid := os.Getuid()

	// Directories that typically hold removable / manual mounts.
	removableRoots := map[string]bool{
		"/mnt":   true,
		"/media": true,
	}
	for _, r := range []string{
		filepath.Join("/run/media", strconv.Itoa(uid)),
		filepath.Join("/run/media", filepath.Base(home)),
		filepath.Join("/media", strconv.Itoa(uid)),
		filepath.Join("/media", filepath.Base(home)),
	} {
		if r != "" && r != "/mnt" && r != "/media" {
			removableRoots[r] = true
		}
	}

	// Filesystems that almost always indicate removable media.
	removableFS := map[string]bool{
		"vfat":    true,
		"exfat":   true,
		"ntfs":    true,
		"ntfs3":   true,
		"fuseblk": true,
		"iso9660": true,
		"udf":     true,
	}

	// System partitions that must never be offered as selectable "devices".
	systemMounts := []string{
		"/", "/boot", "/boot/efi", "/efi", "/home", "/usr", "/var",
		"/etc", "/opt", "/srv", "/snap", "/tmp", "/root",
	}

	// Pseudo filesystems that are never real storage.
	excludedFS := map[string]bool{
		"proc": true, "sysfs": true, "devpts": true, "devtmpfs": true,
		"tmpfs": true, "ramfs": true, "overlay": true, "squashfs": true,
		"cgroup": true, "cgroup2": true, "pstore": true, "securityfs": true,
		"debugfs": true, "tracefs": true, "hugetlbfs": true, "mqueue": true,
		"binfmt_misc": true, "configfs": true, "fusectl": true, "rpc_pipefs": true,
		"autofs": true, "nsfs": true, "bpf": true, "swap": true,
	}

	isSystemMount := func(mp string) bool {
		for _, s := range systemMounts {
			if mp == s || strings.HasPrefix(mp, s+"/") {
				return true
			}
		}
		return false
	}

	// Loop devices (AppImage, snap), zram and fuse are not real selectable storage.
	isVirtualDevice := func(dev string) bool {
		return strings.Contains(dev, "/dev/loop") ||
			strings.Contains(dev, "/dev/zram") ||
			strings.Contains(dev, "/dev/fuse")
	}

	data, err := os.ReadFile("/proc/self/mounts")
	if err != nil {
		return nil
	}

	type mnt struct {
		device     string
		mountpoint string
		fs         string
	}

	var candidates []mnt
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		device := unescapeMountPath(f[0])
		mountpoint := unescapeMountPath(f[1])
		fs := f[2]
		if !strings.HasPrefix(mountpoint, "/") || !strings.HasPrefix(device, "/dev/") {
			continue
		}
		if excludedFS[fs] || isVirtualDevice(device) || isSystemMount(mountpoint) {
			continue
		}
		// Skip same-filesystem sub-mounts unless they are clearly removable.
		if !removableFS[fs] && !underAny(mountpoint, removableRoots) {
			continue
		}
		candidates = append(candidates, mnt{device, mountpoint, fs})
	}

	if len(candidates) == 0 {
		return nil
	}

	// Shallowest first so nested mounts are dropped in favor of their parent.
	sort.Slice(candidates, func(i, j int) bool {
		di := strings.Count(candidates[i].mountpoint, "/")
		dj := strings.Count(candidates[j].mountpoint, "/")
		if di != dj {
			return di < dj
		}
		return candidates[i].mountpoint < candidates[j].mountpoint
	})

	var seen []string
	var out []map[string]interface{}
	for _, m := range candidates {
		nested := false
		for _, mp := range seen {
			if strings.HasPrefix(m.mountpoint, mp+"/") {
				nested = true
				break
			}
		}
		if nested {
			continue
		}
		seen = append(seen, m.mountpoint)
		out = append(out, map[string]interface{}{
			"name":   diskLabel(m.device, m.mountpoint),
			"path":   m.mountpoint,
			"device": m.device,
		})
	}
	return out
}

func underAny(mountpoint string, roots map[string]bool) bool {
	for r := range roots {
		if mountpoint == r || strings.HasPrefix(mountpoint, r+"/") {
			return true
		}
	}
	return false
}

// diskLabel prefers the volume label (via /dev/disk/by-label), falling back to
// the mount point basename (e.g. "WDC 2TB" from /run/media/kaz/WDC 2TB).
func diskLabel(device, mountpoint string) string {
	if entries, err := os.ReadDir("/dev/disk/by-label"); err == nil {
		base := filepath.Base(device)
		for _, e := range entries {
			target, err := os.Readlink(filepath.Join("/dev/disk/by-label", e.Name()))
			if err == nil && filepath.Base(target) == base {
				return e.Name()
			}
		}
	}
	return filepath.Base(mountpoint)
}

// unescapeMountPath decodes octal escapes (e.g. \040 for space) used in
// /proc/self/mounts.
func unescapeMountPath(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) && isOctal(s[i+1]) && isOctal(s[i+2]) && isOctal(s[i+3]) {
			b := byte(0)
			for j := 1; j <= 3; j++ {
				b = b*8 + (s[i+j] - '0')
			}
			sb.WriteByte(b)
			i += 3
		} else {
			sb.WriteByte(s[i])
		}
	}
	return sb.String()
}

func isOctal(c byte) bool {
	return c >= '0' && c <= '7'
}
