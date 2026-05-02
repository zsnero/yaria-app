//go:build !pro

package main

const stubErr = "Mantorex is not available in this build. Get Yaria Pro at yaria.live"

// SearchService is a stub for free builds.
type SearchService struct{}

func NewSearchService() *SearchService { return &SearchService{} }

func (s *SearchService) RefreshMeta(tmdbKey string) {}

func (s *SearchService) SearchTorrents(query, category, sortBy, filterTitle string) map[string]interface{} {
	return map[string]interface{}{"error": stubErr, "results": []interface{}{}, "count": 0}
}

func (s *SearchService) MetaSearch(query string) map[string]interface{} {
	return map[string]interface{}{"error": stubErr, "results": []interface{}{}, "count": 0}
}

func (s *SearchService) MetaTrending(mediaType string) map[string]interface{} {
	return map[string]interface{}{"error": stubErr, "results": []interface{}{}, "count": 0}
}
