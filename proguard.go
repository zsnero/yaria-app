package main

import (
	"sync"
	"time"

	"yaria/pkg/license"
)

var (
	proMu    sync.Mutex
	proCached bool
	proLast  time.Time
)

// proAllowed reports whether Pro is compiled in AND licensed for this device.
// Results are cached for a short TTL so hot calls (GetStatus, Pause, ...)
// don't hit the filesystem or network on every invocation.
func proAllowed() bool {
	proMu.Lock()
	defer proMu.Unlock()
	if time.Since(proLast) > 30*time.Second {
		proCached = ProAvailable() && license.IsPro()
		proLast = time.Now()
	}
	return proCached
}

// proInvalidate clears the cached result so the next proAllowed() call
// re-evaluates immediately. Called after activation / trial start / deactivate.
func proInvalidate() {
	proMu.Lock()
	proLast = time.Time{}
	proMu.Unlock()
}

// denyResult is the uniform error returned by gated methods when Pro is
// not active. Frontends surface result.error to the user.
func denyResult() map[string]interface{} {
	return map[string]interface{}{"error": "Pro license required"}
}
