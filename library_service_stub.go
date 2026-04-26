//go:build !pro

package main

// LibraryService is a stub for free builds.
type LibraryService struct{}

func NewLibraryService() *LibraryService { return &LibraryService{} }

func (l *LibraryService) GetAll() map[string]interface{} {
	return map[string]interface{}{"error": stubErr, "items": []interface{}{}}
}

func (l *LibraryService) Add(item interface{}) interface{} {
	return map[string]interface{}{"error": stubErr}
}

func (l *LibraryService) Remove(id string) map[string]interface{} {
	return map[string]interface{}{"error": stubErr}
}

func (l *LibraryService) UpdateProgress(id string, progress map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"error": stubErr}
}

func (l *LibraryService) FindByTMDBID(tmdbID int, mediaType string) interface{} {
	return nil
}

func (l *LibraryService) FindByTitle(title, mediaType, year string) interface{} {
	return nil
}

func (l *LibraryService) UpdateEpisodeProgress(id string, season, episode int, title string, timeSeconds, durationSeconds int) map[string]interface{} {
	return map[string]interface{}{"error": stubErr}
}

func (l *LibraryService) MarkEpisodeWatched(id string, season, episode int) map[string]interface{} {
	return map[string]interface{}{"error": stubErr}
}

func (l *LibraryService) SetLastKnownEpisode(id string, season, episode int, airDate string) map[string]interface{} {
	return map[string]interface{}{"error": stubErr}
}
