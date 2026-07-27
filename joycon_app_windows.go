//go:build windows

package main

import (
	"context"
	"fmt"
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
			a.collectJoyConCalibrationSampleLocked(status)
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
	a.joyConWorker = nil
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
	a.handleControllerInputEvent("JoyCon", event)
}

func (a *App) clearJoyConInputStateLocked(reason string) {
	a.clearControllerInputStateLocked(reason)
}
