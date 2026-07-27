//go:build windows

package main

import (
	"testing"
	"time"
)

func newControllerInputTestApp(rules ...Rule) *App {
	return &App{
		rules:               rules,
		enabled:             true,
		mouseDown:           map[string]bool{},
		keyDown:             map[uint32]bool{},
		controllerDown:      map[string]bool{},
		controllerPending:   map[string]bool{},
		controllerConsumed:  map[string]bool{},
		controllerHoldRules: map[string]Rule{},
		pendingTap:          map[string]bool{},
		consumedPrefix:      map[string]bool{},
		suppressedDown:      map[string]bool{},
		longPress:           map[string]*longPressState{},
		actionCh:            make(chan outputJob, 8),
		recordHeld:          map[string]bool{},
		recordingRuleIndex:  -1,
	}
}

func xInputTestEvent(code string, down bool, synthetic bool) InputEvent {
	return InputEvent{
		Token:      InputToken{Kind: "XInput", Code: code, DeviceID: "P1"},
		Down:       down,
		SourceID:   "xinput:P1",
		OccurredAt: time.Unix(100, 0),
		Synthetic:  synthetic,
	}
}

func TestXInputSingleTapQueuesOnRelease(t *testing.T) {
	rule := Rule{Enabled: true, Input: []Item{{Kind: "XInput", Code: "P1:A"}}, Output: []Item{{Kind: "Key", Code: "R"}}}
	app := newControllerInputTestApp(rule)
	app.handleXInputInputEvent(xInputTestEvent("P1:A", true, false))
	select {
	case job := <-app.actionCh:
		t.Fatalf("single tap queued on DOWN: %+v", job.Rule)
	default:
	}
	app.handleXInputInputEvent(xInputTestEvent("P1:A", false, false))
	select {
	case job := <-app.actionCh:
		if !sameInput(job.Rule.Input[0], rule.Input[0]) {
			t.Fatalf("queued rule=%+v", job.Rule)
		}
	default:
		t.Fatal("single XInput tap was not queued on UP")
	}
}

func TestXInputMixedRuleConsumesControllerPrefix(t *testing.T) {
	rule := Rule{
		Enabled: true,
		Input:   []Item{{Kind: "XInput", Code: "P1:LB"}, {Kind: "Mouse", Code: "X1"}},
		Output:  []Item{{Kind: "Key", Code: "R"}},
	}
	app := newControllerInputTestApp(rule)
	app.handleXInputInputEvent(xInputTestEvent("P1:LB", true, false))
	app.mu.Lock()
	app.mouseDown["X1"] = true
	matched, ok := app.findBestTriggerLocked(Item{Kind: "Mouse", Code: "X1"})
	if ok {
		app.markPrefixesConsumedLocked(matched)
	}
	consumed := app.controllerConsumed["XInput:P1:LB"]
	app.mu.Unlock()
	if !ok || !consumed {
		t.Fatalf("matched=%v consumed=%v rule=%+v", ok, consumed, matched)
	}
	app.handleXInputInputEvent(xInputTestEvent("P1:LB", false, false))
	select {
	case job := <-app.actionCh:
		t.Fatalf("consumed prefix emitted single action: %+v", job.Rule)
	default:
	}
}

func TestXInputSyntheticDisconnectReleasesHold(t *testing.T) {
	rule := Rule{
		Enabled: true,
		Input:   []Item{{Kind: "XInput", Code: "P1:LStickUp"}},
		Mode:    joyConRuleModeHold,
		Output:  []Item{{Kind: "Key", Code: "W"}},
	}
	app := newControllerInputTestApp(rule)
	app.handleXInputInputEvent(xInputTestEvent("P1:LStickUp", true, false))
	first := <-app.actionCh
	if first.Rule.Mode != joyConRuleModeHoldDown {
		t.Fatalf("first phase=%q", first.Rule.Mode)
	}
	app.handleXInputInputEvent(xInputTestEvent("P1:LStickUp", false, true))
	second := <-app.actionCh
	if second.Rule.Mode != joyConRuleModeHoldUp {
		t.Fatalf("disconnect phase=%q", second.Rule.Mode)
	}
	if len(app.controllerDown) != 0 || len(app.controllerHoldRules) != 0 {
		t.Fatalf("controller state leaked: down=%v holds=%v", app.controllerDown, app.controllerHoldRules)
	}
}

func TestJoyConAndXInputCodesDoNotCollide(t *testing.T) {
	app := newControllerInputTestApp()
	app.handleJoyConInputEvent(InputEvent{Token: InputToken{Code: "L"}, Down: true, SourceID: "joycon:left"})
	app.handleXInputInputEvent(xInputTestEvent("P1:LB", true, false))
	if !app.controllerDown["JoyCon:L"] || !app.controllerDown["XInput:P1:LB"] || len(app.controllerDown) != 2 {
		t.Fatalf("controller state collision: %#v", app.controllerDown)
	}
}
