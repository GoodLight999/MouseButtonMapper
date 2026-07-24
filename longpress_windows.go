//go:build windows

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type longPressState struct {
	Token     uint64
	Rule      Rule
	Trigger   Item
	StartedAt time.Time
	Fired     bool
	Aborted   bool
	Timer     *time.Timer
}

type longPressCompletion struct {
	Handled  bool
	Suppress bool
	HasRule  bool
	Rule     Rule
	Kind     string
}

func longPressKey(it Item) string {
	if strings.EqualFold(it.Kind, "Mouse") {
		return "mouse:" + strings.ToLower(normMouse(it.Code))
	}
	if strings.EqualFold(it.Kind, "Key") {
		if vk, ok := parseVK(it.Code); ok {
			return "key:" + strconv.Itoa(int(genericVK(vk)))
		}
	}
	return ""
}

func isHoldableLongPressTrigger(it Item) bool {
	if strings.EqualFold(it.Kind, "Key") {
		vk, ok := parseVK(it.Code)
		return ok && !isModifier(vk)
	}
	if !strings.EqualFold(it.Kind, "Mouse") {
		return false
	}
	switch normMouse(it.Code) {
	case "X1", "X2":
		return true
	default:
		return false
	}
}

func validateLongPressRule(r Rule) error {
	if !r.LongPressEnabled {
		return nil
	}
	if len(r.Input) == 0 {
		return fmt.Errorf("長押し判定には入力が必要です")
	}
	trigger := r.Input[len(r.Input)-1]
	if !isHoldableLongPressTrigger(trigger) {
		return fmt.Errorf("長押し判定の最後の入力には、サイド1・サイド2・修飾キー以外のキーボードキーを指定してください")
	}
	action := normalizeLongPressAction(r.LongPressAction)
	if action == longPressActionExecute && len(r.LongPressOutput) == 0 {
		return fmt.Errorf("長押し時に別の操作を実行する場合は、長押し時の実行内容が必要です")
	}
	if len(r.Output) == 0 && action == longPressActionCancel {
		return fmt.Errorf("短押し時の実行内容が空のため、長押しキャンセルを設定しても何も起こりません")
	}
	return nil
}

func ruleHasRunnableOutput(r Rule) bool {
	if len(r.Output) > 0 {
		return true
	}
	return r.LongPressEnabled && normalizeLongPressAction(r.LongPressAction) == longPressActionExecute && len(r.LongPressOutput) > 0
}

func longOutputRule(r Rule) Rule {
	out := cloneRule(r)
	out.Output = append([]Item(nil), r.LongPressOutput...)
	out.LongPressEnabled = false
	out.LongPressOutput = nil
	return out
}

func (a *App) startLongPressLocked(r Rule, trigger Item) bool {
	if !r.LongPressEnabled || !isHoldableLongPressTrigger(trigger) {
		return false
	}
	key := longPressKey(trigger)
	if key == "" {
		return false
	}
	if existing := a.longPress[key]; existing != nil {
		if existing.Timer != nil {
			existing.Timer.Stop()
		}
		delete(a.longPress, key)
	}
	a.longPressSeq++
	state := &longPressState{
		Token:     a.longPressSeq,
		Rule:      cloneRule(r),
		Trigger:   normalizeRecordedItem(trigger),
		StartedAt: time.Now(),
	}
	threshold := time.Duration(normalizeLongPressMs(r.LongPressMs)) * time.Millisecond
	state.Timer = time.AfterFunc(threshold, func() {
		a.fireLongPress(key, state.Token)
	})
	a.longPress[key] = state
	return true
}

func (a *App) longPressSuppressedLocked(trigger Item) bool {
	state := a.longPress[longPressKey(trigger)]
	return state != nil && state.Rule.SuppressTrigger
}

func (a *App) abortLongPressForTriggerLocked(trigger Item, reason string) {
	state := a.longPress[longPressKey(trigger)]
	if state == nil {
		return
	}
	if state.Timer != nil {
		state.Timer.Stop()
	}
	state.Fired = true
	state.Aborted = true
	if reason != "" {
		a.logf("long press aborted: reason=%s input=%s", reason, itemsText(state.Rule.Input))
	}
}

func (a *App) abortAllLongPressLocked(reason string, clear bool) {
	for key, state := range a.longPress {
		if state.Timer != nil {
			state.Timer.Stop()
		}
		state.Fired = true
		state.Aborted = true
		if clear {
			delete(a.longPress, key)
		}
	}
	if reason != "" && len(a.longPress) > 0 {
		a.logf("long press states aborted: reason=%s count=%d", reason, len(a.longPress))
	}
}

func (a *App) fireLongPress(key string, token uint64) {
	var out Rule
	hasOutput := false
	action := ""
	inputText := ""

	a.mu.Lock()
	state := a.longPress[key]
	if state == nil || state.Token != token || state.Fired || state.Aborted {
		a.mu.Unlock()
		return
	}
	if !a.enabled || a.emergency || !a.isItemDownLocked(state.Trigger) {
		state.Fired = true
		state.Aborted = true
		a.mu.Unlock()
		return
	}
	state.Fired = true
	action = normalizeLongPressAction(state.Rule.LongPressAction)
	inputText = itemsText(state.Rule.Input)
	if action == longPressActionExecute && len(state.Rule.LongPressOutput) > 0 {
		out = longOutputRule(state.Rule)
		hasOutput = true
	}
	a.lastInputText = inputText + " 長押し"
	a.lastInputAt = time.Now()
	a.postActivityRefreshLocked()
	a.mu.Unlock()

	if hasOutput {
		a.enqueueRuleGuaranteed(out)
		a.logf("long press executed: input=%s output=%s", inputText, itemsText(out.Output))
	} else {
		a.logf("long press cancelled short action: input=%s", inputText)
	}
}

func (a *App) finishLongPressLocked(trigger Item) longPressCompletion {
	key := longPressKey(trigger)
	state := a.longPress[key]
	if state == nil {
		return longPressCompletion{}
	}
	if state.Timer != nil {
		state.Timer.Stop()
	}
	delete(a.longPress, key)

	result := longPressCompletion{Handled: true, Suppress: state.Rule.SuppressTrigger}
	if state.Aborted || !a.enabled || a.emergency {
		result.Kind = "aborted"
		return result
	}

	isLong := state.Fired || longPressReached(time.Since(state.StartedAt), state.Rule.LongPressMs)
	if isLong {
		result.Kind = "long"
		if !state.Fired && normalizeLongPressAction(state.Rule.LongPressAction) == longPressActionExecute && len(state.Rule.LongPressOutput) > 0 {
			result.Rule = longOutputRule(state.Rule)
			result.HasRule = true
		}
		return result
	}

	result.Kind = "short"
	if len(state.Rule.Output) > 0 {
		result.Rule = cloneRule(state.Rule)
		result.Rule.LongPressEnabled = false
		result.Rule.LongPressOutput = nil
		result.HasRule = true
	}
	return result
}

func (a *App) enqueueRuleGuaranteed(r Rule) {
	if len(r.Output) == 0 || a.shuttingDown.Load() {
		return
	}
	if !a.enqueueRule(r) {
		rule := cloneRule(r)
		go func() {
			if a.shuttingDown.Load() {
				return
			}
			a.sendRule(rule)
		}()
	}
}

func ruleLongPressSummary(r Rule) string {
	if !r.LongPressEnabled {
		return "使用しない"
	}
	ms := normalizeLongPressMs(r.LongPressMs)
	if normalizeLongPressAction(r.LongPressAction) == longPressActionCancel {
		return fmt.Sprintf("%dms以上: 短押しをキャンセル", ms)
	}
	return fmt.Sprintf("%dms以上: %s", ms, itemsText(r.LongPressOutput))
}
