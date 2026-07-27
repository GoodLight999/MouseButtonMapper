//go:build windows

package main

import (
	"sort"
	"strings"
)

func isControllerInputKind(kind string) bool {
	return strings.EqualFold(kind, "JoyCon") || strings.EqualFold(kind, "XInput")
}

func normalizeControllerInput(kind, code string) (Item, bool) {
	switch {
	case strings.EqualFold(kind, "JoyCon"):
		code = normalizeJoyConCode(code)
		return Item{Kind: "JoyCon", Code: code}, isKnownJoyConCode(code)
	case strings.EqualFold(kind, "XInput"):
		code = normalizeXInputCode(code)
		return Item{Kind: "XInput", Code: code}, isKnownXInputCode(code)
	default:
		return Item{}, false
	}
}

func controllerInputKey(item Item) string {
	normalized, ok := normalizeControllerInput(item.Kind, item.Code)
	if !ok {
		return ""
	}
	return normalized.Kind + ":" + normalized.Code
}

func controllerItemFromKey(key string) (Item, bool) {
	kind, code, ok := strings.Cut(key, ":")
	if !ok {
		return Item{}, false
	}
	return normalizeControllerInput(kind, code)
}

func (a *App) handleControllerInputEvent(kind string, event InputEvent) {
	trigger, ok := normalizeControllerInput(kind, event.Token.Code)
	if !ok {
		a.logf("ignored unknown %s input: %q", kind, event.Token.Code)
		return
	}
	key := controllerInputKey(trigger)

	a.mu.Lock()
	wasDown := a.controllerDown[key]
	if event.Down {
		a.controllerDown[key] = true
		a.lastControllerInput = trigger
		a.lastControllerSource = event.SourceID
	} else {
		delete(a.controllerDown, key)
	}

	if a.recordingMode != "" {
		if a.recordingMode != "input" {
			a.postActivityRefreshLocked()
			a.mu.Unlock()
			return
		}
		finish := false
		if event.Down && !wasDown {
			a.recordDownLocked(trigger, "押下")
		} else if !event.Down && wasDown {
			finish = a.recordUpLocked(trigger, "離した")
		}
		a.postActivityRefreshLocked()
		a.mu.Unlock()
		if finish {
			go a.finishRecordingAuto()
		}
		return
	}

	if !event.Down {
		holdRule, hadHold := a.controllerHoldRules[key]
		delete(a.controllerHoldRules, key)
		pending := a.controllerPending[key]
		consumed := a.controllerConsumed[key]
		delete(a.controllerPending, key)
		delete(a.controllerConsumed, key)

		if event.Synthetic {
			a.abortLongPressForTriggerLocked(trigger, kind+" disconnected")
			delete(a.longPress, longPressKey(trigger))
			a.mu.Unlock()
			if hadHold {
				a.enqueueRuleGuaranteed(joyConHoldPhaseRule(holdRule, false))
			}
			return
		}

		completion := a.finishLongPressLocked(trigger)
		singleRule, single := a.singleControllerRuleLocked(trigger)
		active := a.enabled && !a.emergency
		a.mu.Unlock()

		if hadHold {
			a.enqueueRuleGuaranteed(joyConHoldPhaseRule(holdRule, false))
		}
		if completion.HasRule {
			a.enqueueRuleGuaranteed(completion.Rule)
		}
		if completion.Handled || hadHold {
			return
		}
		if pending && !consumed && single && active && !singleRule.LongPressEnabled && !isJoyConHoldRule(singleRule) {
			a.enqueueRuleGuaranteed(singleRule)
		}
		return
	}

	if wasDown {
		a.mu.Unlock()
		return
	}
	a.noteLastInputLocked(trigger, "押下")
	if !a.enabled || a.emergency {
		a.mu.Unlock()
		return
	}

	rule, matched := a.findBestTriggerLocked(trigger)
	if matched && len(rule.Input) > 1 {
		a.markPrefixesConsumedLocked(rule)
	}
	if matched && isJoyConHoldRule(rule) {
		a.controllerHoldRules[key] = cloneRule(rule)
		a.mu.Unlock()
		a.enqueueRuleGuaranteed(joyConHoldPhaseRule(rule, true))
		return
	}
	if matched && rule.LongPressEnabled {
		a.startLongPressLocked(rule, trigger)
		a.mu.Unlock()
		return
	}
	if matched && len(rule.Input) > 1 {
		a.mu.Unlock()
		a.enqueueRuleGuaranteed(rule)
		return
	}
	if matched {
		a.controllerPending[key] = true
	}
	a.mu.Unlock()
}

func (a *App) singleControllerRuleLocked(trigger Item) (Rule, bool) {
	for _, rule := range a.rules {
		if len(rule.Input) == 1 && sameInput(rule.Input[0], trigger) {
			return rule, true
		}
	}
	return Rule{}, false
}

func (a *App) clearControllerInputStateLocked(reason string) {
	keys := make([]string, 0, len(a.controllerDown))
	for key := range a.controllerDown {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		trigger, ok := controllerItemFromKey(key)
		if !ok {
			continue
		}
		a.abortLongPressForTriggerLocked(trigger, reason)
		delete(a.longPress, longPressKey(trigger))
	}
	clear(a.controllerDown)
	clear(a.controllerPending)
	clear(a.controllerConsumed)
	clear(a.controllerHoldRules)
}
