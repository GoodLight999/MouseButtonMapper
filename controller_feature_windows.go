//go:build windows

package main

import (
	"fmt"
	"net/http"
)

// ControllerFeatureConfig is the global opt-in gate for every game-controller
// input path. When disabled, Raw HID enumeration and XInput polling are stopped,
// controller-specific UI is collapsed, and stored controller rules are excluded
// from the active rule set without being deleted from config.json.
type ControllerFeatureConfig struct {
	Enabled bool `json:"Enabled"`
	Visible bool `json:"Visible"`
}

func ruleUsesControllerInput(rule Rule) bool {
	for _, item := range rule.Input {
		if isControllerInputKind(item.Kind) {
			return true
		}
	}
	return false
}

func (a *App) controllerFeatureEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Controller.Enabled
}

func (a *App) syncControllerSubsystems() {
	if a.controllerFeatureEnabled() {
		a.startJoyConSubsystem()
		a.startXInputSubsystem()
		return
	}

	// The config gate is already false before reaching this path, so any late
	// worker callback is ignored by handleControllerInputEvent.
	a.mu.Lock()
	a.clearControllerInputStateLocked("controller feature disabled")
	a.joyConCalibrationActive = false
	a.joyConCalibration = nil
	a.joyConCalibrationMessage = "コントローラー機能が無効なため中止しました。"
	a.mu.Unlock()
	a.releaseJoyConHeldOutputs()

	a.stopXInputSubsystem()
	a.stopJoyConSubsystem()

	a.mu.Lock()
	a.joyConStatus = JoyConConnectionStatus{BatteryPercent: -1}
	a.xInputStatus = XInputConnectionStatus{}
	a.lastControllerInput = Item{}
	a.lastControllerSource = ""
	a.postActivityRefreshLocked()
	a.mu.Unlock()
}

func (a *App) setControllerFeatureEnabled(enabled bool) error {
	a.mu.Lock()
	previous := a.config.Controller
	a.config.Controller.Enabled = enabled
	if enabled {
		a.config.Controller.Visible = true
	}
	a.rebuildRulesWithoutJoyConRescanLocked()
	if err := a.saveConfigLocked(); err != nil {
		a.config.Controller = previous
		a.rebuildRulesWithoutJoyConRescanLocked()
		a.mu.Unlock()
		return err
	}
	if !enabled {
		a.clearControllerInputStateLocked("controller feature disabled")
	}
	a.postUIRefreshLocked()
	a.mu.Unlock()

	if !enabled {
		a.releaseJoyConHeldOutputs()
	}
	a.syncControllerSubsystems()
	a.logf("controller feature changed: enabled=%v", enabled)
	return nil
}

type controllerFeatureRequest struct {
	Op      string `json:"op"`
	Enabled bool   `json:"enabled"`
}

func (a *App) setControllerFeatureVisible(visible bool) error {
	a.mu.Lock()
	previous := a.config.Controller
	a.config.Controller.Visible = visible
	// Hiding is an explicit opt-out. Never leave a hidden controller worker
	// running where the user cannot see or stop it from the settings screen.
	if !visible {
		a.config.Controller.Enabled = false
		a.rebuildRulesWithoutJoyConRescanLocked()
		a.clearControllerInputStateLocked("controller UI hidden")
	}
	if err := a.saveConfigLocked(); err != nil {
		a.config.Controller = previous
		a.rebuildRulesWithoutJoyConRescanLocked()
		a.mu.Unlock()
		return err
	}
	a.postUIRefreshLocked()
	a.mu.Unlock()
	if !visible {
		a.releaseJoyConHeldOutputs()
		a.syncControllerSubsystems()
	}
	a.logf("controller UI visibility changed: visible=%v", visible)
	return nil
}

func (a *App) webAPIControllerFeature(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req controllerFeatureRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	var err error
	var message string
	switch req.Op {
	case "show-ui":
		err = a.setControllerFeatureVisible(true)
		message = "実験機能の設定を表示しました。ゲームコントローラー機能はまだ無効です。"
	case "hide-ui":
		err = a.setControllerFeatureVisible(false)
		message = "ゲームコントローラー機能を停止し、設定画面から隠しました。"
	case "", "set-enabled":
		err = a.setControllerFeatureEnabled(req.Enabled)
		message = "ゲームコントローラー機能を無効化しました。HID列挙とXInput監視は停止しています。"
		if req.Enabled {
			message = "ゲームコントローラー機能を有効化しました。Joy-Con／互換HID／XInputの設定を表示します。"
		}
	default:
		err = fmt.Errorf("未知のコントローラー設定操作です: %s", req.Op)
	}
	if err != nil {
		writeError(w, fmt.Errorf("コントローラー機能設定を保存できません: %w", err))
		return
	}
	writeJSON(w, map[string]any{
		"ok":      true,
		"message": message,
		"state":   a.buildWebState(),
		"joyCon":  a.buildJoyConWebState(),
	})
}
