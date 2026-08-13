//go:build !linux && !windows && !darwin

package main

// mountedStorageDevices is unsupported on this platform.
func mountedStorageDevices() []map[string]interface{} {
	return nil
}
