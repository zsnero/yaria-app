//go:build !pro

package main

import "context"

type MediaServer struct{}

func NewMediaServer() *MediaServer                                      { return &MediaServer{} }
func (s *MediaServer) startup(ctx context.Context)                      {}
func (s *MediaServer) shutdown()                                        {}
func (s *MediaServer) LinkMediaService(_ *MediaService)                 {}
func (s *MediaServer) Start() map[string]interface{}                    { return map[string]interface{}{"error": stubErr} }
func (s *MediaServer) Stop() map[string]interface{}                     { return map[string]interface{}{"error": stubErr} }
func (s *MediaServer) Status() map[string]interface{}                   { return map[string]interface{}{"running": false} }
func (s *MediaServer) SetPort(port int) map[string]interface{}          { return map[string]interface{}{"error": stubErr} }
func (s *MediaServer) SetPin(pin string) map[string]interface{}         { return map[string]interface{}{"error": stubErr} }
