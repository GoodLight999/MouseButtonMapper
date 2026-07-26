package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	joyConNintendoVendorID uint16 = 0x057e
	joyConLeftProductID    uint16 = 0x2006

	joyConReportSubcommandReply byte = 0x21
	joyConReportFull            byte = 0x30
	joyConReportSimple          byte = 0x3f

	joyConDirectionMode4 = "4"
	joyConDirectionMode8 = "8"
)

type JoyConButton string

const (
	JoyConButtonUp      JoyConButton = "Up"
	JoyConButtonDown    JoyConButton = "Down"
	JoyConButtonLeft    JoyConButton = "Left"
	JoyConButtonRight   JoyConButton = "Right"
	JoyConButtonL       JoyConButton = "L"
	JoyConButtonZL      JoyConButton = "ZL"
	JoyConButtonSL      JoyConButton = "SL"
	JoyConButtonSR      JoyConButton = "SR"
	JoyConButtonMinus   JoyConButton = "Minus"
	JoyConButtonCapture JoyConButton = "Capture"
	JoyConButtonStick   JoyConButton = "StickPress"
	JoyConStickUp       JoyConButton = "StickUp"
	JoyConStickDown     JoyConButton = "StickDown"
	JoyConStickLeft     JoyConButton = "StickLeft"
	JoyConStickRight    JoyConButton = "StickRight"
)

var joyConLeftPhysicalButtons = []JoyConButton{
	JoyConButtonUp,
	JoyConButtonDown,
	JoyConButtonLeft,
	JoyConButtonRight,
	JoyConButtonL,
	JoyConButtonZL,
	JoyConButtonSL,
	JoyConButtonSR,
	JoyConButtonMinus,
	JoyConButtonCapture,
	JoyConButtonStick,
}

var joyConStickDirections = []JoyConButton{
	JoyConStickUp,
	JoyConStickDown,
	JoyConStickLeft,
	JoyConStickRight,
}

// InputToken is the OS-independent identity used while inputs from different
// devices share one pressed-state set. Existing config.json Item values remain
// the persistence boundary; adapters translate Item to and from this type.
type InputToken struct {
	Kind     string `json:"Kind"`
	Code     string `json:"Code"`
	DeviceID string `json:"DeviceId,omitempty"`
}

func (t InputToken) Normalized() InputToken {
	t.Kind = strings.TrimSpace(t.Kind)
	t.Code = strings.TrimSpace(t.Code)
	t.DeviceID = strings.TrimSpace(t.DeviceID)
	if strings.EqualFold(t.Kind, "joycon") {
		t.Kind = "JoyCon"
	}
	return t
}

func (t InputToken) Key() string {
	t = t.Normalized()
	kind := strings.ToLower(t.Kind)
	code := strings.ToLower(t.Code)
	device := strings.ToLower(t.DeviceID)
	if device == "" {
		return kind + ":" + code
	}
	return kind + ":" + device + ":" + code
}

type InputEvent struct {
	Token      InputToken
	Down       bool
	SourceID   string
	OccurredAt time.Time
	Synthetic  bool
}

// JoyConRawState is one decoded controller report before calibration,
// hysteresis, profile mapping, or rule evaluation.
type JoyConRawState struct {
	ReportID       byte
	Buttons        map[JoyConButton]bool
	StickX         uint16
	StickY         uint16
	BatteryPercent int
	Charging       bool
}

func parseJoyConInputReport(report []byte) (JoyConRawState, error) {
	if len(report) == 0 {
		return JoyConRawState{}, fmt.Errorf("Joy-Con input report is empty")
	}
	state := JoyConRawState{
		ReportID:       report[0],
		Buttons:        make(map[JoyConButton]bool, len(joyConLeftPhysicalButtons)),
		StickX:         2000,
		StickY:         2000,
		BatteryPercent: -1,
	}

	switch report[0] {
	case joyConReportFull, joyConReportSubcommandReply:
		if len(report) < 12 {
			return JoyConRawState{}, fmt.Errorf("Joy-Con report 0x%02x is too short: %d", report[0], len(report))
		}
		parseJoyConStandardButtons(state.Buttons, report[4], report[5])
		state.StickX = uint16(report[6]) | uint16(report[7]&0x0f)<<8
		state.StickY = uint16(report[7]>>4) | uint16(report[8])<<4
		batteryNibble := report[2] >> 4
		state.Charging = batteryNibble&0x01 != 0
		state.BatteryPercent = int(batteryNibble>>1) * 25
		if state.BatteryPercent > 100 {
			state.BatteryPercent = 100
		}
		return state, nil

	case joyConReportSimple:
		if len(report) < 4 {
			return JoyConRawState{}, fmt.Errorf("Joy-Con simple report is too short: %d", len(report))
		}
		// Report 0x3f is an OS-compatible fallback. On an individual Joy-Con,
		// the analog bytes are filler; the hat still gives reliable directions.
		parseJoyConSimpleHat(state.Buttons, report[3]&0x0f)
		return state, nil

	default:
		return JoyConRawState{}, fmt.Errorf("unsupported Joy-Con report id 0x%02x", report[0])
	}
}

func parseJoyConStandardButtons(dst map[JoyConButton]bool, shared, left byte) {
	setJoyConButton(dst, JoyConButtonMinus, shared&(1<<0) != 0)
	setJoyConButton(dst, JoyConButtonStick, shared&(1<<3) != 0)
	setJoyConButton(dst, JoyConButtonCapture, shared&(1<<5) != 0)

	setJoyConButton(dst, JoyConButtonDown, left&(1<<0) != 0)
	setJoyConButton(dst, JoyConButtonUp, left&(1<<1) != 0)
	setJoyConButton(dst, JoyConButtonRight, left&(1<<2) != 0)
	setJoyConButton(dst, JoyConButtonLeft, left&(1<<3) != 0)
	setJoyConButton(dst, JoyConButtonSR, left&(1<<4) != 0)
	setJoyConButton(dst, JoyConButtonSL, left&(1<<5) != 0)
	setJoyConButton(dst, JoyConButtonL, left&(1<<6) != 0)
	setJoyConButton(dst, JoyConButtonZL, left&(1<<7) != 0)
}

func parseJoyConSimpleHat(dst map[JoyConButton]bool, hat byte) {
	up := hat == 0 || hat == 1 || hat == 7
	right := hat == 1 || hat == 2 || hat == 3
	down := hat == 3 || hat == 4 || hat == 5
	left := hat == 5 || hat == 6 || hat == 7
	setJoyConButton(dst, JoyConButtonUp, up)
	setJoyConButton(dst, JoyConButtonRight, right)
	setJoyConButton(dst, JoyConButtonDown, down)
	setJoyConButton(dst, JoyConButtonLeft, left)
}

func setJoyConButton(dst map[JoyConButton]bool, button JoyConButton, down bool) {
	if down {
		dst[button] = true
	}
}

type JoyConAxisCalibration struct {
	Min    uint16 `json:"Min"`
	Center uint16 `json:"Center"`
	Max    uint16 `json:"Max"`
}

type JoyConStickCalibration struct {
	X JoyConAxisCalibration `json:"X"`
	Y JoyConAxisCalibration `json:"Y"`
}

type JoyConStickConfig struct {
	DeadZone      float64                `json:"DeadZone,omitempty"`
	ReleaseZone   float64                `json:"ReleaseZone,omitempty"`
	DirectionMode string                 `json:"DirectionMode,omitempty"`
	InvertX       bool                   `json:"InvertX,omitempty"`
	InvertY       bool                   `json:"InvertY,omitempty"`
	Calibration   JoyConStickCalibration `json:"Calibration,omitempty"`
}

func defaultJoyConAxisCalibration() JoyConAxisCalibration {
	return JoyConAxisCalibration{Min: 500, Center: 2000, Max: 3500}
}

func defaultJoyConStickConfig() JoyConStickConfig {
	axis := defaultJoyConAxisCalibration()
	return JoyConStickConfig{
		DeadZone:      0.28,
		ReleaseZone:   0.20,
		DirectionMode: joyConDirectionMode8,
		Calibration: JoyConStickCalibration{
			X: axis,
			Y: axis,
		},
	}
}

func normalizeJoyConAxisCalibration(c JoyConAxisCalibration) JoyConAxisCalibration {
	if c.Min >= c.Center || c.Center >= c.Max || c.Max > 4095 || c.Max-c.Min < 512 {
		return defaultJoyConAxisCalibration()
	}
	return c
}

func normalizeJoyConStickConfig(c JoyConStickConfig) JoyConStickConfig {
	defaults := defaultJoyConStickConfig()
	if c.DeadZone <= 0 || c.DeadZone >= 0.95 {
		c.DeadZone = defaults.DeadZone
	}
	if c.ReleaseZone <= 0 || c.ReleaseZone >= c.DeadZone {
		c.ReleaseZone = math.Min(defaults.ReleaseZone, c.DeadZone*0.75)
	}
	switch strings.TrimSpace(c.DirectionMode) {
	case joyConDirectionMode4:
		c.DirectionMode = joyConDirectionMode4
	default:
		c.DirectionMode = joyConDirectionMode8
	}
	c.Calibration.X = normalizeJoyConAxisCalibration(c.Calibration.X)
	c.Calibration.Y = normalizeJoyConAxisCalibration(c.Calibration.Y)
	return c
}

func normalizeJoyConAxis(raw uint16, c JoyConAxisCalibration) float64 {
	c = normalizeJoyConAxisCalibration(c)
	var value float64
	if raw >= c.Center {
		value = float64(raw-c.Center) / float64(c.Max-c.Center)
	} else {
		value = -float64(c.Center-raw) / float64(c.Center-c.Min)
	}
	return clampUnit(value)
}

func normalizeJoyConStick(rawX, rawY uint16, config JoyConStickConfig) (float64, float64) {
	config = normalizeJoyConStickConfig(config)
	x := normalizeJoyConAxis(rawX, config.Calibration.X)
	// Nintendo's raw left-stick Y increases upward. Runtime coordinates use
	// positive Y for Up, so no GUI-specific inversion is hidden here.
	y := normalizeJoyConAxis(rawY, config.Calibration.Y)
	if config.InvertX {
		x = -x
	}
	if config.InvertY {
		y = -y
	}
	return clampUnit(x), clampUnit(y)
}

func clampUnit(v float64) float64 {
	if v < -1 {
		return -1
	}
	if v > 1 {
		return 1
	}
	return v
}

type JoyConDirectionState struct {
	down    map[JoyConButton]bool
	primary JoyConButton
}

func newJoyConDirectionState() JoyConDirectionState {
	return JoyConDirectionState{down: make(map[JoyConButton]bool, 4)}
}

func (s *JoyConDirectionState) update(x, y float64, config JoyConStickConfig) map[JoyConButton]bool {
	config = normalizeJoyConStickConfig(config)
	if s.down == nil {
		s.down = make(map[JoyConButton]bool, 4)
	}
	if config.DirectionMode == joyConDirectionMode4 {
		return s.update4Way(x, y, config)
	}
	return s.update8Way(x, y, config)
}

func (s *JoyConDirectionState) update8Way(x, y float64, config JoyConStickConfig) map[JoyConButton]bool {
	next := make(map[JoyConButton]bool, 4)
	updateAxisDirection(next, s.down, JoyConStickLeft, JoyConStickRight, x, config.DeadZone, config.ReleaseZone)
	updateAxisDirection(next, s.down, JoyConStickDown, JoyConStickUp, y, config.DeadZone, config.ReleaseZone)
	s.down = next
	s.primary = ""
	return cloneJoyConButtonSet(next)
}

func updateAxisDirection(next, previous map[JoyConButton]bool, negative, positive JoyConButton, value, press, release float64) {
	if value >= press || (previous[positive] && value > release) {
		next[positive] = true
		return
	}
	if value <= -press || (previous[negative] && value < -release) {
		next[negative] = true
	}
}

func (s *JoyConDirectionState) update4Way(x, y float64, config JoyConStickConfig) map[JoyConButton]bool {
	absX := math.Abs(x)
	absY := math.Abs(y)
	candidate := JoyConButton("")
	candidateMagnitude := 0.0
	if absX >= config.DeadZone || absY >= config.DeadZone {
		if absX >= absY {
			candidateMagnitude = absX
			if x >= 0 {
				candidate = JoyConStickRight
			} else {
				candidate = JoyConStickLeft
			}
		} else {
			candidateMagnitude = absY
			if y >= 0 {
				candidate = JoyConStickUp
			} else {
				candidate = JoyConStickDown
			}
		}
	}

	if s.primary != "" {
		currentMagnitude := directionMagnitude(s.primary, x, y)
		if currentMagnitude > config.ReleaseZone {
			// Keep the current axis around diagonals unless the other axis is
			// clearly stronger. This prevents rapid 4-way direction flipping.
			if candidate == "" || candidate == s.primary || candidateMagnitude < currentMagnitude+0.10 {
				candidate = s.primary
			}
		}
	}

	next := make(map[JoyConButton]bool, 1)
	if candidate != "" {
		next[candidate] = true
	}
	s.primary = candidate
	s.down = next
	return cloneJoyConButtonSet(next)
}

func directionMagnitude(direction JoyConButton, x, y float64) float64 {
	switch direction {
	case JoyConStickLeft:
		return math.Max(0, -x)
	case JoyConStickRight:
		return math.Max(0, x)
	case JoyConStickDown:
		return math.Max(0, -y)
	case JoyConStickUp:
		return math.Max(0, y)
	default:
		return 0
	}
}

func cloneJoyConButtonSet(src map[JoyConButton]bool) map[JoyConButton]bool {
	out := make(map[JoyConButton]bool, len(src))
	for button, down := range src {
		if down {
			out[button] = true
		}
	}
	return out
}

type JoyConStateTracker struct {
	SourceID   string
	down       map[JoyConButton]bool
	directions JoyConDirectionState
}

func newJoyConStateTracker(sourceID string) *JoyConStateTracker {
	return &JoyConStateTracker{
		SourceID:   strings.TrimSpace(sourceID),
		down:       make(map[JoyConButton]bool, 16),
		directions: newJoyConDirectionState(),
	}
}

func (t *JoyConStateTracker) Apply(raw JoyConRawState, config JoyConStickConfig, at time.Time) []InputEvent {
	if t.down == nil {
		t.down = make(map[JoyConButton]bool, 16)
	}
	next := make(map[JoyConButton]bool, 16)
	for _, button := range joyConLeftPhysicalButtons {
		if raw.Buttons[button] {
			next[button] = true
		}
	}
	x, y := normalizeJoyConStick(raw.StickX, raw.StickY, config)
	for button := range t.directions.update(x, y, config) {
		next[button] = true
	}
	events := diffJoyConButtonSets(t.SourceID, t.down, next, at, false)
	t.down = next
	return events
}

func (t *JoyConStateTracker) Disconnect(at time.Time) []InputEvent {
	if t.down == nil {
		return nil
	}
	events := diffJoyConButtonSets(t.SourceID, t.down, map[JoyConButton]bool{}, at, true)
	t.down = make(map[JoyConButton]bool, 16)
	t.directions = newJoyConDirectionState()
	return events
}

func diffJoyConButtonSets(sourceID string, previous, next map[JoyConButton]bool, at time.Time, synthetic bool) []InputEvent {
	all := make(map[JoyConButton]struct{}, len(previous)+len(next))
	for button := range previous {
		all[button] = struct{}{}
	}
	for button := range next {
		all[button] = struct{}{}
	}
	buttons := make([]string, 0, len(all))
	for button := range all {
		buttons = append(buttons, string(button))
	}
	sort.Strings(buttons)

	events := make([]InputEvent, 0, len(buttons))
	for _, code := range buttons {
		button := JoyConButton(code)
		wasDown := previous[button]
		isDown := next[button]
		if wasDown == isDown {
			continue
		}
		events = append(events, InputEvent{
			Token: InputToken{
				Kind:     "JoyCon",
				Code:     code,
				DeviceID: sourceID,
			},
			Down:       isDown,
			SourceID:   sourceID,
			OccurredAt: at,
			Synthetic:  synthetic,
		})
	}
	return events
}

type JoyConCalibrationSession struct {
	minX       uint16
	maxX       uint16
	minY       uint16
	maxY       uint16
	centerXSum uint64
	centerYSum uint64
	centerN    uint64
	samples    uint64
}

func newJoyConCalibrationSession() *JoyConCalibrationSession {
	return &JoyConCalibrationSession{minX: 4095, minY: 4095}
}

func (s *JoyConCalibrationSession) Add(rawX, rawY uint16, neutral bool) {
	if rawX > 4095 {
		rawX = 4095
	}
	if rawY > 4095 {
		rawY = 4095
	}
	if rawX < s.minX {
		s.minX = rawX
	}
	if rawX > s.maxX {
		s.maxX = rawX
	}
	if rawY < s.minY {
		s.minY = rawY
	}
	if rawY > s.maxY {
		s.maxY = rawY
	}
	if neutral {
		s.centerXSum += uint64(rawX)
		s.centerYSum += uint64(rawY)
		s.centerN++
	}
	s.samples++
}

func (s *JoyConCalibrationSession) Result() (JoyConStickCalibration, error) {
	if s == nil || s.samples < 10 {
		return JoyConStickCalibration{}, fmt.Errorf("Joy-Con calibration needs at least 10 samples")
	}
	if s.centerN == 0 {
		return JoyConStickCalibration{}, fmt.Errorf("Joy-Con calibration has no neutral samples")
	}
	centerX := uint16(s.centerXSum / s.centerN)
	centerY := uint16(s.centerYSum / s.centerN)
	cal := JoyConStickCalibration{
		X: JoyConAxisCalibration{Min: s.minX, Center: centerX, Max: s.maxX},
		Y: JoyConAxisCalibration{Min: s.minY, Center: centerY, Max: s.maxY},
	}
	if normalized := normalizeJoyConAxisCalibration(cal.X); normalized != cal.X {
		return JoyConStickCalibration{}, fmt.Errorf("Joy-Con X calibration range is incomplete")
	}
	if normalized := normalizeJoyConAxisCalibration(cal.Y); normalized != cal.Y {
		return JoyConStickCalibration{}, fmt.Errorf("Joy-Con Y calibration range is incomplete")
	}
	return cal, nil
}
