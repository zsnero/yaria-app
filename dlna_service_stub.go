//go:build !pro

package main

import "context"

type DLNAService struct{}

func NewDLNAService() *DLNAService                          { return &DLNAService{} }
func (d *DLNAService) startup(ctx context.Context)          {}
func (d *DLNAService) shutdown()                            {}
func (d *DLNAService) LinkMediaService(_ *MediaService)     {}
func (d *DLNAService) Start() map[string]interface{}        { return map[string]interface{}{"error": stubErr} }
func (d *DLNAService) Stop() map[string]interface{}         { return map[string]interface{}{"error": stubErr} }
func (d *DLNAService) Status() map[string]interface{}       { return map[string]interface{}{"running": false} }
