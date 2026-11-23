//go:build windows

package main

import (
	"testing"
	"time"
)

func newHoldTestApp(rules []Rule) *App {
	return &App{
		enabled:             true,
		rules:               rules,
		mouseDown:           map[string]bool{},
		mouseDownAt:         map[string]time.Time{},
		keyDown:             map[uint32]bool{},
		controllerDown:      map[string]bool{},
		controllerPending:   map[string]bool{},
		controllerConsumed:  map[string]bool{},
		controllerHoldRules: map[string]Rule{},
		activeHoldRules:     map[string]Rule{},
		heldMouseRefs:       map[string]int{},
		pendingTap:          map[string]bool{},
		consumedPrefix:      map[string]bool{},
		suppressedDown:      map[string]bool{},
		longPress:           map[string]*longPressState{},
		joyConOutputRefs:    map[uint32]joyConOutputReference{},
		actionCh:            make(chan outputJob, 16),
		shutdownCh:          make(chan struct{}),
		logCh:               make(chan string, 32),
	}
}

func TestMouseSideButtonHoldModeDownRepeatAndUp(t *testing.T) {
	rule := Rule{
		Enabled:         true,
		Input:           []Item{{Kind: "Mouse", Code: "X1"}},
		Mode:            joyConRuleModeHold,
		Output:          []Item{{Kind: "Key", Code: "69"}}, // E
		SuppressTrigger: true,
	}
	if err := validateJoyConHoldRule(rule); err != nil {
		t.Fatalf("X1 Hold rule must be valid: %v", err)
	}
	a := newHoldTestApp([]Rule{rule})

	if !a.buttonDown("X1") {
		t.Fatal("X1 Hold down should suppress trigger")
	}
	job := <-a.actionCh
	if job.Rule.Mode != joyConRuleModeHoldDown {
		t.Fatalf("first job mode=%q want %q", job.Rule.Mode, joyConRuleModeHoldDown)
	}

	// Repeated DOWN while held must not enqueue duplicate HoldDown.
	if !a.buttonDown("X1") {
		t.Fatal("X1 repeated down while held should stay suppressed")
	}
	select {
	case extra := <-a.actionCh:
		t.Fatalf("unexpected duplicate hold event: %+v", extra.Rule.Mode)
	default:
	}

	if !a.buttonUp("X1") {
		t.Fatal("X1 Hold up should suppress trigger")
	}
	upJob := <-a.actionCh
	if upJob.Rule.Mode != joyConRuleModeHoldUp {
		t.Fatalf("release job mode=%q want %q", upJob.Rule.Mode, joyConRuleModeHoldUp)
	}
	if len(a.activeHoldRules) != 0 {
		t.Fatalf("activeHoldRules leaked: %+v", a.activeHoldRules)
	}
}

func TestLogicalRightSideKeyHoldModeAndAutoRepeat(t *testing.T) {
	defer updateMouseAliasesFromSinoWealth(nil)
	btns, _ := sinoWealthPresetButtons("mbm-all-f13-f16")
	updateMouseAliasesFromSinoWealth(&SinoWealthConfigState{
		ProductName: "Ray Pawn+ (RM-3360LR)",
		Buttons:     btns,
	})

	parsed, err := parseItemsText("右サイド1", true, true)
	if err != nil || len(parsed) != 1 || parsed[0].Code != "124" {
		t.Fatalf("parseItemsText(右サイド1) = (%+v, %v), want Key:124", parsed, err)
	}
	if got := itemsText(parsed); got != "右サイド1" {
		t.Fatalf("itemsText(Key:124) = %q want 右サイド1", got)
	}
	if got := itemsText([]Item{{Kind: "Mouse", Code: "X1"}, {Kind: "Mouse", Code: "X2"}}); got != "左サイド1 + 左サイド2" {
		t.Fatalf("side button names = %q want 左サイド1 + 左サイド2", got)
	}

	rule := Rule{
		Enabled:         true,
		Input:           parsed,
		Mode:            joyConRuleModeHold,
		Output:          []Item{{Kind: "Mouse", Code: "Left"}},
		SuppressTrigger: true,
	}
	if err := validateJoyConHoldRule(rule); err != nil {
		t.Fatalf("RightSide1 -> Left Hold must be valid: %v", err)
	}
	a := newHoldTestApp([]Rule{rule})

	if !a.handleKeyEvent(0x7C, true) {
		t.Fatal("F13 (右サイド1) Hold down should suppress trigger")
	}
	if job := <-a.actionCh; job.Rule.Mode != joyConRuleModeHoldDown {
		t.Fatalf("got mode=%q want HoldDown", job.Rule.Mode)
	}
	// OS key auto-repeat while held must stay suppressed and not re-trigger.
	if !a.handleKeyEvent(0x7C, true) {
		t.Fatal("auto-repeat while held should remain suppressed")
	}
	select {
	case extra := <-a.actionCh:
		t.Fatalf("unexpected duplicate key hold event: %+v", extra.Rule.Mode)
	default:
	}
	if !a.handleKeyEvent(0x7C, false) {
		t.Fatal("F13 Hold up should suppress trigger")
	}
	if job := <-a.actionCh; job.Rule.Mode != joyConRuleModeHoldUp {
		t.Fatalf("got mode=%q want HoldUp", job.Rule.Mode)
	}
}
