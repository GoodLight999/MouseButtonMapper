//go:build windows

package main

import (
	"encoding/json"
	"strings"
	"testing"
	"unsafe"
)

func TestParseJoyConTokenRequiresExplicitPrefix(t *testing.T) {
	checks := map[string]JoyConButton{
		"Joy-Con ZL":          JoyConButtonZL,
		"JoyCon StickUp":      JoyConStickUp,
		"Joy-Con (L) Capture": JoyConButtonCapture,
		"Joy-Con スティック押込み":    JoyConButtonStick,
	}
	for text, want := range checks {
		got, ok := parseJoyConToken(text)
		if !ok || JoyConButton(got) != want {
			t.Errorf("parseJoyConToken(%q)=(%q,%v), want %q", text, got, ok, want)
		}
	}
	for _, text := range []string{"ZL", "Up", "Left", "Capture"} {
		if got, ok := parseJoyConToken(text); ok {
			t.Errorf("ambiguous token %q unexpectedly parsed as %q", text, got)
		}
	}
}

func TestParseItemsTextAcceptsMixedJoyConMouseAndKeyboardInput(t *testing.T) {
	items, err := parseItemsText("Joy-Con ZL + サイド1 + Ctrl", true, true)
	if err != nil {
		t.Fatalf("parseItemsText: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("items=%#v", items)
	}
	if !sameInput(items[0], Item{Kind: "JoyCon", Code: "ZL"}) {
		t.Fatalf("Joy-Con item=%+v", items[0])
	}
	if !sameInput(items[1], Item{Kind: "Mouse", Code: "X1"}) {
		t.Fatalf("mouse item=%+v", items[1])
	}
	if !sameInput(items[2], Item{Kind: "Key", Code: "Ctrl"}) {
		t.Fatalf("key item=%+v", items[2])
	}
}

func TestFindBestTriggerUsesMixedJoyConRules(t *testing.T) {
	comboMouse := Rule{
		Enabled: true,
		Input: []Item{
			{Kind: "JoyCon", Code: "ZL"},
			{Kind: "Mouse", Code: "X1"},
		},
		Output: []Item{{Kind: "Key", Code: "R"}},
	}
	comboKey := Rule{
		Enabled: true,
		Input: []Item{
			{Kind: "JoyCon", Code: "L"},
			{Kind: "Key", Code: "Ctrl"},
		},
		Output: []Item{{Kind: "Key", Code: "Space"}},
	}
	singleMouse := Rule{
		Enabled: true,
		Input:   []Item{{Kind: "Mouse", Code: "X1"}},
		Output:  []Item{{Kind: "Key", Code: "A"}},
	}
	app := &App{
		rules:           []Rule{singleMouse, comboMouse, comboKey},
		mouseDown:       map[string]bool{"X1": true},
		keyDown:         map[uint32]bool{},
		joyConDown:      map[string]bool{"ZL": true, "L": true},
		longPress:       map[string]*longPressState{},
		consumedPrefix:  map[string]bool{},
		joyConConsumed:  map[string]bool{},
		joyConHoldRules: map[string]Rule{},
	}

	got, ok := app.findBestTriggerLocked(Item{Kind: "Mouse", Code: "X1"})
	if !ok || len(got.Input) != 2 || !sameInput(got.Input[0], Item{Kind: "JoyCon", Code: "ZL"}) {
		t.Fatalf("mixed Joy-Con+mouse rule not selected: %+v ok=%v", got, ok)
	}

	app.keyDown[VK_CONTROL] = true
	got, ok = app.findBestTriggerLocked(Item{Kind: "Key", Code: "Ctrl"})
	if !ok || len(got.Input) != 2 || !sameInput(got.Input[0], Item{Kind: "JoyCon", Code: "L"}) {
		t.Fatalf("mixed Joy-Con+keyboard rule not selected: %+v ok=%v", got, ok)
	}
}

func TestPhysicalInputIdleIncludesJoyCon(t *testing.T) {
	app := &App{
		mouseDown:  map[string]bool{},
		keyDown:    map[uint32]bool{},
		joyConDown: map[string]bool{"ZL": true},
	}
	if app.physicalInputIdleLocked() {
		t.Fatal("Joy-Con DOWN was ignored by the profile-switch boundary")
	}
	delete(app.joyConDown, "ZL")
	if !app.physicalInputIdleLocked() {
		t.Fatal("idle state was not restored after Joy-Con UP")
	}
}

func TestJoyConHoldRuleValidation(t *testing.T) {
	valid := Rule{
		Enabled: true,
		Input:   []Item{{Kind: "JoyCon", Code: "StickUp"}},
		Mode:    joyConRuleModeHold,
		Output:  []Item{{Kind: "Key", Code: "W"}, {Kind: "Key", Code: "Shift"}},
	}
	if err := validateJoyConHoldRule(valid); err != nil {
		t.Fatalf("valid Hold rule rejected: %v", err)
	}

	cases := []Rule{
		{Input: []Item{{Kind: "Mouse", Code: "X1"}}, Mode: joyConRuleModeHold, Output: []Item{{Kind: "Key", Code: "W"}}},
		{Input: []Item{{Kind: "JoyCon", Code: "ZL"}, {Kind: "Mouse", Code: "X1"}}, Mode: joyConRuleModeHold, Output: []Item{{Kind: "Key", Code: "W"}}},
		{Input: []Item{{Kind: "JoyCon", Code: "ZL"}}, Mode: joyConRuleModeHold, Output: []Item{{Kind: "Mouse", Code: "X1"}}},
		{Input: []Item{{Kind: "JoyCon", Code: "ZL"}}, Mode: joyConRuleModeHold, LongPressEnabled: true, Output: []Item{{Kind: "Key", Code: "W"}}},
	}
	for i, rule := range cases {
		if err := validateJoyConHoldRule(rule); err == nil {
			t.Errorf("invalid Hold rule %d unexpectedly succeeded", i)
		}
	}
}

func TestJoyConLongPressKeyAndValidation(t *testing.T) {
	trigger := Item{Kind: "JoyCon", Code: "－"}
	if got, want := longPressKey(trigger), "joycon:minus"; got != want {
		t.Fatalf("longPressKey=%q want=%q", got, want)
	}
	rule := Rule{
		Input:            []Item{trigger},
		Output:           []Item{{Kind: "Key", Code: "Esc"}},
		LongPressEnabled: true,
		LongPressMs:      600,
		LongPressAction:  longPressActionCancel,
	}
	if err := validateLongPressRule(rule); err != nil {
		t.Fatalf("Joy-Con long-press rule rejected: %v", err)
	}
}

func TestValidateExecutableOutputItems(t *testing.T) {
	valid := []Item{
		{Kind: "Key", Code: "Ctrl"},
		{Kind: "Key", Code: "R"},
		{Kind: "Mouse", Code: "WheelUp"},
		{Kind: "Mouse", Code: "X1"},
	}
	if err := validateExecutableOutputItems(valid); err != nil {
		t.Fatalf("valid output rejected: %v", err)
	}
	if err := validateExecutableOutputItems([]Item{{Kind: "JoyCon", Code: "ZL"}}); err == nil {
		t.Fatal("Joy-Con output unexpectedly accepted")
	}
	if err := validateExecutableOutputItems([]Item{{Kind: "Mouse", Code: "Unknown"}}); err == nil {
		t.Fatal("unknown mouse output unexpectedly accepted")
	}
}

func TestJoyConMouseOutputEncoding(t *testing.T) {
	inputs, ok := joyConMouseTapInputs("WheelDown")
	if !ok || len(inputs) != 1 || inputs[0].Type != INPUT_MOUSE {
		t.Fatalf("WheelDown inputs=%#v ok=%v", inputs, ok)
	}
	mouse := *(*joyConMouseInput)(unsafe.Pointer(&inputs[0].Data[0]))
	if mouse.DwFlags != joyConMouseEventWheel || mouse.MouseData != joyConWheelData(-joyConWheelDelta) {
		t.Fatalf("wheel input=%+v", mouse)
	}
	if mouse.DwExtraInfo != extraInfoMarker {
		t.Fatalf("extraInfo=%x", mouse.DwExtraInfo)
	}

	inputs, ok = joyConMouseTapInputs("X1")
	if !ok || len(inputs) != 2 {
		t.Fatalf("X1 inputs=%#v ok=%v", inputs, ok)
	}
	down := *(*joyConMouseInput)(unsafe.Pointer(&inputs[0].Data[0]))
	up := *(*joyConMouseInput)(unsafe.Pointer(&inputs[1].Data[0]))
	if down.DwFlags != joyConMouseEventXDown || up.DwFlags != joyConMouseEventXUp || down.MouseData != 1 || up.MouseData != 1 {
		t.Fatalf("X1 down/up=%+v/%+v", down, up)
	}
}

func TestJoyConOutputPreservesPhysicalKeys(t *testing.T) {
	if joyConShouldInjectTapKey(true) {
		t.Fatal("tap output would inject and release a physically-held key")
	}
	if !joyConShouldInjectTapKey(false) {
		t.Fatal("tap output did not inject a free key")
	}

	owned := joyConOutputReference{Count: 1, Owned: true}
	if joyConShouldReleaseOwnedKey(owned, true) {
		t.Fatal("Hold output would release a key that became physically held")
	}
	if !joyConShouldReleaseOwnedKey(owned, false) {
		t.Fatal("Hold output did not release an owned key")
	}

	borrowed := joyConOutputReference{Count: 1, Owned: false}
	if joyConShouldReleaseOwnedKey(borrowed, false) {
		t.Fatal("Hold output would release a key it never pressed")
	}
}

func TestOldConfigWithoutJoyConLoadsAndRoundTrips(t *testing.T) {
	oldJSON := `{"Version":9,"ActiveProfileId":"p1","Profiles":[{"Id":"p1","Name":"旧設定","Rules":[{"Enabled":true,"Input":[{"Kind":"Mouse","Code":"X1"}],"Mode":"Tap","Output":[{"Kind":"Key","Code":"A"}]}]}]}`
	var cfg Config
	if err := json.Unmarshal([]byte(oldJSON), &cfg); err != nil {
		t.Fatalf("unmarshal old config: %v", err)
	}
	normalized := normalizeConfig(cfg)
	if len(normalized.Profiles) != 1 || len(normalized.Profiles[0].Rules) != 1 {
		t.Fatalf("old profile/rules were lost: %+v", normalized)
	}
	joy := normalized.Profiles[0].JoyCon
	defaults := defaultJoyConProfileConfig()
	if joy.Stick.DeadZone != defaults.Stick.DeadZone || joy.Stick.ReleaseZone != defaults.Stick.ReleaseZone {
		t.Fatalf("old config Joy-Con defaults=%+v", joy)
	}
	if joy.Enabled {
		t.Fatal("Joy-Con was enabled for an old config without explicit settings")
	}

	normalized.Profiles[0].JoyCon.Enabled = true
	normalized.Profiles[0].JoyCon.Stick.DirectionMode = joyConDirectionMode4
	normalized.Profiles[0].Rules = append(normalized.Profiles[0].Rules, Rule{
		Enabled: true,
		Input:   []Item{{Kind: "JoyCon", Code: "ZL"}},
		Mode:    joyConRuleModeHold,
		Output:  []Item{{Kind: "Key", Code: "Shift"}},
	})
	encoded, err := json.Marshal(normalized)
	if err != nil {
		t.Fatalf("marshal new config: %v", err)
	}
	var reloaded Config
	if err := json.Unmarshal(encoded, &reloaded); err != nil {
		t.Fatalf("unmarshal new config: %v", err)
	}
	reloaded = normalizeConfig(reloaded)
	if !reloaded.Profiles[0].JoyCon.Enabled || reloaded.Profiles[0].JoyCon.Stick.DirectionMode != joyConDirectionMode4 {
		t.Fatalf("Joy-Con settings did not round-trip: %+v", reloaded.Profiles[0].JoyCon)
	}
	if len(reloaded.Profiles[0].Rules) != 2 || reloaded.Profiles[0].Rules[1].Mode != joyConRuleModeHold {
		t.Fatalf("Joy-Con rule did not round-trip: %+v", reloaded.Profiles[0].Rules)
	}
}

func TestJoyConUIIsHookedIntoEmbeddedWebApp(t *testing.T) {
	if !strings.Contains(webHTML, `<script src="/joycon-ui.js"></script>`) {
		t.Fatal("embedded web app does not load the Joy-Con UI script")
	}
	for _, required := range []string{
		"Joy-Conを接続・再検索",
		"Joy-Con入力を記録",
		"選択中の割り当てへJoy-Con入力を設定",
		"Joy-Con接続・スティック設定を保存",
		"キャリブレーションを開始",
		"選択中の割り当てを保存",
		"joyRuleMode",
	} {
		if !strings.Contains(joyConUIJS, required) {
			t.Errorf("Joy-Con UI is missing %q", required)
		}
	}
}
