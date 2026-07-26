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

	joyConMouseEventMove       = 0x0001
	joyConMouseEventLeftDown   = 0x0002
	joyConMouseEventLeftUp     = 0x0004
	joyConMouseEventRightDown  = 0x0008
	joyConMouseEventRightUp    = 0x0010
	joyConMouseEventMiddleDown = 0x0020
	joyConMouseEventMiddleUp   = 0x0040
	joyConMouseEventWheel      = 0x0800
	joyConMouseEventXDown      = 0x0080
	joyConMouseEventXUp        = 0x0100
	joyConWheelDelta           = 120
)

type joyConMouseInput struct {
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

func validateJoyConHoldRule(rule Rule) error {
	if !isJoyConHoldRule(rule) {
		return nil
	}
	if rule.LongPressEnabled {
		return fmt.Errorf("Holdモードと長押しは同じ割り当てでは併用できません")
	}
	if len(rule.Input) != 1 || !strings.EqualFold(rule.Input[0].Kind, "JoyCon") {
		return fmt.Errorf("HoldモードはJoy-Con単独入力にだけ設定できます")
	}
	if len(rule.Output) == 0 {
		return fmt.Errorf("Holdモードには保持するキーが必要です")
	}
	for _, item := range rule.Output {
		if !strings.EqualFold(item.Kind, "Key") {
			return fmt.Errorf("Holdモードの実行内容にはキーボードキーだけを指定してください")
		}
		if _, ok := parseVK(item.Code); !ok {
			return fmt.Errorf("Holdモードに不明なキーがあります: %s", item.Code)
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

func (a *App) sendJoyConRuleOutput(rule Rule) {
	switch rule.Mode {
	case joyConRuleModeHoldDown:
		a.sendJoyConHeldKeys(rule.Output, true)
	case joyConRuleModeHoldUp:
		a.sendJoyConHeldKeys(rule.Output, false)
	default:
		a.sendJoyConTapOutput(rule.Output)
	}
}

func (a *App) sendJoyConTapOutput(items []Item) {
	modifiers := make([]uint32, 0, len(items))
	actions := make([]INPUT, 0, len(items)*2)
	unsupported := make([]string, 0)
	seenModifiers := make(map[uint32]bool)

	for _, item := range items {
		if strings.EqualFold(item.Kind, "Key") {
			vk, ok := parseVK(item.Code)
			if !ok {
				unsupported = append(unsupported, item.Kind+":"+item.Code)
				continue
			}
			if isModifier(vk) {
				vk = normalizeModifier(vk)
				if !seenModifiers[vk] {
					modifiers = append(modifiers, vk)
					seenModifiers[vk] = true
				}
				continue
			}
			actions = append(actions, makeKeyInput(vk, false), makeKeyInput(vk, true))
			continue
		}
		if strings.EqualFold(item.Kind, "Mouse") {
			mouseInputs, ok := joyConMouseTapInputs(item.Code)
			if !ok {
				unsupported = append(unsupported, item.Kind+":"+item.Code)
				continue
			}
			actions = append(actions, mouseInputs...)
			continue
		}
		unsupported = append(unsupported, item.Kind+":"+item.Code)
	}
	if len(unsupported) > 0 {
		a.logf("unsupported output ignored: %v", unsupported)
	}
	if len(actions) == 0 && len(modifiers) == 0 {
		return
	}

	a.sendMu.Lock()
	defer a.sendMu.Unlock()
	if a.shuttingDown.Load() {
		return
	}

	inputs := make([]INPUT, 0, len(actions)+len(modifiers)*2)
	pressedByUs := make([]uint32, 0, len(modifiers))
	for _, vk := range modifiers {
		if a.physicalKeyDown(vk) {
			continue
		}
		inputs = append(inputs, makeKeyInput(vk, false))
		pressedByUs = append(pressedByUs, vk)
	}
	inputs = append(inputs, actions...)
	for i := len(pressedByUs) - 1; i >= 0; i-- {
		inputs = append(inputs, makeKeyInput(pressedByUs[i], true))
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
	for _, item := range items {
		if !strings.EqualFold(item.Kind, "Key") {
			continue
		}
		if vk, ok := parseVK(item.Code); ok {
			if isModifier(vk) {
				vk = normalizeModifier(vk)
			}
			keys = append(keys, vk)
		}
	}
	if len(keys) == 0 {
		return
	}

	a.sendMu.Lock()
	defer a.sendMu.Unlock()
	if a.joyConOutputRefs == nil {
		a.joyConOutputRefs = make(map[uint32]joyConOutputReference)
	}
	inputs := make([]INPUT, 0, len(keys))
	for _, vk := range keys {
		ref := a.joyConOutputRefs[vk]
		if down {
			if ref.Count == 0 {
				ref.Owned = !a.physicalKeyDown(vk)
				if ref.Owned && !a.shuttingDown.Load() {
					inputs = append(inputs, makeKeyInput(vk, false))
				}
			}
			ref.Count++
			a.joyConOutputRefs[vk] = ref
			continue
		}
		if ref.Count == 0 {
			continue
		}
		ref.Count--
		if ref.Count == 0 {
			delete(a.joyConOutputRefs, vk)
			if ref.Owned {
				inputs = append(inputs, makeKeyInput(vk, true))
			}
		} else {
			a.joyConOutputRefs[vk] = ref
		}
	}
	a.callSendInput(inputs)
}

func (a *App) appendJoyConHeldReleaseInputs(inputs []INPUT) []INPUT {
	for vk, ref := range a.joyConOutputRefs {
		if ref.Count > 0 && ref.Owned {
			inputs = append(inputs, makeKeyInput(vk, true))
		}
	}
	clear(a.joyConOutputRefs)
	return inputs
}

func (a *App) physicalKeyDown(vk uint32) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.keyDown[vk] || a.keyDown[genericVK(vk)]
}

func joyConMouseTapInputs(code string) ([]INPUT, bool) {
	switch normMouse(code) {
	case "Left":
		return []INPUT{makeJoyConMouseInput(0, joyConMouseEventLeftDown), makeJoyConMouseInput(0, joyConMouseEventLeftUp)}, true
	case "Right":
		return []INPUT{makeJoyConMouseInput(0, joyConMouseEventRightDown), makeJoyConMouseInput(0, joyConMouseEventRightUp)}, true
	case "Middle":
		return []INPUT{makeJoyConMouseInput(0, joyConMouseEventMiddleDown), makeJoyConMouseInput(0, joyConMouseEventMiddleUp)}, true
	case "X1":
		return []INPUT{makeJoyConMouseInput(1, joyConMouseEventXDown), makeJoyConMouseInput(1, joyConMouseEventXUp)}, true
	case "X2":
		return []INPUT{makeJoyConMouseInput(2, joyConMouseEventXDown), makeJoyConMouseInput(2, joyConMouseEventXUp)}, true
	case "WheelUp":
		return []INPUT{makeJoyConMouseInput(uint32(joyConWheelDelta), joyConMouseEventWheel)}, true
	case "WheelDown":
		return []INPUT{makeJoyConMouseInput(uint32(0)-uint32(joyConWheelDelta), joyConMouseEventWheel)}, true
	default:
		return nil, false
	}
}

func makeJoyConMouseInput(data uint32, flags uint32) INPUT {
	var input INPUT
	input.Type = INPUT_MOUSE
	mouse := joyConMouseInput{
		MouseData:   data,
		DwFlags:     flags,
		DwExtraInfo: extraInfoMarker,
	}
	*(*joyConMouseInput)(unsafe.Pointer(&input.Data[0])) = mouse
	return input
}
