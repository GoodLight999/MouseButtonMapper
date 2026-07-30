//go:build windows

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestControllerFeatureDefaultsOffAndMigratesConfigVersion(t *testing.T) {
	cfg := normalizeConfig(Config{
		Version:         9,
		ActiveProfileId: "profile",
		Profiles:        []Profile{{Id: "profile", Name: "Profile"}},
	})
	if cfg.Version != 11 {
		t.Fatalf("version=%d want 11", cfg.Version)
	}
	if cfg.Controller.Enabled || cfg.Controller.Visible {
		t.Fatal("controller feature and its UI must remain hidden opt-in for legacy configs")
	}

	var defaultCfg Config
	if err := json.Unmarshal([]byte(defaultConfigJSON), &defaultCfg); err != nil {
		t.Fatalf("default config: %v", err)
	}
	if defaultCfg.Controller.Enabled || defaultCfg.Controller.Visible {
		t.Fatal("embedded default exposes or enables experimental controller feature")
	}
}

func TestDisabledControllerFeatureExcludesOnlyControllerRules(t *testing.T) {
	mouseRule := Rule{Enabled: true, Input: []Item{{Kind: "Mouse", Code: "X1"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: "A"}}}
	joyRule := Rule{Enabled: true, Input: []Item{{Kind: "JoyCon", Code: "L"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: "B"}}}
	xInputRule := Rule{Enabled: true, Input: []Item{{Kind: "XInput", Code: "P1:LB"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: "C"}}}
	app := &App{
		config: Config{
			Version:         11,
			ActiveProfileId: "profile",
			Profiles:        []Profile{{Id: "profile", Name: "Profile", Rules: []Rule{mouseRule, joyRule, xInputRule}}},
		},
	}
	app.rebuildRulesWithoutJoyConRescanLocked()
	if len(app.rules) != 1 || !sameInput(app.rules[0].Input[0], mouseRule.Input[0]) {
		t.Fatalf("disabled active rules=%+v", app.rules)
	}

	app.config.Controller.Enabled = true
	app.rebuildRulesWithoutJoyConRescanLocked()
	if len(app.rules) != 3 {
		t.Fatalf("enabled active rule count=%d want 3", len(app.rules))
	}
}

func TestDisabledControllerFeatureIgnoresLateWorkerEvents(t *testing.T) {
	app := newControllerInputTestApp(Rule{
		Enabled: true,
		Input:   []Item{{Kind: "XInput", Code: "P1:A"}},
		Mode:    "Tap",
		Output:  []Item{{Kind: "Key", Code: "R"}},
	})
	app.config.Controller.Enabled = false
	app.handleXInputInputEvent(xInputTestEvent("P1:A", true, false))
	if len(app.controllerDown) != 0 || app.lastControllerInput.Kind != "" {
		t.Fatalf("disabled feature accepted input: down=%v last=%+v", app.controllerDown, app.lastControllerInput)
	}
	select {
	case job := <-app.actionCh:
		t.Fatalf("disabled feature queued output: %+v", job.Rule)
	default:
	}
}

func TestDisabledControllerFeatureDoesNotStartWorkers(t *testing.T) {
	app := &App{
		config:              Config{Controller: ControllerFeatureConfig{Enabled: false}},
		controllerDown:      map[string]bool{},
		controllerPending:   map[string]bool{},
		controllerConsumed:  map[string]bool{},
		controllerHoldRules: map[string]Rule{},
		longPress:           map[string]*longPressState{},
		joyConOutputRefs:    map[uint32]joyConOutputReference{},
	}
	app.startJoyConSubsystem()
	app.startXInputSubsystem()
	if app.joyConWorker != nil || app.xInputCancel != nil {
		t.Fatalf("workers started while disabled: joy=%p xinput=%v", app.joyConWorker, app.xInputCancel != nil)
	}
}

func TestControllerFeatureUIProvidesTrueHideAndBetterJoyStyleRegistration(t *testing.T) {
	for _, text := range []string{
		"controllerFeatureEnabled",
		"controllerHideUI",
		"experimentalFeatureRestore",
		"hide-ui",
		"BetterJoy",
		"全HIDインターフェース",
		"機能を停止して設定画面から隠す",
	} {
		if !strings.Contains(joyConUIJS, text) {
			t.Fatalf("controller feature UI missing %q", text)
		}
	}
}

func TestDisablingControllerFeaturePreservesStoredRulesAndMouseLongPress(t *testing.T) {
	mouseRule := Rule{Enabled: true, Input: []Item{{Kind: "Mouse", Code: "X1"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: "A"}}}
	controllerRule := Rule{Enabled: true, Input: []Item{{Kind: "XInput", Code: "P1:A"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: "B"}}}
	mouseTrigger := Item{Kind: "Mouse", Code: "X1"}
	controllerTrigger := Item{Kind: "XInput", Code: "P1:A"}
	app := &App{
		config: Config{
			Version:         11,
			ActiveProfileId: "profile",
			Controller:      ControllerFeatureConfig{Enabled: false},
			Profiles:        []Profile{{Id: "profile", Name: "Profile", Rules: []Rule{mouseRule, controllerRule}}},
		},
		controllerDown:      map[string]bool{controllerInputKey(controllerTrigger): true},
		controllerPending:   map[string]bool{},
		controllerConsumed:  map[string]bool{},
		controllerHoldRules: map[string]Rule{},
		longPress: map[string]*longPressState{
			longPressKey(mouseTrigger):      {Trigger: mouseTrigger},
			longPressKey(controllerTrigger): {Trigger: controllerTrigger},
		},
		logCh: make(chan string, 8),
	}

	app.clearControllerInputStateLocked("controller feature disabled")
	if app.longPress[longPressKey(mouseTrigger)] == nil {
		t.Fatal("disabling controllers removed unrelated mouse long-press state")
	}
	if app.longPress[longPressKey(controllerTrigger)] != nil {
		t.Fatal("controller long-press state survived controller disable")
	}
	if got := len(app.config.Profiles[0].Rules); got != 2 {
		t.Fatalf("stored rules changed: got %d want 2", got)
	}
	app.rebuildRulesWithoutJoyConRescanLocked()
	if got := len(app.rules); got != 1 || !sameInput(app.rules[0].Input[0], mouseTrigger) {
		t.Fatalf("active rules after disable=%+v", app.rules)
	}
}

func TestXInputLongPressUsesStableControllerKey(t *testing.T) {
	trigger := Item{Kind: "XInput", Code: "P1:LB"}
	if got := longPressKey(trigger); got != "xinput:p1:lb" {
		t.Fatalf("longPressKey=%q", got)
	}
	if !isHoldableLongPressTrigger(trigger) {
		t.Fatal("known XInput button is not accepted as a long-press trigger")
	}
}

func TestControllerVisibilityMigrationPreservesEnabledOnly(t *testing.T) {
	disabled := normalizeConfig(Config{Version: 10, ActiveProfileId: "p", Profiles: []Profile{{Id: "p", Name: "P"}}})
	if disabled.Controller.Enabled || disabled.Controller.Visible {
		t.Fatalf("disabled v10 migration=%+v", disabled.Controller)
	}
	enabled := normalizeConfig(Config{Version: 10, ActiveProfileId: "p", Controller: ControllerFeatureConfig{Enabled: true}, Profiles: []Profile{{Id: "p", Name: "P"}}})
	if !enabled.Controller.Enabled || !enabled.Controller.Visible {
		t.Fatalf("enabled v10 migration=%+v", enabled.Controller)
	}
}
