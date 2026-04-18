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

func (l *LibraryService) UpdateProgress(id string, progress interface{}) map[string]interface{} {
	return map[string]interface{}{"error": stubErr}
}
