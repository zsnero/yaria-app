//go:build !pro

package main

// TMDBService is a stub for free builds.
type TMDBService struct{}

func NewTMDBService() *TMDBService { return &TMDBService{} }

func (t *TMDBService) SearchMulti(query string, page int) interface{} {
	return map[string]interface{}{"error": stubErr, "results": []interface{}{}}
}

func (t *TMDBService) GetTrending(mediaType string, page int) interface{} {
	return map[string]interface{}{"error": stubErr, "results": []interface{}{}}
}

func (t *TMDBService) GetMovieDetails(id int) interface{} {
	return map[string]interface{}{"error": stubErr}
}

func (t *TMDBService) GetTVDetails(id int) interface{} {
	return map[string]interface{}{"error": stubErr}
}

func (t *TMDBService) GetSeasonDetails(tvID, season int) interface{} {
	return map[string]interface{}{"error": stubErr}
}

func (t *TMDBService) RefreshClient(apiKey string) {}
