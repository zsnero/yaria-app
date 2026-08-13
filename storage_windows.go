//go:build windows

package main

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// mountedStorageDevices returns all accessible logical drives (fixed, removable,
// optical) with their volume labels. Network drives are skipped.
func mountedStorageDevices() []map[string]interface{} {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getLogicalDrives := kernel32.NewProc("GetLogicalDrives")
	getDriveType := kernel32.NewProc("GetDriveTypeW")
	getVolumeInfo := kernel32.NewProc("GetVolumeInformationW")

	drivesMask, _, _ := getLogicalDrives.Call()
	if drivesMask == 0 {
		return nil
	}

	var out []map[string]interface{}
	for i := 0; i < 26; i++ {
		if drivesMask&(1<<uint(i)) == 0 {
			continue
		}
		letter := string(rune('A' + i))
		root := letter + `:\`
		rootPtr, err := syscall.UTF16PtrFromString(root)
		if err != nil {
			continue
		}
		dt, _, _ := getDriveType.Call(uintptr(unsafe.Pointer(rootPtr)))
		switch uint(dt) {
		case 0, 1: // unknown / DRIVE_NO_ROOT_DIR
			continue
		case 4: // DRIVE_REMOTE - network share, not local storage
			continue
		}
		label := windowsVolumeLabel(getVolumeInfo, root)
		name := strings.TrimSpace(label)
		if name == "" {
			name = strings.TrimSuffix(root, `\`) + " Drive"
		}
		out = append(out, map[string]interface{}{
			"name":   name,
			"path":   filepath.Clean(root),
			"device": root,
		})
	}
	return out
}

func windowsVolumeLabel(getVolumeInfo *syscall.LazyProc, root string) string {
	rootPtr, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return ""
	}
	buf := make([]uint16, 256)
	var fsName [64]uint16
	r, _, _ := getVolumeInfo.Call(
		uintptr(unsafe.Pointer(rootPtr)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0, // serial number
		0, // max component length
		0, // filesystem flags
		uintptr(unsafe.Pointer(&fsName[0])),
		uintptr(len(fsName)),
	)
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
