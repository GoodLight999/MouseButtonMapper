package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	sinoWealthVID           = 0x258A
	sinoWealthPID1007       = 0x1007
	sinoWealthReportConfig  = 0x04
	sinoWealthReportCmd     = 0x05
	sinoWealthCmdLen        = 6
	sinoWealthReportLen     = 520
	sinoWealthConfigLen     = 123
	sinoWealthConfigWrite   = sinoWealthConfigLen - 8 // 0x73 (115)
	sinoWealthButtonsLen    = 88
	sinoWealthButtonsWrite  = sinoWealthButtonsLen - 8 // 0x50 (80)
	sinoWealthButtonSlots   = 20
	sinoWealthDPISlots      = 8
	sinoWealthCmdFWVersion  = 0x01
	sinoWealthCmdProfile    = 0x02
	sinoWealthCmdConfigP1   = 0x11
	sinoWealthCmdButtonsP1  = 0x12
	sinoWealthCmdConfigP2   = 0x21
	sinoWealthCmdButtonsP2  = 0x22
	sinoWealthSensorPMW3360 = 0x06
	sinoWealthSensorPMW3212 = 0x08
	sinoWealthSensorPMW3327 = 0x0E
	sinoWealthSensorPMW3389 = 0x0F
)

// SinoWealthButtonAction represents a single 4-byte onboard button slot.
type SinoWealthButtonAction struct {
	Index     int    `json:"index"`
	Label     string `json:"label"`
	Kind      string `json:"kind"` // mouse, wheel, key, media, dpi, dpi_lock, special, repeated, macro, none, raw
	Value     string `json:"value"`
	Modifiers uint8  `json:"modifiers,omitempty"` // bitmask: 1=LCtrl, 2=LShift, 4=LAlt, 8=LWin
	HIDUsage  uint8  `json:"hidUsage,omitempty"`  // USB HID Keyboard Usage ID (when kind=="key")
	DPILock   int    `json:"dpiLock,omitempty"`   // DPI value (when kind=="dpi_lock")
	RawHex    string `json:"rawHex"`              // 4-byte hex representation
	Display   string `json:"display"`             // Human-readable summary
}

type SinoWealthDPISlot struct {
	Index   int    `json:"index"`
	Enabled bool   `json:"enabled"`
	DPI     int    `json:"dpi"`
	Color   string `json:"color"` // #RRGGBB (raw byte order)
}

type SinoWealthConfigState struct {
	VendorID         uint16                   `json:"vendorId"`
	ProductID        uint16                   `json:"productId"`
	Firmware         string                   `json:"firmware"`
	ProductName      string                   `json:"productName"`
	SensorType       uint8                    `json:"sensorType"`
	SensorName       string                   `json:"sensorName"`
	MaxDPI           int                      `json:"maxDpi"`
	ActiveProfile    int                      `json:"activeProfile"`
	ReportRateHz     int                      `json:"reportRateHz"`
	ActiveDPISlot    int                      `json:"activeDpiSlot"`  // 1-based among enabled slots (or slot index+1)
	ActiveDPIIndex   int                      `json:"activeDpiIndex"` // 0-based physical slot index
	XYIndependent    bool                     `json:"xyIndependent"`
	DPISlots         []SinoWealthDPISlot      `json:"dpiSlots"`
	ButtonCount      int                      `json:"buttonCount"`
	Buttons          []SinoWealthButtonAction `json:"buttons"`
	RGBEffectRaw     uint8                    `json:"rgbEffectRaw"`
	RGBEffectMode    uint8                    `json:"rgbEffectMode"`
	SingleColor      string                   `json:"singleColor"`
	SingleBrightness int                      `json:"singleBrightness"`
	ColorOrder       string                   `json:"colorOrder"`
	RawConfigHex     string                   `json:"rawConfigHex,omitempty"`
	RawButtonsHex    string                   `json:"rawButtonsHex,omitempty"`
}

// SinoWealthHardwareLink configures optional per-profile hardware sync.
type SinoWealthHardwareLink struct {
	Enabled       bool   `json:"Enabled,omitempty"`
	ReportRateHz  int    `json:"ReportRateHz,omitempty"`  // 0=unchanged, 125/250/500/1000
	ActiveDPISlot int    `json:"ActiveDPISlot,omitempty"` // 0=unchanged, 1..8 (1-based physical slot index)
	LEDMode       int    `json:"LEDMode,omitempty"`       // 0=unchanged, 1=single color, 2=off, 3=rainbow
	LEDColor      string `json:"LEDColor,omitempty"`      // #RRGGBB when LEDMode==1
}

type SinoWealthHIDKeyOption struct {
	Usage uint8  `json:"usage"`
	Name  string `json:"name"`
	Group string `json:"group"`
}

var sinoWealthPawnPlusLabels = []string{
	"左クリック",
	"右クリック",
	"中クリック (ホイール)",
	"左サイド (戻る / X1)",
	"左サイド (進む / X2)",
	"DPI + ボタン",
	"DPI - ボタン",
	"右サイド (上)",
	"右サイド (下)",
}

var sinoWealthHIDKeys = []SinoWealthHIDKeyOption{
	{0x68, "F13", "拡張Fキー (安全・推奨)"},
	{0x6D, "F18", "拡張Fキー (安全・推奨)"},
	{0x6E, "F19", "拡張Fキー (安全・推奨)"},
	{0x6F, "F20", "拡張Fキー (安全・推奨)"},
	{0x6C, "F17", "拡張Fキー (一部修飾キー注意)"},
	{0x69, "F14", "拡張Fキー (Win併用注意)"},
	{0x6A, "F15", "拡張Fキー (Win併用注意)"},
	{0x6B, "F16", "拡張Fキー (Win+F16はシャットダウン注意)"},
	{0x70, "F21", "拡張Fキー (Win併用注意)"},
	{0x71, "F22", "拡張Fキー (Win併用注意)"},
	{0x72, "F23", "拡張Fキー (Win併用注意)"},
	{0x73, "F24", "拡張Fキー (Win併用注意)"},
	{0x4B, "PageUp", "ナビゲーション"},
	{0x4E, "PageDown", "ナビゲーション"},
	{0x4A, "Home", "ナビゲーション"},
	{0x4D, "End", "ナビゲーション"},
	{0x49, "Insert", "ナビゲーション"},
	{0x4C, "Delete", "ナビゲーション"},
	{0x29, "Escape", "制御"},
	{0x2B, "Tab", "制御"},
	{0x28, "Enter", "制御"},
	{0x2A, "Backspace", "制御"},
	{0x2C, "Space", "制御"},
	{0x46, "PrintScreen", "制御"},
	{0x4F, "Right", "矢印"},
	{0x50, "Left", "矢印"},
	{0x51, "Down", "矢印"},
	{0x52, "Up", "矢印"},
	{0x3A, "F1", "Fキー"},
	{0x3B, "F2", "Fキー"},
	{0x3C, "F3", "Fキー"},
	{0x3D, "F4", "Fキー"},
	{0x3E, "F5", "Fキー"},
	{0x3F, "F6", "Fキー"},
	{0x40, "F7", "Fキー"},
	{0x41, "F8", "Fキー"},
	{0x42, "F9", "Fキー"},
	{0x43, "F10", "Fキー"},
	{0x44, "F11", "Fキー"},
	{0x45, "F12", "Fキー"},
	{0x04, "A", "文字"}, {0x05, "B", "文字"}, {0x06, "C", "文字"}, {0x07, "D", "文字"},
	{0x08, "E", "文字"}, {0x09, "F", "文字"}, {0x0A, "G", "文字"}, {0x0B, "H", "文字"},
	{0x0C, "I", "文字"}, {0x0D, "J", "文字"}, {0x0E, "K", "文字"}, {0x0F, "L", "文字"},
	{0x10, "M", "文字"}, {0x11, "N", "文字"}, {0x12, "O", "文字"}, {0x13, "P", "文字"},
	{0x14, "Q", "文字"}, {0x15, "R", "文字"}, {0x16, "S", "文字"}, {0x17, "T", "文字"},
	{0x18, "U", "文字"}, {0x19, "V", "文字"}, {0x1A, "W", "文字"}, {0x1B, "X", "文字"},
	{0x1C, "Y", "文字"}, {0x1D, "Z", "文字"},
	{0x1E, "1", "数字"}, {0x1F, "2", "数字"}, {0x20, "3", "数字"}, {0x21, "4", "数字"},
	{0x22, "5", "数字"}, {0x23, "6", "数字"}, {0x24, "7", "数字"}, {0x25, "8", "数字"},
	{0x26, "9", "数字"}, {0x27, "0", "数字"},
}

func sinoWealthSensorInfo(sensor uint8) (name string, maxDPI int) {
	switch sensor {
	case sinoWealthSensorPMW3360:
		return "PixArt PMW3360", 12000
	case sinoWealthSensorPMW3327:
		return "PixArt PMW3327", 10200
	case sinoWealthSensorPMW3389:
		return "PixArt PMW3389", 16000
	case sinoWealthSensorPMW3212:
		return "PixArt PMW3212", 7200
	default:
		return fmt.Sprintf("Unknown (0x%02X)", sensor), 4000
	}
}

func sinoWealthRawToDPI(sensor uint8, raw uint8) int {
	v := int(raw)
	if sensor == sinoWealthSensorPMW3360 || sensor == sinoWealthSensorPMW3327 {
		v++
	}
	return v * 100
}

func sinoWealthDPIToRaw(sensor uint8, dpi int) (uint8, error) {
	_, maxDPI := sinoWealthSensorInfo(sensor)
	if dpi < 100 || dpi > maxDPI || dpi%100 != 0 {
		return 0, fmt.Errorf("DPIは100〜%dの100刻みで指定してください（入力: %d）", maxDPI, dpi)
	}
	v := dpi / 100
	if sensor == sinoWealthSensorPMW3360 || sensor == sinoWealthSensorPMW3327 {
		v--
	}
	if v < 0 || v > 255 {
		return 0, fmt.Errorf("DPI値 %d はエンコード範囲外です", dpi)
	}
	return uint8(v), nil
}

func sinoWealthRawToRate(raw uint8) int {
	switch raw & 0x0F {
	case 1:
		return 125
	case 2:
		return 250
	case 3:
		return 500
	case 4:
		return 1000
	default:
		return 0
	}
}

func sinoWealthRateToRaw(hz int) (uint8, error) {
	switch hz {
	case 125:
		return 1, nil
	case 250:
		return 2, nil
	case 500:
		return 3, nil
	case 1000:
		return 4, nil
	default:
		return 0, fmt.Errorf("未対応のポーリングレートです: %d Hz（125/250/500/1000のみ対応）", hz)
	}
}

func sinoWealthHIDUsageName(usage uint8) string {
	for _, k := range sinoWealthHIDKeys {
		if k.Usage == usage {
			return k.Name
		}
	}
	return fmt.Sprintf("HID 0x%02X", usage)
}

func sinoWealthModifiersText(mod uint8) string {
	var parts []string
	if mod&0x01 != 0 {
		parts = append(parts, "Ctrl")
	}
	if mod&0x02 != 0 {
		parts = append(parts, "Shift")
	}
	if mod&0x04 != 0 {
		parts = append(parts, "Alt")
	}
	if mod&0x08 != 0 {
		parts = append(parts, "Win")
	}
	return strings.Join(parts, "+")
}

func sinoWealthButtonLabel(index int) string {
	if index >= 0 && index < len(sinoWealthPawnPlusLabels) {
		return sinoWealthPawnPlusLabels[index]
	}
	return fmt.Sprintf("ボタン %d", index+1)
}

func decodeSinoWealthButton(index int, raw [4]byte) SinoWealthButtonAction {
	act := SinoWealthButtonAction{
		Index:  index,
		Label:  sinoWealthButtonLabel(index),
		RawHex: hex.EncodeToString(raw[:]),
	}
	switch raw[0] {
	case 0x00:
		act.Kind = "none"
		act.Value = "none"
		act.Display = "未設定"
	case 0x11:
		act.Kind = "mouse"
		switch raw[1] {
		case 0x01:
			act.Value = "left"
			act.Display = "左クリック"
		case 0x02:
			act.Value = "right"
			act.Display = "右クリック"
		case 0x04:
			act.Value = "middle"
			act.Display = "中クリック"
		case 0x08:
			act.Value = "x1"
			act.Display = "戻る (X1)"
		case 0x10:
			act.Value = "x2"
			act.Display = "進む (X2)"
		default:
			act.Kind = "raw"
			act.Value = act.RawHex
			act.Display = fmt.Sprintf("マウス(0x%02X)", raw[1])
		}
	case 0x12:
		act.Kind = "wheel"
		switch raw[1] {
		case 0x01:
			act.Value = "up"
			act.Display = "ホイール上"
		case 0xFF:
			act.Value = "down"
			act.Display = "ホイール下"
		default:
			act.Kind = "raw"
			act.Value = act.RawHex
			act.Display = fmt.Sprintf("ホイール(0x%02X)", raw[1])
		}
	case 0x21:
		act.Kind = "key"
		act.Modifiers = raw[1] & 0x0F
		act.HIDUsage = raw[2]
		keyName := sinoWealthHIDUsageName(raw[2])
		act.Value = fmt.Sprintf("0x%02X", raw[2])
		if mod := sinoWealthModifiersText(act.Modifiers); mod != "" {
			act.Display = mod + "+" + keyName
		} else {
			act.Display = keyName
		}
	case 0x22:
		act.Kind = "media"
		switch {
		case raw[1] == 0x01 && raw[2] == 0 && raw[3] == 0:
			act.Value = "next"
			act.Display = "次の曲"
		case raw[1] == 0x02 && raw[2] == 0 && raw[3] == 0:
			act.Value = "prev"
			act.Display = "前の曲"
		case raw[1] == 0x04 && raw[2] == 0 && raw[3] == 0:
			act.Value = "stop"
			act.Display = "停止"
		case raw[1] == 0x08 && raw[2] == 0 && raw[3] == 0:
			act.Value = "play_pause"
			act.Display = "再生/一時停止"
		case raw[1] == 0x10 && raw[2] == 0 && raw[3] == 0:
			act.Value = "mute"
			act.Display = "ミュート"
		case raw[1] == 0x40 && raw[2] == 0 && raw[3] == 0:
			act.Value = "vol_up"
			act.Display = "音量アップ"
		case raw[1] == 0x80 && raw[2] == 0 && raw[3] == 0:
			act.Value = "vol_down"
			act.Display = "音量ダウン"
		default:
			act.Kind = "raw"
			act.Value = act.RawHex
			act.Display = "メディア(" + act.RawHex + ")"
		}
	case 0x41:
		act.Kind = "dpi"
		switch raw[1] {
		case 0x00:
			act.Value = "cycle"
			act.Display = "DPI 循環"
		case 0x01:
			act.Value = "up"
			act.Display = "DPI +"
		case 0x02:
			act.Value = "down"
			act.Display = "DPI -"
		default:
			act.Kind = "raw"
			act.Value = act.RawHex
			act.Display = fmt.Sprintf("DPI(0x%02X)", raw[1])
		}
	case 0x42:
		act.Kind = "dpi_lock"
		act.DPILock = int(raw[1]) * 100
		act.Value = fmt.Sprintf("%d", act.DPILock)
		act.Display = fmt.Sprintf("DPIロック (%d)", act.DPILock)
	case 0x50:
		act.Kind = "special"
		switch raw[1] {
		case 0x01:
			act.Value = "disabled"
			act.Display = "無効"
		case 0x04:
			act.Value = "rate_cycle"
			act.Display = "ポーリングレート切替"
		case 0x06:
			act.Value = "profile_cycle"
			act.Display = "プロファイル切替"
		case 0x07:
			act.Value = "led_cycle"
			act.Display = "LEDモード切替"
		default:
			act.Kind = "raw"
			act.Value = act.RawHex
			act.Display = fmt.Sprintf("特殊(0x%02X)", raw[1])
		}
	case 0x31:
		act.Kind = "repeated"
		act.Value = act.RawHex
		act.Display = fmt.Sprintf("連射 (btn=%d, %dms, x%d)", raw[1], raw[2], raw[3])
	case 0x70:
		act.Kind = "macro"
		act.Value = act.RawHex
		act.Display = fmt.Sprintf("マクロ #%d", raw[1])
	default:
		act.Kind = "raw"
		act.Value = act.RawHex
		act.Display = "Raw " + act.RawHex
	}
	return act
}

func encodeSinoWealthButton(act SinoWealthButtonAction) ([4]byte, error) {
	var out [4]byte
	switch strings.ToLower(strings.TrimSpace(act.Kind)) {
	case "none":
		return out, nil
	case "mouse":
		out[0] = 0x11
		switch strings.ToLower(strings.TrimSpace(act.Value)) {
		case "left":
			out[1] = 0x01
		case "right":
			out[1] = 0x02
		case "middle":
			out[1] = 0x04
		case "x1", "back":
			out[1] = 0x08
		case "x2", "forward":
			out[1] = 0x10
		default:
			return out, fmt.Errorf("未対応のマウスボタン値: %q", act.Value)
		}
		return out, nil
	case "wheel":
		out[0] = 0x12
		switch strings.ToLower(strings.TrimSpace(act.Value)) {
		case "up":
			out[1] = 0x01
		case "down":
			out[1] = 0xFF
		default:
			return out, fmt.Errorf("未対応のホイール値: %q", act.Value)
		}
		return out, nil
	case "key":
		usage := act.HIDUsage
		if usage == 0 && act.Value != "" {
			var parsed int
			if _, err := fmt.Sscanf(strings.TrimSpace(act.Value), "0x%x", &parsed); err == nil && parsed > 0 && parsed <= 0xFF {
				usage = uint8(parsed)
			} else if _, err := fmt.Sscanf(strings.TrimSpace(act.Value), "%d", &parsed); err == nil && parsed > 0 && parsed <= 0xFF {
				usage = uint8(parsed)
			}
		}
		if usage == 0 {
			return out, errors.New("キー割り当てには有効なHIDキーコードが必要です")
		}
		out[0] = 0x21
		out[1] = act.Modifiers & 0x0F
		out[2] = usage
		out[3] = 0x00
		return out, nil
	case "media":
		out[0] = 0x22
		switch strings.ToLower(strings.TrimSpace(act.Value)) {
		case "next":
			out[1] = 0x01
		case "prev":
			out[1] = 0x02
		case "stop":
			out[1] = 0x04
		case "play_pause":
			out[1] = 0x08
		case "mute":
			out[1] = 0x10
		case "vol_up":
			out[1] = 0x40
		case "vol_down":
			out[1] = 0x80
		default:
			return out, fmt.Errorf("未対応のメディアキー値: %q", act.Value)
		}
		return out, nil
	case "dpi":
		out[0] = 0x41
		switch strings.ToLower(strings.TrimSpace(act.Value)) {
		case "cycle":
			out[1] = 0x00
		case "up":
			out[1] = 0x01
		case "down":
			out[1] = 0x02
		default:
			return out, fmt.Errorf("未対応のDPI操作値: %q", act.Value)
		}
		return out, nil
	case "dpi_lock":
		dpi := act.DPILock
		if dpi < 100 || dpi > 16000 || dpi%100 != 0 {
			return out, fmt.Errorf("DPIロック値は100〜16000の100刻みで指定してください: %d", dpi)
		}
		out[0] = 0x42
		out[1] = uint8(dpi / 100)
		return out, nil
	case "special":
		out[0] = 0x50
		switch strings.ToLower(strings.TrimSpace(act.Value)) {
		case "disabled":
			out[1] = 0x01
		case "rate_cycle":
			out[1] = 0x04
		case "profile_cycle":
			out[1] = 0x06
		case "led_cycle":
			out[1] = 0x07
		default:
			return out, fmt.Errorf("未対応の特殊機能値: %q", act.Value)
		}
		return out, nil
	case "raw", "repeated", "macro":
		rawStr := strings.TrimSpace(act.RawHex)
		if rawStr == "" {
			rawStr = strings.TrimSpace(act.Value)
		}
		rawBytes, err := hex.DecodeString(strings.ReplaceAll(rawStr, " ", ""))
		if err != nil || len(rawBytes) != 4 {
			return out, fmt.Errorf("Rawボタン値は4バイト(8桁hex)で指定してください: %q", rawStr)
		}
		copy(out[:], rawBytes)
		return out, nil
	default:
		return out, fmt.Errorf("未知のボタン種別です: %q", act.Kind)
	}
}

func encodeRGBOrder(r, g, b uint8, order string) (uint8, uint8, uint8) {
	if strings.EqualFold(strings.TrimSpace(order), "RBG") {
		return r, b, g
	}
	return r, g, b
}

func decodeRGBOrder(b0, b1, b2 uint8, order string) (uint8, uint8, uint8) {
	if strings.EqualFold(strings.TrimSpace(order), "RBG") {
		return b0, b2, b1
	}
	return b0, b1, b2
}

func normalizeColorOrder(order string) string {
	if strings.EqualFold(strings.TrimSpace(order), "RBG") {
		return "RBG"
	}
	return "RGB"
}

func decodeSinoWealthState(vid, pid uint16, fw, productName string, activeProfile int, cfgBuf, btnBuf []byte, colorOrder ...string) (SinoWealthConfigState, error) {
	order := "RGB"
	if len(colorOrder) > 0 {
		order = normalizeColorOrder(colorOrder[0])
	}
	if len(cfgBuf) != sinoWealthReportLen {
		return SinoWealthConfigState{}, fmt.Errorf("設定レポート長が不正です: %d (期待値 %d)", len(cfgBuf), sinoWealthReportLen)
	}
	if len(btnBuf) != sinoWealthReportLen {
		return SinoWealthConfigState{}, fmt.Errorf("ボタンレポート長が不正です: %d (期待値 %d)", len(btnBuf), sinoWealthReportLen)
	}
	if cfgBuf[0] != sinoWealthReportConfig || cfgBuf[1] != sinoWealthCmdConfigP1 {
		return SinoWealthConfigState{}, fmt.Errorf("設定レポートヘッダが不正です: %02X %02X", cfgBuf[0], cfgBuf[1])
	}
	if btnBuf[0] != sinoWealthReportConfig || btnBuf[1] != sinoWealthCmdButtonsP1 {
		return SinoWealthConfigState{}, fmt.Errorf("ボタンレポートヘッダが不正です: %02X %02X", btnBuf[0], btnBuf[1])
	}

	sensor := cfgBuf[0x09]
	sensorName, maxDPI := sinoWealthSensorInfo(sensor)
	rateHz := sinoWealthRawToRate(cfgBuf[0x0A] & 0x0F)
	xyIndep := (cfgBuf[0x0A] & 0x80) != 0
	activeDPIOrdinal := int((cfgBuf[0x0B] >> 4) & 0x0F) // 1-based among enabled slots
	disabledMask := cfgBuf[0x0C]

	dpiSlots := make([]SinoWealthDPISlot, sinoWealthDPISlots)
	enabledOrd := 0
	activePhysicalIdx := 0
	for i := 0; i < sinoWealthDPISlots; i++ {
		enabled := (disabledMask & (1 << uint(i))) == 0
		var rawDPI uint8
		if xyIndep {
			rawDPI = cfgBuf[0x0D+i*2]
		} else {
			rawDPI = cfgBuf[0x0D+i]
		}
		cOff := 0x1D + i*3
		dpiVal := sinoWealthRawToDPI(sensor, rawDPI)
		if !enabled && rawDPI == 0 {
			dpiVal = 0
		}
		cr, cg, cb := decodeRGBOrder(cfgBuf[cOff], cfgBuf[cOff+1], cfgBuf[cOff+2], order)
		dpiSlots[i] = SinoWealthDPISlot{
			Index:   i,
			Enabled: enabled,
			DPI:     dpiVal,
			Color:   fmt.Sprintf("#%02X%02X%02X", cr, cg, cb),
		}
		if enabled {
			enabledOrd++
			if enabledOrd == activeDPIOrdinal {
				activePhysicalIdx = i
			}
		}
	}

	buttonCount := 9
	if fw != "GM07" {
		// Detect last non-zero button slot up to 20, minimum 6.
		buttonCount = 6
		for i := 0; i < sinoWealthButtonSlots; i++ {
			off := 0x08 + i*4
			if btnBuf[off] != 0 || btnBuf[off+1] != 0 || btnBuf[off+2] != 0 || btnBuf[off+3] != 0 {
				buttonCount = i + 1
			}
		}
	}
	buttons := make([]SinoWealthButtonAction, buttonCount)
	for i := 0; i < buttonCount; i++ {
		off := 0x08 + i*4
		var raw [4]byte
		copy(raw[:], btnBuf[off:off+4])
		buttons[i] = decodeSinoWealthButton(i, raw)
	}

	sr, sg, sb := decodeRGBOrder(cfgBuf[0x39], cfgBuf[0x3A], cfgBuf[0x3B], order)
	singleBrightness := int((cfgBuf[0x38] >> 4) & 0x0F)
	if singleBrightness == 0 {
		singleBrightness = 4
	}

	return SinoWealthConfigState{
		VendorID:         vid,
		ProductID:        pid,
		Firmware:         fw,
		ProductName:      productName,
		SensorType:       sensor,
		SensorName:       sensorName,
		MaxDPI:           maxDPI,
		ActiveProfile:    activeProfile,
		ReportRateHz:     rateHz,
		ActiveDPISlot:    activeDPIOrdinal,
		ActiveDPIIndex:   activePhysicalIdx,
		XYIndependent:    xyIndep,
		DPISlots:         dpiSlots,
		ButtonCount:      buttonCount,
		Buttons:          buttons,
		RGBEffectRaw:     cfgBuf[0x35],
		RGBEffectMode:    cfgBuf[0x35] & 0x0F,
		SingleColor:      fmt.Sprintf("#%02X%02X%02X", sr, sg, sb),
		SingleBrightness: singleBrightness,
		ColorOrder:       order,
		RawConfigHex:     hex.EncodeToString(cfgBuf[:sinoWealthConfigLen]),
		RawButtonsHex:    hex.EncodeToString(btnBuf[:sinoWealthButtonsLen]),
	}, nil
}

// buildSinoWealthButtonsWriteReport performs safe Read-Modify-Write on an existing 520-byte BUTTONS report.
func buildSinoWealthButtonsWriteReport(origBtnBuf []byte, updates []SinoWealthButtonAction) ([]byte, error) {
	if len(origBtnBuf) != sinoWealthReportLen {
		return nil, fmt.Errorf("元のボタンレポート長が不正です: %d", len(origBtnBuf))
	}
	if origBtnBuf[0] != sinoWealthReportConfig || origBtnBuf[1] != sinoWealthCmdButtonsP1 {
		return nil, fmt.Errorf("元のボタンレポートヘッダが不正です: %02X %02X", origBtnBuf[0], origBtnBuf[1])
	}
	out := make([]byte, sinoWealthReportLen)
	copy(out, origBtnBuf)
	out[0x03] = sinoWealthButtonsWrite

	for _, act := range updates {
		if act.Index < 0 || act.Index >= sinoWealthButtonSlots {
			return nil, fmt.Errorf("ボタン番号が範囲外です: %d", act.Index)
		}
		raw, err := encodeSinoWealthButton(act)
		if err != nil {
			return nil, fmt.Errorf("ボタン %d のエンコードに失敗: %w", act.Index+1, err)
		}
		off := 0x08 + act.Index*4
		copy(out[off:off+4], raw[:])
	}
	return out, nil
}

type SinoWealthConfigPatch struct {
	ReportRateHz       int                 `json:"reportRateHz,omitempty"`       // 0=unchanged
	ActiveDPIIndex     int                 `json:"activeDpiIndex"`               // -1=unchanged, 0..7=physical slot index
	DPISlots           []SinoWealthDPISlot `json:"dpiSlots,omitempty"`           // nil=unchanged
	RGBEffectMode      int                 `json:"rgbEffectMode"`                // -1=unchanged, 0..10 (preserves high nibble!)
	SingleColor        string              `json:"singleColor,omitempty"`        // "#RRGGBB" for body single-color LED (0x39..0x3B)
	SingleBrightness   int                 `json:"singleBrightness,omitempty"`   // 0=unchanged(or 4), 1..4
	SyncActiveDPIColor bool                `json:"syncActiveDpiColor,omitempty"` // also paint active DPI slot indicator with SingleColor
	ColorOrder         string              `json:"colorOrder,omitempty"`         // "RGB" (default) or "RBG"
}

// buildSinoWealthConfigWriteReport performs safe Read-Modify-Write on an existing 520-byte CONFIG report.
// It preserves sensor_type, fixed header bytes (0x06, 0x08), high nibble of 0x35, and tail LED tables.
func buildSinoWealthConfigWriteReport(origCfgBuf []byte, patch SinoWealthConfigPatch) ([]byte, error) {
	if len(origCfgBuf) != sinoWealthReportLen {
		return nil, fmt.Errorf("元の設定レポート長が不正です: %d", len(origCfgBuf))
	}
	if origCfgBuf[0] != sinoWealthReportConfig || origCfgBuf[1] != sinoWealthCmdConfigP1 {
		return nil, fmt.Errorf("元の設定レポートヘッダが不正です: %02X %02X", origCfgBuf[0], origCfgBuf[1])
	}
	sensor := origCfgBuf[0x09]
	if sensor == 0 {
		return nil, errors.New("センサー種別が0のため安全のため書き込みを中止しました")
	}

	order := normalizeColorOrder(patch.ColorOrder)
	out := make([]byte, sinoWealthReportLen)
	copy(out, origCfgBuf)
	out[0x03] = sinoWealthConfigWrite

	if patch.ReportRateHz > 0 {
		rawRate, err := sinoWealthRateToRaw(patch.ReportRateHz)
		if err != nil {
			return nil, err
		}
		out[0x0A] = (out[0x0A] & 0xF0) | (rawRate & 0x0F)
	}

	xyIndep := (out[0x0A] & 0x80) != 0
	if len(patch.DPISlots) > 0 {
		if len(patch.DPISlots) != sinoWealthDPISlots {
			return nil, fmt.Errorf("DPIスロット数は%d個である必要があります", sinoWealthDPISlots)
		}
		var enabledMask uint8
		enabledCount := 0
		for i, slot := range patch.DPISlots {
			if slot.Enabled {
				rawDPI, err := sinoWealthDPIToRaw(sensor, slot.DPI)
				if err != nil {
					return nil, fmt.Errorf("DPIスロット %d: %w", i+1, err)
				}
				enabledMask |= 1 << uint(i)
				enabledCount++
				if xyIndep {
					out[0x0D+i*2] = rawDPI
					out[0x0D+i*2+1] = rawDPI
				} else {
					out[0x0D+i] = rawDPI
				}
			} else {
				if xyIndep {
					out[0x0D+i*2] = 0
					out[0x0D+i*2+1] = 0
				} else {
					out[0x0D+i] = 0
				}
			}
			if slot.Color != "" {
				r, g, b, err := parseHexRGB(slot.Color)
				if err != nil {
					return nil, fmt.Errorf("DPIスロット %d の色指定が不正です: %w", i+1, err)
				}
				c0, c1, c2 := encodeRGBOrder(r, g, b, order)
				cOff := 0x1D + i*3
				out[cOff] = c0
				out[cOff+1] = c1
				out[cOff+2] = c2
			}
		}
		if enabledCount == 0 {
			return nil, errors.New("少なくとも1つのDPIスロットを有効にする必要があります")
		}
		out[0x0C] = ^enabledMask
		// Keep current active ordinal clamped to enabledCount
		curOrd := int((out[0x0B] >> 4) & 0x0F)
		if curOrd < 1 || curOrd > enabledCount {
			curOrd = 1
		}
		out[0x0B] = uint8((curOrd << 4) | (enabledCount & 0x0F))
	}

	if patch.ActiveDPIIndex >= 0 {
		if patch.ActiveDPIIndex >= sinoWealthDPISlots {
			return nil, fmt.Errorf("アクティブDPIスロット番号が範囲外です: %d", patch.ActiveDPIIndex)
		}
		disabledMask := out[0x0C]
		if (disabledMask & (1 << uint(patch.ActiveDPIIndex))) != 0 {
			return nil, fmt.Errorf("無効化されたDPIスロット %d はアクティブにできません", patch.ActiveDPIIndex+1)
		}
		ord := 0
		enabledCount := 0
		for i := 0; i < sinoWealthDPISlots; i++ {
			if (disabledMask & (1 << uint(i))) == 0 {
				enabledCount++
				if i == patch.ActiveDPIIndex {
					ord = enabledCount
				}
			}
		}
		out[0x0B] = uint8((ord << 4) | (enabledCount & 0x0F))
	}

	if patch.RGBEffectMode >= 0 {
		if patch.RGBEffectMode > 0x0A {
			return nil, fmt.Errorf("未対応のLEDモードです: %d", patch.RGBEffectMode)
		}
		// Preserve upper nibble (device-specific flags, e.g. 0x20 on GM07)
		out[0x35] = (out[0x35] & 0xF0) | uint8(patch.RGBEffectMode&0x0F)
	}

	if strings.TrimSpace(patch.SingleColor) != "" {
		r, g, b, err := parseHexRGB(patch.SingleColor)
		if err != nil {
			return nil, fmt.Errorf("本体LED色指定が不正です: %w", err)
		}
		c0, c1, c2 := encodeRGBOrder(r, g, b, order)
		bright := patch.SingleBrightness
		if bright <= 0 {
			bright = int((out[0x38] >> 4) & 0x0F)
		}
		if bright < 1 || bright > 4 {
			bright = 4
		}
		out[0x38] = uint8(bright << 4)
		out[0x39] = c0
		out[0x3A] = c1
		out[0x3B] = c2

		if patch.SyncActiveDPIColor {
			disabledMask := out[0x0C]
			activeOrd := int((out[0x0B] >> 4) & 0x0F)
			enabledOrd := 0
			activeIdx := 0
			for i := 0; i < sinoWealthDPISlots; i++ {
				if (disabledMask & (1 << uint(i))) == 0 {
					enabledOrd++
					if enabledOrd == activeOrd {
						activeIdx = i
						break
					}
				}
			}
			cOff := 0x1D + activeIdx*3
			out[cOff] = c0
			out[cOff+1] = c1
			out[cOff+2] = c2
		}
	}

	return out, nil
}

func parseHexRGB(s string) (uint8, uint8, uint8, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("#RRGGBB形式で指定してください: %q", s)
	}
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 3 {
		return 0, 0, 0, fmt.Errorf("16進カラー文字列が不正です: %q", s)
	}
	return b[0], b[1], b[2], nil
}

// sinoWealthPresetButtons builds curated onboard button layouts for MouseButtonMapper integration.
func sinoWealthPresetButtons(preset string) ([]SinoWealthButtonAction, error) {
	base := []SinoWealthButtonAction{
		{Index: 0, Kind: "mouse", Value: "left"},
		{Index: 1, Kind: "mouse", Value: "right"},
		{Index: 2, Kind: "mouse", Value: "middle"},
		{Index: 3, Kind: "mouse", Value: "x1"},
		{Index: 4, Kind: "mouse", Value: "x2"},
		{Index: 5, Kind: "dpi", Value: "up"},
		{Index: 6, Kind: "dpi", Value: "down"},
		{Index: 7, Kind: "key", HIDUsage: 0x4B}, // PageUp
		{Index: 8, Kind: "key", HIDUsage: 0x4E}, // PageDown
	}
	switch preset {
	case "mbm-side-f13-f14", "mbm-side-safe":
		// Keep DPI+/- on hardware, map Right Side buttons to F13/F18 (avoiding Windows hidden Win+F14/F15/F16 hotkeys)
		base[7] = SinoWealthButtonAction{Index: 7, Kind: "key", HIDUsage: 0x68} // F13
		base[8] = SinoWealthButtonAction{Index: 8, Kind: "key", HIDUsage: 0x6D} // F18
		return base, nil
	case "mbm-all-f13-f16", "mbm-all-safe":
		// Map Right Side (F13/F18) and DPI+/- (F19/F20) to Windows 11 conflict-free F-keys.
		// Avoids Win+F14, Win+F15, Win+F16 (SlideToShutDown.exe), Win+F21..F24.
		base[5] = SinoWealthButtonAction{Index: 5, Kind: "key", HIDUsage: 0x6E} // F19
		base[6] = SinoWealthButtonAction{Index: 6, Kind: "key", HIDUsage: 0x6F} // F20
		base[7] = SinoWealthButtonAction{Index: 7, Kind: "key", HIDUsage: 0x68} // F13
		base[8] = SinoWealthButtonAction{Index: 8, Kind: "key", HIDUsage: 0x6D} // F18
		return base, nil
	case "factory-pawn-plus":
		return base, nil
	default:
		return nil, fmt.Errorf("未知のプリセット名です: %q", preset)
	}
}

// sinoWealthReportsEqualIgnoringWriteFlag compares two 520-byte reports ignoring byte[0x03] (config_write).
func sinoWealthReportsEqualIgnoringWriteFlag(a, b []byte) bool {
	if len(a) != len(b) || len(a) < 8 {
		return false
	}
	if !bytes.Equal(a[:3], b[:3]) {
		return false
	}
	return bytes.Equal(a[4:], b[4:])
}
