//go:build !pro

package main

import "context"

// ProAvailable returns false in free builds.
func ProAvailable() bool { return false }

// ProServices returns stub Mantorex services in free builds.
// Each stub uses the same struct name as the real service so that
// Wails generates matching JS bindings (e.g. window.go.main.SearchService).
func ProServices() []interface{} {
	return []interface{}{
		&SearchService{},
		&StreamService{},
		&LibraryService{},
		&TMDBService{},
	}
}

// ProStartup is a no-op in free builds.
func ProStartup(ctx context.Context, services []interface{}) {}

// ProShutdown is a no-op in free builds.
func ProShutdown(ctx context.Context, services []interface{}) {}
