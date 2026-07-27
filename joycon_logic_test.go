package main

import (
	"math"
	"testing"
	"time"
)

func TestParseJoyConFullReport(t *testing.T) {
	report := make([]byte, 12)
	report[0] = joyConReportFull
	report[2] = 0x90 // full and charging
	report[4] = (1 << 0) | (1 << 3) | (1 << 5)
	report[5] = 0xff
	packJoyConLeftStick(report, 3500, 500)

	state, err := parseJoyConInputReport(report)
	if err != nil {
		t.Fatalf("parseJoyConInputReport: %v", err)
	}
	if state.ReportID != joyConReportFull {
		t.Fatalf("ReportID=%#x", state.ReportID)
	}
	if state.StickX != 3500 || state.StickY != 500 {
		t.Fatalf("stick=(%d,%d)", state.StickX, state.StickY)
	}
	if state.BatteryPercent != 100 || !state.Charging {
		t.Fatalf("battery=%d charging=%v", state.BatteryPercent, state.Charging)
	}
	for _, button := range joyConLeftPhysicalButtons {
		if !state.Buttons[button] {
			t.Errorf("button %s was not decoded", button)
		}
	}
}

func TestParseJoyConSubcommandReplyUsesStandardState(t *testing.T) {
	report := make([]byte, 12)
	report[0] = joyConReportSubcommandReply
	report[5] = 1 << 7
	packJoyConLeftStick(report, 2000, 2000)

	state, err := parseJoyConInputReport(report)
	if err != nil {
		t.Fatalf("parseJoyConInputReport: %v", err)
	}
	if !state.Buttons[JoyConButtonZL] {
		t.Fatal("ZL was not decoded from subcommand reply")
	}
}

func TestParseJoyConSimpleReportSeparatesButtonsAndStickHat(t *testing.T) {
	report := []byte{
		joyConReportSimple,
		(1 << 0) | (1 << 4) | (1 << 5),
		(1 << 0) | (1 << 2) | (1 << 5) | (1 << 6) | (1 << 7),
		1,
	}
	state, err := parseJoyConInputReport(report)
	if err != nil {
		t.Fatalf("parseJoyConInputReport: %v", err)
	}
	for _, button := range []JoyConButton{
		JoyConButtonLeft, JoyConButtonSL, JoyConButtonSR,
		JoyConButtonMinus, JoyConButtonStick, JoyConButtonCapture, JoyConButtonL, JoyConButtonZL,
		JoyConStickUp, JoyConStickRight,
	} {
		if !state.Buttons[button] {
			t.Errorf("simple report did not decode %s: %#v", button, state.Buttons)
		}
	}
	if state.Buttons[JoyConButtonUp] || state.Buttons[JoyConButtonRight] {
		t.Fatalf("stick hat leaked into physical direction buttons: %#v", state.Buttons)
	}
	if state.BatteryPercent != -1 {
		t.Fatalf("simple report battery=%d", state.BatteryPercent)
	}
}

func TestParseJoyConReportErrors(t *testing.T) {
	cases := [][]byte{
		nil,
		{joyConReportFull, 0},
		{joyConReportSimple, 0},
		{0x99, 0, 0, 0},
	}
	for _, report := range cases {
		if _, err := parseJoyConInputReport(report); err == nil {
			t.Errorf("report %#v unexpectedly succeeded", report)
		}
	}
}

func TestDiffJoyConButtonSetsReleasesBeforePresses(t *testing.T) {
	at := time.Unix(123, 0)
	events := diffJoyConButtonSets(
		"left-a",
		map[JoyConButton]bool{JoyConStickLeft: true},
		map[JoyConButton]bool{JoyConStickRight: true},
		at,
		false,
	)
	if len(events) != 2 {
		t.Fatalf("events=%#v", events)
	}
	if events[0].Token.Code != string(JoyConStickLeft) || events[0].Down {
		t.Fatalf("first event=%+v, want StickLeft UP", events[0])
	}
	if events[1].Token.Code != string(JoyConStickRight) || !events[1].Down {
		t.Fatalf("second event=%+v, want StickRight DOWN", events[1])
	}
}

func TestDiffJoyConButtonSetsReleasesBeforeAlphabeticallyEarlierPress(t *testing.T) {
	at := time.Unix(124, 0)
	events := diffJoyConButtonSets(
		"left-a",
		map[JoyConButton]bool{JoyConButtonZL: true},
		map[JoyConButton]bool{JoyConButtonL: true},
		at,
		false,
	)
	if len(events) != 2 || events[0].Down || !events[1].Down {
		t.Fatalf("events=%#v", events)
	}
	if events[0].Token.Code != string(JoyConButtonZL) || events[1].Token.Code != string(JoyConButtonL) {
		t.Fatalf("release/press order=%#v", events)
	}
}

func TestNormalizeJoyConAxis(t *testing.T) {
	cal := JoyConAxisCalibration{Min: 500, Center: 2000, Max: 3500}
	checks := []struct {
		raw  uint16
		want float64
	}{
		{500, -1},
		{1250, -0.5},
		{2000, 0},
		{2750, 0.5},
		{3500, 1},
		{4095, 1},
	}
	for _, check := range checks {
		got := normalizeJoyConAxis(check.raw, cal)
		if math.Abs(got-check.want) > 0.0001 {
			t.Errorf("raw=%d got=%f want=%f", check.raw, got, check.want)
		}
	}
}

func TestNormalizeJoyConStickInversion(t *testing.T) {
	config := defaultJoyConStickConfig()
	config.InvertX = true
	config.InvertY = true
	x, y := normalizeJoyConStick(3500, 500, config)
	if x != -1 || y != 1 {
		t.Fatalf("inverted stick=(%f,%f)", x, y)
	}
}

func TestNormalizeJoyConStickConfigRepairsInvalidValues(t *testing.T) {
	config := normalizeJoyConStickConfig(JoyConStickConfig{
		DeadZone:      2,
		ReleaseZone:   3,
		DirectionMode: "invalid",
		Calibration: JoyConStickCalibration{
			X: JoyConAxisCalibration{Min: 1000, Center: 900, Max: 800},
			Y: JoyConAxisCalibration{},
		},
	})
	defaults := defaultJoyConStickConfig()
	if config.DeadZone != defaults.DeadZone || config.ReleaseZone != defaults.ReleaseZone {
		t.Fatalf("zones=%f/%f", config.DeadZone, config.ReleaseZone)
	}
	if config.DirectionMode != joyConDirectionMode8 {
		t.Fatalf("DirectionMode=%q", config.DirectionMode)
	}
	if config.Calibration.X != defaults.Calibration.X || config.Calibration.Y != defaults.Calibration.Y {
		t.Fatalf("calibration=%+v", config.Calibration)
	}
}

func TestJoyConStateTrackerSuppressesDuplicateDownAndRecoversMissingUp(t *testing.T) {
	tracker := newJoyConStateTracker("left-1")
	config := defaultJoyConStickConfig()
	at := time.Unix(100, 0)
	down := neutralJoyConRawState()
	down.Buttons[JoyConButtonZL] = true

	events := tracker.Apply(down, config, at)
	assertJoyConEvent(t, events, JoyConButtonZL, true, false)
	if duplicate := tracker.Apply(down, config, at.Add(time.Millisecond)); len(duplicate) != 0 {
		t.Fatalf("duplicate report produced events: %#v", duplicate)
	}

	up := neutralJoyConRawState()
	events = tracker.Apply(up, config, at.Add(2*time.Millisecond))
	assertJoyConEvent(t, events, JoyConButtonZL, false, false)
}

func TestJoyConStateTrackerDisconnectReleasesEveryInput(t *testing.T) {
	tracker := newJoyConStateTracker("left-1")
	config := defaultJoyConStickConfig()
	raw := neutralJoyConRawState()
	raw.Buttons[JoyConButtonL] = true
	raw.Buttons[JoyConButtonZL] = true
	raw.StickX = 3500
	tracker.Apply(raw, config, time.Unix(100, 0))

	events := tracker.Disconnect(time.Unix(101, 0))
	for _, button := range []JoyConButton{JoyConButtonL, JoyConButtonZL, JoyConStickRight} {
		assertJoyConEvent(t, events, button, false, true)
	}
	if again := tracker.Disconnect(time.Unix(102, 0)); len(again) != 0 {
		t.Fatalf("second disconnect produced events: %#v", again)
	}
}

func TestJoyConStickEightWayAndHysteresis(t *testing.T) {
	tracker := newJoyConStateTracker("left-1")
	config := defaultJoyConStickConfig()
	at := time.Unix(100, 0)

	neutral := neutralJoyConRawState()
	if events := tracker.Apply(neutral, config, at); len(events) != 0 {
		t.Fatalf("neutral produced events: %#v", events)
	}

	raw := neutralJoyConRawState()
	raw.StickX = 2500 // 0.333, above press threshold
	events := tracker.Apply(raw, config, at.Add(time.Millisecond))
	assertJoyConEvent(t, events, JoyConStickRight, true, false)

	raw.StickX = 2350 // 0.233, below press but above release threshold
	if events := tracker.Apply(raw, config, at.Add(2*time.Millisecond)); len(events) != 0 {
		t.Fatalf("hysteresis released too early: %#v", events)
	}

	raw.StickX = 2250 // 0.167, below release threshold
	events = tracker.Apply(raw, config, at.Add(3*time.Millisecond))
	assertJoyConEvent(t, events, JoyConStickRight, false, false)

	raw.StickX = 3000
	raw.StickY = 3000
	events = tracker.Apply(raw, config, at.Add(4*time.Millisecond))
	assertJoyConEvent(t, events, JoyConStickRight, true, false)
	assertJoyConEvent(t, events, JoyConStickUp, true, false)
}

func TestJoyConStickFourWayKeepsDominantAxisNearDiagonal(t *testing.T) {
	tracker := newJoyConStateTracker("left-1")
	config := defaultJoyConStickConfig()
	config.DirectionMode = joyConDirectionMode4
	at := time.Unix(100, 0)

	raw := neutralJoyConRawState()
	raw.StickX = 2900 // 0.60
	raw.StickY = 2750 // 0.50
	events := tracker.Apply(raw, config, at)
	assertJoyConEvent(t, events, JoyConStickRight, true, false)
	if hasJoyConEvent(events, JoyConStickUp, true) {
		t.Fatal("4-way mode emitted a diagonal")
	}

	raw.StickX = 2750 // 0.50
	raw.StickY = 2825 // 0.55, not enough to switch axis
	if events := tracker.Apply(raw, config, at.Add(time.Millisecond)); len(events) != 0 {
		t.Fatalf("near diagonal flipped axis: %#v", events)
	}

	raw.StickX = 2450 // 0.30
	raw.StickY = 3200 // 0.80, clearly stronger
	events = tracker.Apply(raw, config, at.Add(2*time.Millisecond))
	assertJoyConEvent(t, events, JoyConStickRight, false, false)
	assertJoyConEvent(t, events, JoyConStickUp, true, false)
}

func TestJoyConCalibrationSession(t *testing.T) {
	session := newJoyConCalibrationSession()
	for i := 0; i < 5; i++ {
		session.Add(2000, 2010, true)
	}
	for i := 0; i < 5; i++ {
		session.Add(500, 550, false)
		session.Add(3500, 3450, false)
	}
	calibration, err := session.Result()
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if calibration.X.Min != 500 || calibration.X.Center != 2000 || calibration.X.Max != 3500 {
		t.Fatalf("X calibration=%+v", calibration.X)
	}
	if calibration.Y.Min != 550 || calibration.Y.Center != 2010 || calibration.Y.Max != 3450 {
		t.Fatalf("Y calibration=%+v", calibration.Y)
	}
}

func TestJoyConCalibrationRejectsIncompleteRange(t *testing.T) {
	session := newJoyConCalibrationSession()
	for i := 0; i < 12; i++ {
		session.Add(2000, 2000, true)
	}
	if _, err := session.Result(); err == nil {
		t.Fatal("incomplete range unexpectedly succeeded")
	}
}

func TestInputTokenKeyNormalizesJoyConIdentity(t *testing.T) {
	token := InputToken{Kind: " joycon ", Code: " ZL ", DeviceID: " LEFT-1 "}
	if got, want := token.Key(), "joycon:left-1:zl"; got != want {
		t.Fatalf("Key=%q want=%q", got, want)
	}
}

func packJoyConLeftStick(report []byte, x, y uint16) {
	report[6] = byte(x)
	report[7] = byte((x>>8)&0x0f) | byte((y&0x0f)<<4)
	report[8] = byte(y >> 4)
}

func neutralJoyConRawState() JoyConRawState {
	return JoyConRawState{
		ReportID:       joyConReportFull,
		Buttons:        map[JoyConButton]bool{},
		StickX:         2000,
		StickY:         2000,
		BatteryPercent: 100,
	}
}

func assertJoyConEvent(t *testing.T, events []InputEvent, button JoyConButton, down, synthetic bool) {
	t.Helper()
	for _, event := range events {
		if JoyConButton(event.Token.Code) == button && event.Down == down {
			if event.Synthetic != synthetic {
				t.Fatalf("event %s synthetic=%v want=%v", button, event.Synthetic, synthetic)
			}
			return
		}
	}
	t.Fatalf("event button=%s down=%v not found in %#v", button, down, events)
}

func hasJoyConEvent(events []InputEvent, button JoyConButton, down bool) bool {
	for _, event := range events {
		if JoyConButton(event.Token.Code) == button && event.Down == down {
			return true
		}
	}
	return false
}

func TestParseJoyConInputOnlyReport(t *testing.T) {
	report := []byte{
		(1 << 4) | (1 << 5) | (1 << 6) | (1 << 7),
		(1 << 0) | (1 << 2) | (1 << 5),
		1,
		255, 0,
		128, 128,
	}
	state, err := parseJoyConInputOnlyReport(report)
	if err != nil {
		t.Fatalf("parseJoyConInputOnlyReport: %v", err)
	}
	for _, button := range []JoyConButton{
		JoyConButtonL, JoyConButtonSR, JoyConButtonZL, JoyConButtonSL,
		JoyConButtonMinus, JoyConButtonStick, JoyConButtonCapture,
		JoyConButtonUp, JoyConButtonRight,
	} {
		if !state.Buttons[button] {
			t.Errorf("input-only report did not decode %s: %#v", button, state.Buttons)
		}
	}
	if state.StickX != 4095 || state.StickY != 4095 {
		t.Fatalf("input-only stick=(%d,%d)", state.StickX, state.StickY)
	}
}

func TestParseJoyConInputOnlyReportAcceptsLeadingZeroReportID(t *testing.T) {
	report := []byte{0, 0, 0, 8, 128, 128, 128, 128}
	state, err := parseJoyConInputOnlyReport(report)
	if err != nil {
		t.Fatalf("parseJoyConInputOnlyReport: %v", err)
	}
	if len(state.Buttons) != 0 {
		t.Fatalf("unexpected buttons=%#v", state.Buttons)
	}
	if state.StickX < 2030 || state.StickX > 2070 || state.StickY < 2030 || state.StickY > 2070 {
		t.Fatalf("center stick=(%d,%d)", state.StickX, state.StickY)
	}
}

func TestParseJoyConInputOnlyReportRejectsLongUnknownPackets(t *testing.T) {
	report := make([]byte, 64)
	report[0] = 0x01
	if _, err := parseJoyConInputOnlyReport(report); err == nil {
		t.Fatal("long arbitrary HID packet was misclassified as Switch input-only")
	}
}
