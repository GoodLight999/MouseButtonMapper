//go:build windows

package main

import (
	"testing"
	"time"
)

func testLongPressRule(action string) Rule {
	return Rule{
		Enabled:          true,
		Input:            []Item{{Kind: "Mouse", Code: "X1"}},
		Mode:             "Tap",
		Output:           []Item{{Kind: "Key", Code: "65"}},
		SuppressTrigger:  true,
		LongPressEnabled: true,
		LongPressMs:      5000,
		LongPressAction:  action,
		LongPressOutput:  []Item{{Kind: "Key", Code: "66"}},
	}
}

func newLongPressTestApp() *App {
	return &App{
		enabled:    true,
		mouseDown:  map[string]bool{"X1": true},
		keyDown:    map[uint32]bool{},
		longPress:  map[string]*longPressState{},
		logCh:      make(chan string, 16),
		actionCh:   make(chan outputJob, 16),
		shutdownCh: make(chan struct{}),
	}
}

func TestValidateLongPressRuleRejectsWheelAndPrimaryButtons(t *testing.T) {
	wheel := testLongPressRule(longPressActionExecute)
	wheel.Input = []Item{{Kind: "Mouse", Code: "WheelUp"}}
	if err := validateLongPressRule(wheel); err == nil {
		t.Fatal("wheel must not be accepted as a holdable trigger")
	}
	primary := testLongPressRule(longPressActionExecute)
	primary.Input = []Item{{Kind: "Mouse", Code: "Right"}}
	if err := validateLongPressRule(primary); err == nil {
		t.Fatal("primary mouse buttons must not be accepted for long press")
	}
	modifier := testLongPressRule(longPressActionExecute)
	modifier.Input = []Item{{Kind: "Key", Code: "17"}}
	if err := validateLongPressRule(modifier); err == nil {
		t.Fatal("modifier keys must not be accepted for long press")
	}
}

func TestFinishLongPressBeforeThresholdUsesShortOutput(t *testing.T) {
	a := newLongPressTestApp()
	r := testLongPressRule(longPressActionExecute)
	trigger := r.Input[0]
	a.mu.Lock()
	if !a.startLongPressLocked(r, trigger) {
		t.Fatal("failed to start long press")
	}
	result := a.finishLongPressLocked(trigger)
	a.mu.Unlock()
	if !result.Handled || !result.HasRule || result.Kind != "short" {
		t.Fatalf("unexpected completion: %+v", result)
	}
	if got := result.Rule.Output[0].Code; got != "65" {
		t.Fatalf("short output = %s, want 65", got)
	}
}

func TestFinishLongPressAfterThresholdUsesLongOutputOnce(t *testing.T) {
	a := newLongPressTestApp()
	r := testLongPressRule(longPressActionExecute)
	trigger := r.Input[0]
	a.mu.Lock()
	if !a.startLongPressLocked(r, trigger) {
		t.Fatal("failed to start long press")
	}
	state := a.longPress[longPressKey(trigger)]
	state.StartedAt = time.Now().Add(-6 * time.Second)
	result := a.finishLongPressLocked(trigger)
	a.mu.Unlock()
	if !result.Handled || !result.HasRule || result.Kind != "long" {
		t.Fatalf("unexpected completion: %+v", result)
	}
	if got := result.Rule.Output[0].Code; got != "66" {
		t.Fatalf("long output = %s, want 66", got)
	}
}

func TestLongPressCancelProducesNoOutput(t *testing.T) {
	a := newLongPressTestApp()
	r := testLongPressRule(longPressActionCancel)
	r.LongPressOutput = nil
	trigger := r.Input[0]
	a.mu.Lock()
	if !a.startLongPressLocked(r, trigger) {
		t.Fatal("failed to start long press")
	}
	state := a.longPress[longPressKey(trigger)]
	state.StartedAt = time.Now().Add(-6 * time.Second)
	result := a.finishLongPressLocked(trigger)
	a.mu.Unlock()
	if !result.Handled || result.HasRule || result.Kind != "long" {
		t.Fatalf("unexpected cancellation result: %+v", result)
	}
}

func TestFireLongPressQueuesLongOutputAndReleaseDoesNotRepeat(t *testing.T) {
	a := newLongPressTestApp()
	r := testLongPressRule(longPressActionExecute)
	trigger := r.Input[0]

	a.mu.Lock()
	if !a.startLongPressLocked(r, trigger) {
		a.mu.Unlock()
		t.Fatal("failed to start long press")
	}
	state := a.longPress[longPressKey(trigger)]
	token := state.Token
	a.mu.Unlock()

	a.fireLongPress(longPressKey(trigger), token)
	select {
	case job := <-a.actionCh:
		if got := job.Rule.Output[0].Code; got != "66" {
			t.Fatalf("queued long output = %s, want 66", got)
		}
	default:
		t.Fatal("long press did not queue an output")
	}

	a.mu.Lock()
	result := a.finishLongPressLocked(trigger)
	a.mu.Unlock()
	if !result.Handled || result.HasRule || result.Kind != "long" {
		t.Fatalf("release after fired long press must not execute again: %+v", result)
	}
	select {
	case extra := <-a.actionCh:
		t.Fatalf("unexpected second output after release: %+v", extra)
	default:
	}
}

func TestFireLongPressCancelSuppressesShortOutput(t *testing.T) {
	a := newLongPressTestApp()
	r := testLongPressRule(longPressActionCancel)
	r.LongPressOutput = nil
	trigger := r.Input[0]

	a.mu.Lock()
	if !a.startLongPressLocked(r, trigger) {
		a.mu.Unlock()
		t.Fatal("failed to start long press")
	}
	token := a.longPress[longPressKey(trigger)].Token
	a.mu.Unlock()

	a.fireLongPress(longPressKey(trigger), token)
	select {
	case job := <-a.actionCh:
		t.Fatalf("cancel action must not queue output: %+v", job)
	default:
	}

	a.mu.Lock()
	result := a.finishLongPressLocked(trigger)
	a.mu.Unlock()
	if !result.Handled || result.HasRule || result.Kind != "long" {
		t.Fatalf("cancelled release must remain output-free: %+v", result)
	}
}

func TestAbortLongPressPreventsShortAndLongOutput(t *testing.T) {
	a := newLongPressTestApp()
	r := testLongPressRule(longPressActionExecute)
	trigger := r.Input[0]

	a.mu.Lock()
	if !a.startLongPressLocked(r, trigger) {
		a.mu.Unlock()
		t.Fatal("failed to start long press")
	}
	a.abortLongPressForTriggerLocked(trigger, "test")
	result := a.finishLongPressLocked(trigger)
	a.mu.Unlock()

	if !result.Handled || result.HasRule || result.Kind != "aborted" {
		t.Fatalf("aborted long press must not execute: %+v", result)
	}
}

func TestLongPressOutputIsIgnoredAfterShutdownStarts(t *testing.T) {
	a := newLongPressTestApp()
	a.shuttingDown.Store(true)
	a.enqueueRuleGuaranteed(testLongPressRule(longPressActionExecute))
	select {
	case job := <-a.actionCh:
		t.Fatalf("shutdown must reject new output: %+v", job)
	default:
	}
}

func TestValidateLongOnlyRule(t *testing.T) {
	r := testLongPressRule(longPressActionExecute)
	r.Output = nil
	if err := validateLongPressRule(r); err != nil {
		t.Fatalf("long-only rule must be accepted: %v", err)
	}
}

func TestValidateCancelRequiresShortOutput(t *testing.T) {
	r := testLongPressRule(longPressActionCancel)
	r.Output = nil
	r.LongPressOutput = nil
	if err := validateLongPressRule(r); err == nil {
		t.Fatal("cancel without a short action must be rejected")
	}
}

func TestNormalizeConfigKeepsV8RulesCompatible(t *testing.T) {
	cfg := Config{
		Version:         8,
		ActiveProfileId: "default",
		Profiles: []Profile{{
			Id:   "default",
			Name: "既定",
			Rules: []Rule{{
				Enabled: true,
				Input:   []Item{{Kind: "Mouse", Code: "X1"}},
				Mode:    "Tap",
				Output:  []Item{{Kind: "Key", Code: "65"}},
			}},
		}},
	}

	got := normalizeConfig(cfg)
	if got.Version != 9 {
		t.Fatalf("config version = %d, want 9", got.Version)
	}
	r := got.Profiles[0].Rules[0]
	if r.LongPressEnabled {
		t.Fatal("an old rule must not gain long-press behavior during migration")
	}
	if len(r.Output) != 1 || r.Output[0].Code != "65" {
		t.Fatalf("old rule output changed during migration: %+v", r.Output)
	}
}

func TestNormalizeConfigNormalizesLongPressDefaults(t *testing.T) {
	cfg := Config{
		Version:         9,
		ActiveProfileId: "default",
		Profiles: []Profile{{
			Id:   "default",
			Name: "既定",
			Rules: []Rule{{
				Enabled:          true,
				Input:            []Item{{Kind: "Mouse", Code: "X1"}},
				Mode:             "Tap",
				Output:           []Item{{Kind: "Key", Code: "65"}},
				LongPressEnabled: true,
				LongPressMs:      0,
				LongPressAction:  "cancel",
				LongPressOutput:  []Item{{Kind: "Key", Code: "66"}},
			}},
		}},
	}

	r := normalizeConfig(cfg).Profiles[0].Rules[0]
	if r.LongPressMs != defaultLongPressMs {
		t.Fatalf("long press ms = %d, want %d", r.LongPressMs, defaultLongPressMs)
	}
	if r.LongPressAction != longPressActionCancel {
		t.Fatalf("long press action = %q, want Cancel", r.LongPressAction)
	}
	if len(r.LongPressOutput) != 0 {
		t.Fatalf("cancel action must discard stale long output: %+v", r.LongPressOutput)
	}
}
