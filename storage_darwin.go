//go:build darwin

package main

import (
	"os"
	"path/filepath"
	"strings"
)

// mountedStorageDevices returns mounts under /Volumes (external HDDs, USB
// drives, DMGs, etc.).
func mountedStorageDevices() []map[string]interface{} {
	entries, err := os.ReadDir("/Volumes")
	if err != nil {
		return nil
	}
	var out []map[string]interface{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") || !e.IsDir() {
			continue
		}
		out = append(out, map[string]interface{}{
			"name":   e.Name(),
			"path":   filepath.Join("/Volumes", e.Name()),
			"device": "",
		})
	}
	return out
}
