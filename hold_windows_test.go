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
		suppressedKeyDown:   map[uint32]bool{},
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

func TestHoldModeSurvivesReconcilePhysicalStateWithoutDoubleFire(t *testing.T) {
	rule := Rule{
		Enabled:         true,
		Input:           []Item{{Kind: "Key", Code: "131"}}, // F20 (DPI-ボタン)
		Mode:            joyConRuleModeHold,
		Output:          []Item{{Kind: "Key", Code: "91"}, {Kind: "Key", Code: "72"}}, // Win + H
		SuppressTrigger: true,
	}
	a := newHoldTestApp([]Rule{rule})

	if !a.handleKeyEvent(0x83, true) {
		t.Fatal("F20 Hold down should suppress trigger")
	}
	if job := <-a.actionCh; job.Rule.Mode != joyConRuleModeHoldDown {
		t.Fatalf("got mode=%q want HoldDown", job.Rule.Mode)
	}

	// Simulate AutoSwitch calling reconcilePhysicalStateLocked while F20 is suppressed.
	// GetAsyncKeyState(F20) is 0 because WH_KEYBOARD_LL suppressed it; reconcile must NOT drop keyDown.
	a.mu.Lock()
	a.reconcilePhysicalStateLocked()
	stillDown := a.keyDown[0x83]
	a.mu.Unlock()
	if !stillDown {
		t.Fatal("reconcilePhysicalStateLocked evicted active Hold keyDown")
	}

	// Even if keyDown were somehow cleared mid-hold, repeated DOWN must stay suppressed and never double-fire HoldDown.
	a.mu.Lock()
	delete(a.keyDown, 0x83)
	a.mu.Unlock()
	if !a.handleKeyEvent(0x83, true) {
		t.Fatal("F20 repeat while Hold active must stay suppressed")
	}
	select {
	case extra := <-a.actionCh:
		t.Fatalf("unexpected duplicate HoldDown after reconcile: %+v", extra.Rule.Mode)
	default:
	}

	if !a.handleKeyEvent(0x83, false) {
		t.Fatal("F20 Hold up should suppress trigger")
	}
	if job := <-a.actionCh; job.Rule.Mode != joyConRuleModeHoldUp {
		t.Fatalf("got mode=%q want HoldUp", job.Rule.Mode)
	}
}

func TestTapKeyRuleSuppressesRepeatAndKeyUp(t *testing.T) {
	rule := Rule{
		Enabled:         true,
		Input:           []Item{{Kind: "Key", Code: "130"}}, // F19 (DPI+ボタン)
		Mode:            "Tap",
		Output:          []Item{{Kind: "Key", Code: "91"}, {Kind: "Key", Code: "86"}}, // Win + V
		SuppressTrigger: true,
	}
	a := newHoldTestApp([]Rule{rule})

	if !a.handleKeyEvent(0x82, true) {
		t.Fatal("Tap key down should suppress trigger")
	}
	if job := <-a.actionCh; job.Rule.Mode != "Tap" {
		t.Fatalf("got mode=%q want Tap", job.Rule.Mode)
	}
	// Auto-repeat while held must stay suppressed and not re-fire Tap.
	if !a.handleKeyEvent(0x82, true) {
		t.Fatal("Tap key auto-repeat must stay suppressed")
	}
	select {
	case extra := <-a.actionCh:
		t.Fatalf("unexpected duplicate Tap event on repeat: %+v", extra.Rule.Mode)
	default:
	}
	// Matching KEYUP must also be suppressed.
	if !a.handleKeyEvent(0x82, false) {
		t.Fatal("Tap key up must stay suppressed")
	}
}

func TestSuppressedMouseChordPrefixSurvivesReconcileAcrossVirtualDesktopSwitch(t *testing.T) {
	rules := []Rule{
		{
			Enabled:         true,
			Input:           []Item{{Kind: "Mouse", Code: "X2"}},
			Mode:            "Tap",
			Output:          []Item{{Kind: "Key", Code: "17"}, {Kind: "Key", Code: "67"}}, // Ctrl+C
			SuppressTrigger: true,
		},
		{
			Enabled:         true,
			Input:           []Item{{Kind: "Mouse", Code: "X2"}, {Kind: "Mouse", Code: "WheelDown"}},
			Mode:            "Tap",
			Output:          []Item{{Kind: "Key", Code: "17"}, {Kind: "Key", Code: "91"}, {Kind: "Key", Code: "39"}}, // Ctrl+Win+Right
			SuppressTrigger: true,
			SuppressPrefix:  true,
		},
		{
			Enabled:         true,
			Input:           []Item{{Kind: "Mouse", Code: "X2"}, {Kind: "Mouse", Code: "WheelUp"}},
			Mode:            "Tap",
			Output:          []Item{{Kind: "Key", Code: "17"}, {Kind: "Key", Code: "91"}, {Kind: "Key", Code: "37"}}, // Ctrl+Win+Left
			SuppressTrigger: true,
			SuppressPrefix:  true,
		},
	}
	a := newHoldTestApp(rules)

	// Hold X2 and scroll down to switch Virtual Desktop.
	if !a.buttonDown("X2") {
		t.Fatal("X2 down should be suppressed")
	}
	if !a.wheel("WheelDown") {
		t.Fatal("X2+WheelDown should match and suppress wheel")
	}
	if job := <-a.actionCh; job.Rule.Output[2].Code != "39" {
		t.Fatalf("expected WheelDown rule (Right=39), got %+v", job.Rule.Output)
	}

	// Simulate foreground change on the destination desktop triggering reconcilePhysicalStateLocked.
	// GetAsyncKeyState(VK_XBUTTON2) is 0 because X2 was suppressed by WH_MOUSE_LL; X2 must stay held.
	a.mu.Lock()
	a.reconcilePhysicalStateLocked()
	stillDown := a.mouseDown["X2"]
	a.mu.Unlock()
	if !stillDown {
		t.Fatal("reconcilePhysicalStateLocked dropped suppressed X2 mid-chord")
	}

	// Scrolling back up while still holding X2 must immediately fire X2+WheelUp.
	if !a.wheel("WheelUp") {
		t.Fatal("X2+WheelUp should still match while holding X2 after desktop switch")
	}
	if job := <-a.actionCh; job.Rule.Output[2].Code != "37" {
		t.Fatalf("expected WheelUp rule (Left=37), got %+v", job.Rule.Output)
	}

	// Releasing X2 after chord consumption must not fire single-tap Ctrl+C.
	if !a.buttonUp("X2") {
		t.Fatal("X2 up should stay suppressed after chord")
	}
	select {
	case extra := <-a.actionCh:
		t.Fatalf("unexpected single-tap output after chord release: %+v", extra.Rule.Output)
	default:
	}
}
