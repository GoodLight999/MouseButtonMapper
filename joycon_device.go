package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	defaultJoyConReconnectMs = 1000
	minJoyConReconnectMs     = 250
	maxJoyConReconnectMs     = 10000
	joyConOutputReportLength = 49
	joyConInputReportLength  = 64
)

type JoyConReconnectConfig struct {
	Enabled    bool `json:"Enabled,omitempty"`
	IntervalMs int  `json:"IntervalMs,omitempty"`
}

type JoyConProfileConfig struct {
	Enabled         bool                  `json:"Enabled,omitempty"`
	PreferredDevice string                `json:"PreferredDevice,omitempty"`
	Stick           JoyConStickConfig     `json:"Stick,omitempty"`
	Reconnect       JoyConReconnectConfig `json:"Reconnect,omitempty"`
}

func defaultJoyConProfileConfig() JoyConProfileConfig {
	return JoyConProfileConfig{
		Stick: defaultJoyConStickConfig(),
		Reconnect: JoyConReconnectConfig{
			Enabled:    true,
			IntervalMs: defaultJoyConReconnectMs,
		},
	}
}

func normalizeJoyConProfileConfig(config JoyConProfileConfig) JoyConProfileConfig {
	config.PreferredDevice = strings.TrimSpace(config.PreferredDevice)
	config.Stick = normalizeJoyConStickConfig(config.Stick)
	if config.Reconnect.IntervalMs == 0 {
		config.Reconnect.IntervalMs = defaultJoyConReconnectMs
	}
	if config.Reconnect.IntervalMs < minJoyConReconnectMs {
		config.Reconnect.IntervalMs = minJoyConReconnectMs
	}
	if config.Reconnect.IntervalMs > maxJoyConReconnectMs {
		config.Reconnect.IntervalMs = maxJoyConReconnectMs
	}
	return config
}

type JoyConDeviceInfo struct {
	Path        string `json:"-"`
	Fingerprint string `json:"Fingerprint"`
	VendorID    uint16 `json:"VendorId"`
	ProductID   uint16 `json:"ProductId"`
	Version     uint16 `json:"Version,omitempty"`
	Product     string `json:"Product,omitempty"`
	Serial      string `json:"Serial,omitempty"`
}

func (d JoyConDeviceInfo) IsLeftJoyCon() bool {
	return d.VendorID == joyConNintendoVendorID && d.ProductID == joyConLeftProductID
}

func (d JoyConDeviceInfo) StableID() string {
	if serial := strings.TrimSpace(d.Serial); serial != "" {
		return "serial:" + strings.ToLower(serial)
	}
	if fingerprint := strings.TrimSpace(d.Fingerprint); fingerprint != "" {
		return "path:" + strings.ToLower(fingerprint)
	}
	return fmt.Sprintf("vid:%04x:pid:%04x", d.VendorID, d.ProductID)
}

func fingerprintJoyConDevicePath(path string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(path))))
	return hex.EncodeToString(sum[:8])
}

func chooseJoyConDevice(devices []JoyConDeviceInfo, preferred string) (JoyConDeviceInfo, bool) {
	preferred = strings.ToLower(strings.TrimSpace(preferred))
	if preferred != "" {
		for _, device := range devices {
			if !device.IsLeftJoyCon() {
				continue
			}
			if strings.EqualFold(device.StableID(), preferred) ||
				strings.EqualFold(device.Serial, preferred) ||
				strings.EqualFold(device.Fingerprint, preferred) {
				return device, true
			}
		}
	}
	for _, device := range devices {
		if device.IsLeftJoyCon() {
			return device, true
		}
	}
	return JoyConDeviceInfo{}, false
}

type JoyConConnectionStatus struct {
	Connected      bool             `json:"Connected"`
	Device         JoyConDeviceInfo `json:"Device"`
	BatteryPercent int              `json:"BatteryPercent"`
	Charging       bool             `json:"Charging"`
	LastInput      string           `json:"LastInput,omitempty"`
	LastReportAt   time.Time        `json:"LastReportAt,omitempty"`
	LastError      string           `json:"LastError,omitempty"`
	ReconnectCount uint64           `json:"ReconnectCount"`
}

func buildJoyConSubcommandReport(packet byte, subcommand byte, data []byte) ([]byte, error) {
	if len(data) > joyConOutputReportLength-11 {
		return nil, fmt.Errorf("Joy-Con subcommand payload is too large: %d", len(data))
	}
	report := make([]byte, joyConOutputReportLength)
	report[0] = 0x01
	report[1] = packet & 0x0f
	copy(report[2:10], []byte{0x00, 0x01, 0x40, 0x40, 0x00, 0x01, 0x40, 0x40})
	report[10] = subcommand
	copy(report[11:], data)
	return report, nil
}

func buildJoyConFullReportModeCommand(packet byte) ([]byte, error) {
	return buildJoyConSubcommandReport(packet, 0x03, []byte{joyConReportFull})
}
