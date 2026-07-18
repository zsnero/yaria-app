package main

import (
	"context"
	"time"

	"yaria/pkg/license"
)

// LicenseService provides license management methods to the frontend.
// Always included in both free and pro builds.
type LicenseService struct {
	ctx context.Context
}

func (l *LicenseService) startup(ctx context.Context) {
	l.ctx = ctx
}

// CheckLicense returns the current license status.
func (l *LicenseService) CheckLicense() map[string]interface{} {
	info := license.CheckLicense()
	return map[string]interface{}{
		"valid":         info.Valid,
		"plan":          info.Plan,
		"email":         info.Email,
		"device_id":     info.DeviceID,
		"device_name":   info.DeviceName,
		"key":           info.Key,
		"expires_at":    formatExpiry(info.ExpiresAt),
		"pro_available": ProAvailable(),
	}
}

func formatExpiry(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// ActivateKey activates a license key for this device.
func (l *LicenseService) ActivateKey(key string) map[string]interface{} {
	if key == "" {
		return map[string]interface{}{"error": "license key is required"}
	}

	info, err := license.ActivateKey(key)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"valid":       info.Valid,
		"plan":        info.Plan,
		"email":       info.Email,
		"device_id":   info.DeviceID,
		"device_name": info.DeviceName,
		"expires_at":  formatExpiry(info.ExpiresAt),
	}
}

// StartTrial starts a one-time 30-day Pro trial for this device.
func (l *LicenseService) StartTrial() map[string]interface{} {
	info, err := license.StartTrial()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"valid":       info.Valid,
		"plan":        info.Plan,
		"email":       info.Email,
		"device_id":   info.DeviceID,
		"device_name": info.DeviceName,
		"expires_at":  formatExpiry(info.ExpiresAt),
	}
}

// Deactivate removes the stored license.
func (l *LicenseService) Deactivate() map[string]interface{} {
	if err := license.Deactivate(); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "deactivated"}
}

// GetDeviceInfo returns the current device ID and summary.
func (l *LicenseService) GetDeviceInfo() map[string]interface{} {
	id, summary := license.GetDeviceInfo()
	return map[string]interface{}{
		"device_id":   id,
		"device_name": summary,
	}
}

// IsPro returns whether this device has an active pro license AND
// the pro module is compiled in.
func (l *LicenseService) IsPro() bool {
	return ProAvailable() && license.IsPro()
}
