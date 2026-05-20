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

func (s *StreamService) PauseStream() map[string]interface{} {
	return map[string]interface{}{"error": stubErr}
}
func (s *StreamService) ResumeStream() map[string]interface{} {
	return map[string]interface{}{"error": stubErr}
}

func (s *StreamService) GetStreamURL() string                                    { return "" }
func (s *StreamService) ListSubtitleFiles() []map[string]interface{}              { return nil }
func (s *StreamService) GetStreamDuration() float64                              { return 0 }
func (s *StreamService) GetTranscodeURL() string                                 { return "" }
func (s *StreamService) GetHLSURL() string                                       { return "" }
func (s *StreamService) IsHLSActive() bool                                       { return false }
func (s *StreamService) GetVODURL() string                                       { return "" }
func (s *StreamService) IsVODActive() bool                                       { return false }
func (s *StreamService) PrepareFileVOD(filePath string) map[string]interface{}   { return map[string]interface{}{"error": stubErr} }
func (s *StreamService) PrepareStream() map[string]interface{}                   { return map[string]interface{}{"mode": "direct"} }
func (s *StreamService) ListFiles() []map[string]interface{}        { return nil }
func (s *StreamService) SelectFile(index int) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
