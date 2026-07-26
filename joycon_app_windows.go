//go:build windows

package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

func normalizeJoyConCode(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "up", "dpadup", "方向上":
		return string(JoyConButtonUp)
	case "down", "dpaddown", "方向下":
		return string(JoyConButtonDown)
	case "left", "dpadleft", "方向左":
		return string(JoyConButtonLeft)
	case "right", "dpadright", "方向右":
		return string(JoyConButtonRight)
	case "l":
		return string(JoyConButtonL)
	case "zl":
		return string(JoyConButtonZL)
	case "sl":
		return string(JoyConButtonSL)
	case "sr":
		return string(JoyConButtonSR)
	case "minus", "-", "－":
		return string(JoyConButtonMinus)
	case "capture", "キャプチャー":
		return string(JoyConButtonCapture)
	case "stickpress", "stick", "スティック押込み":
		return string(JoyConButtonStick)
	case "stickup", "スティック上":
		return string(JoyConStickUp)
	case "stickdown", "スティック下":
		return string(JoyConStickDown)
	case "stickleft", "スティック左":
		return string(JoyConStickLeft)
	case "stickright", "スティック右":
		return string(JoyConStickRight)
	default:
		return strings.TrimSpace(code)
	}
}

func isKnownJoyConCode(code string) bool {
	code = normalizeJoyConCode(code)
	for _, button := range joyConLeftPhysicalButtons {
		if code == string(button) {
			return true
		}
	}
	for _, direction := range joyConStickDirections {
		if code == string(direction) {
			return true
		}
	}
	return false
}

func joyConCodeText(code string) string {
	switch normalizeJoyConCode(code) {
	case string(JoyConButtonUp):
		return "方向上"
	case string(JoyConButtonDown):
		return "方向下"
	case string(JoyConButtonLeft):
		return "方向左"
	case string(JoyConButtonRight):
		return "方向右"
	case string(JoyConButtonMinus):
		return "－"
	case string(JoyConButtonCapture):
		return "キャプチャー"
	case string(JoyConButtonStick):
		return "スティック押込み"
	case string(JoyConStickUp):
		return "スティック上"
	case string(JoyConStickDown):
		return "スティック下"
	case string(JoyConStickLeft):
		return "スティック左"
	case string(JoyConStickRight):
		return "スティック右"
	default:
		return normalizeJoyConCode(code)
	}
}

func (a *App) startJoyConSubsystem() {
	worker, err := NewJoyConWorker(JoyConWorkerOptions{
		Backend: WindowsJoyConBackend{},
		Config:  a.effectiveJoyConConfig,
		Emit:    a.handleJoyConInputEvent,
		StatusChanged: func(status JoyConConnectionStatus) {
			a.mu.Lock()
			a.joyConStatus = status
			a.postActivityRefreshLocked()
			a.mu.Unlock()
		},
	})
	if err != nil {
		a.logf("Joy-Con worker init failed: %v", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	a.mu.Lock()
	a.joyConWorker = worker
	a.joyConCancel = cancel
	a.joyConDone = done
	a.mu.Unlock()

	a.workerWG.Add(1)
	go func() {
		defer a.workerWG.Done()
		defer close(done)
		if err := worker.Run(ctx); err != nil {
			a.logf("Joy-Con worker stopped with error: %v", err)
		}
	}()
}

func (a *App) stopJoyConSubsystem() {
	a.mu.Lock()
	cancel := a.joyConCancel
	done := a.joyConDone
	a.joyConCancel = nil
	a.joyConDone = nil
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			a.logf("Joy-Con worker shutdown timeout")
		}
	}
}

func (a *App) effectiveJoyConConfig() JoyConProfileConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.activeProfileIndex < 0 || a.activeProfileIndex >= len(a.config.Profiles) {
		return defaultJoyConProfileConfig()
	}
	return normalizeJoyConProfileConfig(a.config.Profiles[a.activeProfileIndex].JoyCon)
}

func (a *App) requestJoyConRescanLocked() {
	if a.joyConWorker != nil {
		a.joyConWorker.RequestRescan()
	}
}

func (a *App) requestJoyConRescan() {
	a.mu.RLock()
	worker := a.joyConWorker
	a.mu.RUnlock()
	if worker != nil {
		worker.RequestRescan()
	}
}

func (a *App) joyConStatusSnapshot() JoyConConnectionStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.joyConStatus
}

func (a *App) joyConStatusTextLocked() string {
	status := a.joyConStatus
	if status.Connected {
		name := strings.TrimSpace(status.Device.Product)
		if name == "" {
			name = "Joy-Con (L)"
		}
		battery := "バッテリー不明"
		if status.BatteryPercent >= 0 {
			battery = fmt.Sprintf("バッテリー%d%%", status.BatteryPercent)
			if status.Charging {
				battery += "・充電中"
			}
		}
		return fmt.Sprintf("接続中: %s / %s", name, battery)
	}
	if status.LastError != "" {
		return "未接続: " + status.LastError
	}
	return "未接続"
}

func (a *App) handleJoyConInputEvent(event InputEvent) {
	code := normalizeJoyConCode(event.Token.Code)
	if !isKnownJoyConCode(code) {
		a.logf("ignored unknown Joy-Con input: %q", event.Token.Code)
		return
	}
	trigger := Item{Kind: "JoyCon", Code: code}

	a.mu.Lock()
	wasDown := a.joyConDown[code]
	if event.Down {
		a.joyConDown[code] = true
	} else {
		delete(a.joyConDown, code)
	}

	if a.recordingMode != "" {
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
		holdRule, hadHold := a.joyConHoldRules[code]
		delete(a.joyConHoldRules, code)
		pending := a.joyConPending[code]
		consumed := a.joyConConsumed[code]
		delete(a.joyConPending, code)
		delete(a.joyConConsumed, code)

		if event.Synthetic {
			a.abortLongPressForTriggerLocked(trigger, "Joy-Con disconnected")
			delete(a.longPress, longPressKey(trigger))
			a.mu.Unlock()
			if hadHold {
				a.enqueueRuleGuaranteed(joyConHoldPhaseRule(holdRule, false))
			}
			return
		}

		completion := a.finishLongPressLocked(trigger)
		singleRule, single := a.singleJoyConRuleLocked(code)
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
		a.joyConHoldRules[code] = cloneRule(rule)
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
		a.joyConPending[code] = true
	}
	a.mu.Unlock()
}

func (a *App) singleJoyConRuleLocked(code string) (Rule, bool) {
	code = normalizeJoyConCode(code)
	for _, rule := range a.rules {
		if len(rule.Input) == 1 && strings.EqualFold(rule.Input[0].Kind, "JoyCon") && normalizeJoyConCode(rule.Input[0].Code) == code {
			return rule, true
		}
	}
	return Rule{}, false
}

func (a *App) clearJoyConInputStateLocked(reason string) {
	codes := make([]string, 0, len(a.joyConDown))
	for code := range a.joyConDown {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		trigger := Item{Kind: "JoyCon", Code: code}
		a.abortLongPressForTriggerLocked(trigger, reason)
		delete(a.longPress, longPressKey(trigger))
	}
	clear(a.joyConDown)
	clear(a.joyConPending)
	clear(a.joyConConsumed)
	clear(a.joyConHoldRules)
}
