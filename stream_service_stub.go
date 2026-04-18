//go:build !pro

package main

import "context"

// StreamService is a stub for free builds.
type StreamService struct{}

func (s *StreamService) startup(ctx context.Context) {}
func (s *StreamService) shutdown(ctx context.Context) {}

func (s *StreamService) StartStream(magnet string) map[string]interface{} {
	return map[string]interface{}{"error": stubErr}
}

func (s *StreamService) GetStatus() map[string]interface{} {
	return map[string]interface{}{"error": stubErr, "state": "idle", "active": false}
}

func (s *StreamService) StopStream() map[string]interface{} {
	return map[string]interface{}{"error": stubErr}
}

func (s *StreamService) GetStreamURL() string    { return "" }
func (s *StreamService) GetTranscodeURL() string { return "" }
