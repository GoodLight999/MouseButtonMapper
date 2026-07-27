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
	joyConProProductID       = 0x2009
	joyConTypeLeft           = 0x01

	hidUsagePageGenericDesktop = 0x01
	hidUsageJoystick           = 0x04
	hidUsageGamePad            = 0x05
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
	Path               string `json:"-"`
	Fingerprint        string `json:"Fingerprint"`
	VendorID           uint16 `json:"VendorId"`
	ProductID          uint16 `json:"ProductId"`
	Version            uint16 `json:"Version,omitempty"`
	Product            string `json:"Product,omitempty"`
	Serial             string `json:"Serial,omitempty"`
	ControllerType     uint8  `json:"ControllerType,omitempty"`
	UsagePage          uint16 `json:"UsagePage,omitempty"`
	Usage              uint16 `json:"Usage,omitempty"`
	InputReportLength  uint16 `json:"InputReportLength,omitempty"`
	OutputReportLength uint16 `json:"OutputReportLength,omitempty"`
	InputOnly          bool   `json:"InputOnly,omitempty"`
	ForcedCompatible   bool   `json:"ForcedCompatible,omitempty"`
}

func (d JoyConDeviceInfo) IsLeftJoyCon() bool {
	return d.ControllerType == joyConTypeLeft ||
		(d.VendorID == joyConNintendoVendorID && d.ProductID == joyConLeftProductID) ||
		isExplicitLeftJoyConProduct(d.Product)
}

func (d JoyConDeviceInfo) MightBeLeftJoyCon() bool {
	return d.IsLeftJoyCon() ||
		(d.VendorID == joyConNintendoVendorID && d.ProductID == joyConProProductID)
}

func (d JoyConDeviceInfo) IsGameControllerCollection() bool {
	return d.UsagePage == hidUsagePageGenericDesktop &&
		(d.Usage == hidUsageJoystick || d.Usage == hidUsageGamePad)
}

func (d JoyConDeviceInfo) CanOpenAsCompatibleJoyCon() bool {
	return d.IsLeftJoyCon() || d.ForcedCompatible
}

func shouldOpenJoyConInputOnly(device JoyConDeviceInfo) bool {
	// Zero report lengths mean HID caps were unavailable, not that the device
	// cannot accept output. Keep known Joy-Con devices on the normal R/W path.
	return device.InputOnly || device.ForcedCompatible ||
		(device.OutputReportLength > 0 && device.OutputReportLength < joyConOutputReportLength) ||
		(device.InputReportLength > 0 && device.InputReportLength <= 8)
}

func isExplicitLeftJoyConProduct(product string) bool {
	name := strings.ToLower(strings.NewReplacer(" ", "", "-", "", "_", "").Replace(product))
	return strings.Contains(name, "joycon(l)") ||
		strings.Contains(name, "joyconleft") ||
		strings.Contains(name, "leftjoycon")
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

func (d JoyConDeviceInfo) DisplayName() string {
	name := strings.TrimSpace(d.Product)
	if name == "" {
		name = "HID game controller"
	}
	return fmt.Sprintf("%s (VID %04x / PID %04x / ID %s)", name, d.VendorID, d.ProductID, d.StableID())
}

func fingerprintJoyConDevicePath(path string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(path))))
	return hex.EncodeToString(sum[:8])
}

func matchesPreferredJoyConDevice(device JoyConDeviceInfo, preferred string) bool {
	preferred = strings.TrimSpace(preferred)
	return strings.EqualFold(device.StableID(), preferred) ||
		strings.EqualFold(device.Serial, preferred) ||
		strings.EqualFold(device.Fingerprint, preferred)
}

func chooseJoyConDevice(devices []JoyConDeviceInfo, preferred string) (JoyConDeviceInfo, bool) {
	preferred = strings.TrimSpace(preferred)
	if preferred != "" {
		for _, device := range devices {
			if !matchesPreferredJoyConDevice(device, preferred) {
				continue
			}
			if !device.IsGameControllerCollection() && !device.MightBeLeftJoyCon() {
				continue
			}
			device.ForcedCompatible = !device.IsLeftJoyCon()
			return device, true
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
	Connected      bool               `json:"Connected"`
	Device         JoyConDeviceInfo   `json:"Device"`
	Candidates     []JoyConDeviceInfo `json:"Candidates,omitempty"`
	BatteryPercent int                `json:"BatteryPercent"`
	Charging       bool               `json:"Charging"`
	LastInput      string             `json:"LastInput,omitempty"`
	LastReportAt   time.Time          `json:"LastReportAt,omitempty"`
	LastError      string             `json:"LastError,omitempty"`
	ReconnectCount uint64             `json:"ReconnectCount"`
	RawStickX      uint16             `json:"RawStickX"`
	RawStickY      uint16             `json:"RawStickY"`
	StickX         float64            `json:"StickX"`
	StickY         float64            `json:"StickY"`
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

func buildJoyConDeviceInfoCommand(packet byte) ([]byte, error) {
	return buildJoyConSubcommandReport(packet, 0x02, nil)
}

func parseJoyConControllerTypeReply(report []byte) (uint8, bool) {
	// 0x21 + controller state (12 bytes) + ACK + subcommand +
	// firmware major/minor + controller type.
	if len(report) < 18 || report[0] != joyConReportSubcommandReply {
		return 0, false
	}
	if report[13]&0x80 == 0 || report[14] != 0x02 {
		return 0, false
	}
	return report[17], true
}

func buildJoyConFullReportModeCommand(packet byte) ([]byte, error) {
	return buildJoyConSubcommandReport(packet, 0x03, []byte{joyConReportFull})
}
