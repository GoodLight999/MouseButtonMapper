//go:build windows

package main

import (
	"fmt"
	"strings"
	"unsafe"
)

const (
	joyConRuleModeHold     = "Hold"
	joyConRuleModeHoldDown = "JoyConHoldDown"
	joyConRuleModeHoldUp   = "JoyConHoldUp"

	mouseEventMove       = 0x0001
	mouseEventLeftDown   = 0x0002
	mouseEventLeftUp     = 0x0004
	mouseEventRightDown  = 0x0008
	mouseEventRightUp    = 0x0010
	mouseEventMiddleDown = 0x0020
	mouseEventMiddleUp   = 0x0040
	mouseEventWheel      = 0x0800
	mouseEventXDown      = 0x0080
	mouseEventXUp        = 0x0100
	wheelDelta           = 120
)

type mouseInput struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type joyConOutputReference struct {
	Count int
	Owned bool
}

func normalizeJoyConRuleMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), joyConRuleModeHold) {
		return joyConRuleModeHold
	}
	return "Tap"
}

func isJoyConHoldRule(rule Rule) bool {
	return strings.EqualFold(rule.Mode, joyConRuleModeHold)
}

func isHoldableInputItem(item Item) bool {
	if isControllerInputKind(item.Kind) {
		return true
	}
	if strings.EqualFold(item.Kind, "Mouse") {
		code := normMouse(item.Code)
		// Allow side buttons (X1/X2); disallow primary Left/Right/Middle clicks and stateless wheel.
		return code == "X1" || code == "X2"
	}
	if strings.EqualFold(item.Kind, "Key") {
		_, ok := parseVK(item.Code)
		return ok
	}
	return false
}

func isHoldableMouseOutputCode(code string) bool {
	switch normMouse(code) {
	case "Left", "Right", "Middle", "X1", "X2":
		return true
	default:
		return false
	}
}

func validateJoyConHoldRule(rule Rule) error {
	if !isJoyConHoldRule(rule) {
		return nil
	}
	if rule.LongPressEnabled {
		return fmt.Errorf("押し続け連動(Hold)モードと長押し分岐は同じ割り当てでは併用できません")
	}
	if len(rule.Input) != 1 || !isHoldableInputItem(rule.Input[0]) {
		return fmt.Errorf("押し続け連動(Hold)モードはサイドボタン・拡張ボタン・キー・コントローラーの単独入力に設定できます（左/右/中クリック単独やホイールは対象外）")
	}
	if len(rule.Output) == 0 {
		return fmt.Errorf("押し続け連動(Hold)モードには押し続けるキーまたはマウスボタンが必要です")
	}
	for _, item := range rule.Output {
		switch {
		case strings.EqualFold(item.Kind, "Key"):
			if _, ok := parseVK(item.Code); !ok {
				return fmt.Errorf("Holdモードに不明なキーがあります: %s", item.Code)
			}
		case strings.EqualFold(item.Kind, "Mouse"):
			if !isHoldableMouseOutputCode(item.Code) {
				return fmt.Errorf("Holdモードの出力にホイールは指定できません（キーまたはマウスボタンを指定してください）: %s", item.Code)
			}
		default:
			return fmt.Errorf("Holdモードの実行内容にはキーボードキーまたはマウスボタンを指定してください")
		}
	}
	return nil
}

func joyConHoldPhaseRule(rule Rule, down bool) Rule {
	phase := cloneRule(rule)
	phase.LongPressEnabled = false
	phase.LongPressOutput = nil
	if down {
		phase.Mode = joyConRuleModeHoldDown
	} else {
		phase.Mode = joyConRuleModeHoldUp
	}
	return phase
}

func (a *App) sendRuleOutput(rule Rule) {
	switch rule.Mode {
	case joyConRuleModeHoldDown:
		a.sendJoyConHeldKeys(rule.Output, true)
	case joyConRuleModeHoldUp:
		a.sendJoyConHeldKeys(rule.Output, false)
	default:
		a.sendTapOutput(rule.Output)
	}
}

func (a *App) sendTapOutput(items []Item) {
	keys := make([]uint32, 0, len(items))
	mouseActions := make([]INPUT, 0, len(items)*2)
	unsupported := make([]string, 0)
	seenKeys := make(map[uint32]bool)

	for _, item := range items {
		if strings.EqualFold(item.Kind, "Key") {
			vk, ok := parseVK(item.Code)
			if !ok {
				unsupported = append(unsupported, item.Kind+":"+item.Code)
				continue
			}
			if isModifier(vk) {
				vk = normalizeModifier(vk)
			}
			if !seenKeys[vk] {
				keys = append(keys, vk)
				seenKeys[vk] = true
			}
			continue
		}
		if strings.EqualFold(item.Kind, "Mouse") {
			mouseInputs, ok := mouseTapInputs(item.Code)
			if !ok {
				unsupported = append(unsupported, item.Kind+":"+item.Code)
				continue
			}
			mouseActions = append(mouseActions, mouseInputs...)
			continue
		}
		unsupported = append(unsupported, item.Kind+":"+item.Code)
	}
	if len(unsupported) > 0 {
		a.logf("unsupported output ignored: %v", unsupported)
	}
	if len(keys) == 0 && len(mouseActions) == 0 {
		return
	}

	a.sendMu.Lock()
	defer a.sendMu.Unlock()
	if a.shuttingDown.Load() {
		return
	}

	// All injected keys are pressed before any mouse action and released in
	// reverse order. This preserves Ctrl+R and also supports A+B as a true
	// simultaneous chord instead of two sequential taps.
	inputs := make([]INPUT, 0, len(keys)*2+len(mouseActions))
	pressedByUs := make([]uint32, 0, len(keys))
	for _, vk := range keys {
		if !joyConShouldInjectTapKey(a.physicalKeyDown(vk)) {
			// Borrow any physically-held key; never synthesize its UP.
			continue
		}
		inputs = append(inputs, makeKeyInput(vk, false))
		pressedByUs = append(pressedByUs, vk)
	}
	inputs = append(inputs, mouseActions...)
	for i := len(pressedByUs) - 1; i >= 0; i-- {
		inputs = append(inputs, makeKeyInput(pressedByUs[i], true))
	}
	injectedCtrl := false
	for _, vk := range pressedByUs {
		if genericVK(vk) == VK_CONTROL {
			injectedCtrl = true
			break
		}
	}
	if injectedCtrl {
		a.injectingCtrl.Store(true)
		defer a.injectingCtrl.Store(false)
	}
	if ok := a.callSendInput(inputs); !ok && len(pressedByUs) > 0 {
		cleanup := make([]INPUT, 0, len(pressedByUs))
		for i := len(pressedByUs) - 1; i >= 0; i-- {
			cleanup = append(cleanup, makeKeyInput(pressedByUs[i], true))
		}
		a.callSendInput(cleanup)
	}
}

func (a *App) sendJoyConHeldKeys(items []Item, down bool) {
	keys := make([]uint32, 0, len(items))
	mouseButtons := make([]string, 0, len(items))
	seenKeys := make(map[uint32]bool)
	seenMouse := make(map[string]bool)
	for _, item := range items {
		if strings.EqualFold(item.Kind, "Key") {
			if vk, ok := parseVK(item.Code); ok {
				if isModifier(vk) {
					vk = normalizeModifier(vk)
				}
				if !seenKeys[vk] {
					seenKeys[vk] = true
					keys = append(keys, vk)
				}
			}
		} else if strings.EqualFold(item.Kind, "Mouse") {
			code := normMouse(item.Code)
			if isHoldableMouseOutputCode(code) && !seenMouse[code] {
				seenMouse[code] = true
				mouseButtons = append(mouseButtons, code)
			}
		}
	}
	if len(keys) == 0 && len(mouseButtons) == 0 {
		return
	}

	a.sendMu.Lock()
	defer a.sendMu.Unlock()
	if a.joyConOutputRefs == nil {
		a.joyConOutputRefs = make(map[uint32]joyConOutputReference)
	}
	if a.heldMouseRefs == nil {
		a.heldMouseRefs = make(map[string]int)
	}
	inputs := make([]INPUT, 0, len(keys)+len(mouseButtons))
	if down {
		for _, vk := range keys {
			ref := a.joyConOutputRefs[vk]
			if ref.Count == 0 {
				ref.Owned = !a.physicalKeyDown(vk)
				if ref.Owned && !a.shuttingDown.Load() {
					inputs = append(inputs, makeKeyInput(vk, false))
				}
			}
			ref.Count++
			a.joyConOutputRefs[vk] = ref
		}
		for _, btn := range mouseButtons {
			cnt := a.heldMouseRefs[btn]
			if cnt == 0 && !a.shuttingDown.Load() {
				if inp, ok := mouseHoldInput(btn, true); ok {
					inputs = append(inputs, inp)
				}
			}
			a.heldMouseRefs[btn] = cnt + 1
		}
	} else {
		// Release in reverse order (mouse buttons, then non-modifier keys, then modifiers)
		// so chords like Win+H release H before Win and avoid stray Start Menu opens.
		for i := len(mouseButtons) - 1; i >= 0; i-- {
			btn := mouseButtons[i]
			cnt := a.heldMouseRefs[btn]
			if cnt == 0 {
				continue
			}
			cnt--
			if cnt == 0 {
				delete(a.heldMouseRefs, btn)
				if inp, ok := mouseHoldInput(btn, false); ok {
					inputs = append(inputs, inp)
				}
			} else {
				a.heldMouseRefs[btn] = cnt
			}
		}
		for i := len(keys) - 1; i >= 0; i-- {
			vk := keys[i]
			ref := a.joyConOutputRefs[vk]
			if ref.Count == 0 {
				continue
			}
			ref.Count--
			if ref.Count == 0 {
				delete(a.joyConOutputRefs, vk)
				if joyConShouldReleaseOwnedKey(ref, a.physicalKeyDown(vk)) {
					inputs = append(inputs, makeKeyInput(vk, true))
				}
			} else {
				a.joyConOutputRefs[vk] = ref
			}
		}
	}
	a.callSendInput(inputs)
}

func (a *App) appendJoyConHeldReleaseInputs(inputs []INPUT) []INPUT {
	for vk, ref := range a.joyConOutputRefs {
		if ref.Count > 0 && joyConShouldReleaseOwnedKey(ref, a.physicalKeyDown(vk)) {
			inputs = append(inputs, makeKeyInput(vk, true))
		}
	}
	clear(a.joyConOutputRefs)
	for btn, cnt := range a.heldMouseRefs {
		if cnt > 0 {
			if inp, ok := mouseHoldInput(btn, false); ok {
				inputs = append(inputs, inp)
			}
		}
	}
	clear(a.heldMouseRefs)
	return inputs
}

func joyConShouldInjectTapKey(physicalDown bool) bool {
	return !physicalDown
}

func joyConShouldReleaseOwnedKey(ref joyConOutputReference, physicalDown bool) bool {
	return ref.Owned && !physicalDown
}

func (a *App) physicalKeyDown(vk uint32) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.keyDown[vk] || a.keyDown[genericVK(vk)]
}

func wheelData(delta int32) uint32 {
	return uint32(delta)
}

func mouseHoldInput(code string, down bool) (INPUT, bool) {
	switch normMouse(code) {
	case "Left":
		if down {
			return makeMouseInput(0, mouseEventLeftDown), true
		}
		return makeMouseInput(0, mouseEventLeftUp), true
	case "Right":
		if down {
			return makeMouseInput(0, mouseEventRightDown), true
		}
		return makeMouseInput(0, mouseEventRightUp), true
	case "Middle":
		if down {
			return makeMouseInput(0, mouseEventMiddleDown), true
		}
		return makeMouseInput(0, mouseEventMiddleUp), true
	case "X1":
		if down {
			return makeMouseInput(1, mouseEventXDown), true
		}
		return makeMouseInput(1, mouseEventXUp), true
	case "X2":
		if down {
			return makeMouseInput(2, mouseEventXDown), true
		}
		return makeMouseInput(2, mouseEventXUp), true
	default:
		return INPUT{}, false
	}
}

func mouseTapInputs(code string) ([]INPUT, bool) {
	switch normMouse(code) {
	case "Left":
		return []INPUT{makeMouseInput(0, mouseEventLeftDown), makeMouseInput(0, mouseEventLeftUp)}, true
	case "Right":
		return []INPUT{makeMouseInput(0, mouseEventRightDown), makeMouseInput(0, mouseEventRightUp)}, true
	case "Middle":
		return []INPUT{makeMouseInput(0, mouseEventMiddleDown), makeMouseInput(0, mouseEventMiddleUp)}, true
	case "X1":
		return []INPUT{makeMouseInput(1, mouseEventXDown), makeMouseInput(1, mouseEventXUp)}, true
	case "X2":
		return []INPUT{makeMouseInput(2, mouseEventXDown), makeMouseInput(2, mouseEventXUp)}, true
	case "WheelUp":
		return []INPUT{makeMouseInput(wheelData(wheelDelta), mouseEventWheel)}, true
	case "WheelDown":
		return []INPUT{makeMouseInput(wheelData(-wheelDelta), mouseEventWheel)}, true
	default:
		return nil, false
	}
}

func makeMouseInput(data uint32, flags uint32) INPUT {
	var input INPUT
	input.Type = INPUT_MOUSE
	mouse := mouseInput{
		MouseData:   data,
		DwFlags:     flags,
		DwExtraInfo: extraInfoMarker,
	}
	*(*mouseInput)(unsafe.Pointer(&input.Data[0])) = mouse
	return input
}
