//go:build windows

package main

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

func diskFreeBytes(path string) (uint64, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}
	// Use the volume root (e.g. C:\)
	vol := filepath.VolumeName(path) + `\`
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")
	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	p, err := syscall.UTF16PtrFromString(vol)
	if err != nil {
		return 0, err
	}
	r1, _, e1 := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if r1 == 0 {
		if e1 != nil {
			return 0, e1
		}
		return 0, syscall.EINVAL
	}
	return freeBytesAvailable, nil
}
