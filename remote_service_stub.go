//go:build !pro

package main

import "context"

type RemoteSource struct{}
type DiscoveredDevice struct{}
type RemoteFile struct{}

type RemoteService struct{}

func NewRemoteService() *RemoteService                                           { return &RemoteService{} }
func (r *RemoteService) LinkMediaDB(_ *MediaService)                             {}
func (r *RemoteService) startup(ctx context.Context)                            {}
func (r *RemoteService) shutdown()                                              {}
func (r *RemoteService) GetSources() []RemoteSource                             { return nil }
func (r *RemoteService) AddSource(sourceJSON string) map[string]interface{}     { return map[string]interface{}{"error": stubErr} }
func (r *RemoteService) RemoveSource(id string) map[string]interface{}          { return map[string]interface{}{"error": stubErr} }
func (r *RemoteService) Connect(id, password string) map[string]interface{}     { return map[string]interface{}{"error": stubErr} }
func (r *RemoteService) Disconnect(id string) map[string]interface{}            { return map[string]interface{}{"error": stubErr} }
func (r *RemoteService) IsConnected(id string) bool                             { return false }
func (r *RemoteService) ConnectedSources() []string                             { return nil }
func (r *RemoteService) BrowseRemote(sourceID, path string) []RemoteFile        { return nil }
func (r *RemoteService) GetRemoteStreamURL(sourceID, filePath string) string    { return "" }
func (r *RemoteService) DiscoverDevices() []DiscoveredDevice                    { return nil }
func (r *RemoteService) QuickConnect(host string, port int, connType string) map[string]interface{} { return map[string]interface{}{"error": stubErr} }
func (r *RemoteService) SelectKeyFile() string                                  { return "" }
