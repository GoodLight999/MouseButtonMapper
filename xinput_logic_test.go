package main

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeXInputCode(t *testing.T) {
	cases := map[string]string{
		"A":                "P1:A",
		"P2/LB":            "P2:LB",
		"Pad4:LeftStickUp": "P4:LStickUp",
		"P1:十字左":           "P1:DPadLeft",
		"p3:r3":            "P3:RStickPress",
	}
	for input, want := range cases {
		if got := normalizeXInputCode(input); got != want {
			t.Errorf("normalizeXInputCode(%q)=%q want %q", input, got, want)
		}
	}
}

func TestXInputTrackerButtonsTriggersAndDirections(t *testing.T) {
	tracker := newXInputStateTracker(0)
	state := xInputGamepadState{
		Buttons:     0x0001 | 0x0100 | 0x1000,
		LeftTrigger: 40,
		LeftX:       25000,
		LeftY:       -25000,
	}
	events := tracker.Update(state, time.Unix(1, 0))
	down := map[string]bool{}
	for _, event := range events {
		if event.Down {
			down[event.Token.Code] = true
		}
	}
	for _, code := range []string{"P1:DPadUp", "P1:LB", "P1:A", "P1:LT", "P1:LStickRight", "P1:LStickDown"} {
		if !down[code] {
			t.Errorf("missing down event %s in %#v", code, events)
		}
	}
}

func TestXInputTrackerUsesHysteresis(t *testing.T) {
	tracker := newXInputStateTracker(0)
	tracker.Update(xInputGamepadState{LeftX: 22000, LeftTrigger: 40}, time.Unix(1, 0))
	if events := tracker.Update(xInputGamepadState{LeftX: 15000, LeftTrigger: 25}, time.Unix(2, 0)); len(events) != 0 {
		t.Fatalf("hysteresis released too early: %#v", events)
	}
	events := tracker.Update(xInputGamepadState{LeftX: 10000, LeftTrigger: 10}, time.Unix(3, 0))
	released := map[string]bool{}
	for _, event := range events {
		if !event.Down {
			released[event.Token.Code] = true
		}
	}
	if !released["P1:LStickRight"] || !released["P1:LT"] {
		t.Fatalf("expected releases after crossing release thresholds: %#v", events)
	}
}

func TestXInputDiffReleasesBeforePresses(t *testing.T) {
	events := diffXInputControls(0,
		map[string]bool{xInputLStickLeft: true},
		map[string]bool{xInputLStickRight: true},
		time.Unix(1, 0), false)
	if len(events) != 2 || events[0].Down || !events[1].Down {
		t.Fatalf("events not release-before-press: %#v", events)
	}
}

func TestXInputDisconnectSynthesizesUps(t *testing.T) {
	tracker := newXInputStateTracker(1)
	tracker.Update(xInputGamepadState{Buttons: 0x1000}, time.Unix(1, 0))
	events := tracker.Disconnect(time.Unix(2, 0))
	if len(events) != 1 || events[0].Down || !events[0].Synthetic || events[0].Token.Code != "P2:A" {
		t.Fatalf("disconnect events=%#v", events)
	}
}

func TestXInputTextRoundTripsThroughRuleParserFormat(t *testing.T) {
	for _, code := range []string{"P1:LB", "P2:DPadUp", "P4:LStickLeft"} {
		text := "XInput " + xInputCodeText(code)
		parsed := strings.TrimPrefix(text, "XInput ")
		if got := normalizeXInputCode(parsed); got != code {
			t.Fatalf("round trip %q -> %q; want %q", text, got, code)
		}
	}
}
