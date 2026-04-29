//go:build !pro

package main

import "context"

type TorrentDownloadService struct {
	streamService *StreamService // unused in free build
}

func NewTorrentDownloadService(_ *StreamService) *TorrentDownloadService { return &TorrentDownloadService{} }
func (s *TorrentDownloadService) startup(ctx context.Context) {}
func (s *TorrentDownloadService) shutdown()                    {}
func (s *TorrentDownloadService) AddDownload(magnet, title, dir string) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (s *TorrentDownloadService) ListDownloads() []map[string]interface{} { return nil }
func (s *TorrentDownloadService) RemoveDownload(id string) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (s *TorrentDownloadService) DeleteDownload(id string) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (s *TorrentDownloadService) CancelDownload(id string) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (s *TorrentDownloadService) PauseDownload(id string) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (s *TorrentDownloadService) ResumeDownload(id string) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (s *TorrentDownloadService) GetDownloadDir() string { return "" }
func (s *TorrentDownloadService) SelectDownloadDir() string { return "" }
