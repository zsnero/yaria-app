//go:build !pro

package main

import "context"

type LocalMedia struct{}
type TVShowGroup struct{}
type SubtitleTrack struct{}
type MediaVersion struct{}

type MediaService struct{}

func NewMediaService() *MediaService                                          { return &MediaService{} }
func (m *MediaService) startup(ctx context.Context)                           {}
func (m *MediaService) shutdown()                                             {}
func (m *MediaService) GetMediaFolders() map[string]interface{}               { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) AddMediaFolder(path, mediaType string) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) RemoveMediaFolder(path, mediaType string) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) BrowseMediaFolder() string                             { return "" }
func (m *MediaService) ScanAll() map[string]interface{}                       { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) ScanStatus() map[string]interface{}                    { return map[string]interface{}{"scanning": false} }
func (m *MediaService) GetAllMedia(sortBy string) []LocalMedia                { return nil }
func (m *MediaService) GetMovies(sortBy string) []LocalMedia                  { return nil }
func (m *MediaService) GetTVShows() []TVShowGroup                             { return nil }
func (m *MediaService) GetRecentlyAdded(days int) []LocalMedia                { return nil }
func (m *MediaService) SearchMedia(query string) []LocalMedia                 { return nil }
func (m *MediaService) GetMediaDetails(id string) *LocalMedia                 { return nil }
func (m *MediaService) GetThumbnailData(id string) string                     { return "" }
func (m *MediaService) PlayMedia(id string) map[string]interface{}            { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) PlayMediaExternal(id string) map[string]interface{}    { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) GetLocalStreamURL() string                             { return "" }
func (m *MediaService) MatchTMDB(mediaID string, tmdbID int, mediaType string) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) GetUnmatched() []LocalMedia                            { return nil }
func (m *MediaService) ClearAllMedia() map[string]interface{}                 { return map[string]interface{}{"error": stubErr} }
// Feature 4: Watch state
func (m *MediaService) MarkWatched(id string) map[string]interface{}          { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) MarkUnwatched(id string) map[string]interface{}        { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) UpdateResumePosition(id string, pos int) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) GetContinueWatching() []LocalMedia                     { return nil }
func (m *MediaService) GetWatchHistory() []LocalMedia                         { return nil }
// Feature 7: Music
func (m *MediaService) ScanMusic() map[string]interface{}                     { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) GetMusicLibrary() []map[string]interface{}             { return nil }
// Feature 11: HW accel
func (m *MediaService) DetectHWAccel() map[string]interface{}                 { return map[string]interface{}{"available": false} }
// Feature 12: NFO writing
func (m *MediaService) WriteNFO(id string) map[string]interface{}             { return map[string]interface{}{"error": stubErr} }
// Feature 8: Profiles
type ServerProfile struct{}
type LyricLine struct{}
func (m *MediaService) GetProfiles() []ServerProfile                          { return nil }
func (m *MediaService) AddProfile(name, pin, avatar string) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (m *MediaService) RemoveProfile(id string) map[string]interface{}        { return map[string]interface{}{"error": stubErr} }
// Feature 14: Lyrics
func (m *MediaService) GetLyrics(id string) map[string]interface{}            { return map[string]interface{}{"error": stubErr} }
// Counts
func (m *MediaService) GetMediaCount() map[string]interface{}                 { return map[string]interface{}{"total": 0} }
