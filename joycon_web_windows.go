//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
)

type joyConWebState struct {
	ProfileIndex       int                    `json:"profileIndex"`
	ProfileName        string                 `json:"profileName"`
	Enabled            bool                   `json:"enabled"`
	PreferredDevice    string                 `json:"preferredDevice"`
	ReconnectEnabled   bool                   `json:"reconnectEnabled"`
	ReconnectMs        int                    `json:"reconnectMs"`
	Stick              JoyConStickConfig      `json:"stick"`
	Status             JoyConConnectionStatus `json:"status"`
	StatusText         string                 `json:"statusText"`
	XInputStatus       XInputConnectionStatus `json:"xInputStatus"`
	XInputStatusText   string                 `json:"xInputStatusText"`
	LastInputText      string                 `json:"lastInputText"`
	LastControllerKind string                 `json:"lastControllerKind"`
	LastControllerCode string                 `json:"lastControllerCode"`
	LastControllerText string                 `json:"lastControllerText"`
	CalibrationActive  bool                   `json:"calibrationActive"`
	CalibrationText    string                 `json:"calibrationText"`
}

type joyConWebRequest struct {
	Op               string  `json:"op"`
	Enabled          bool    `json:"enabled"`
	PreferredDevice  string  `json:"preferredDevice"`
	ReconnectEnabled bool    `json:"reconnectEnabled"`
	ReconnectMs      int     `json:"reconnectMs"`
	DeadZone         float64 `json:"deadZone"`
	ReleaseZone      float64 `json:"releaseZone"`
	DirectionMode    string  `json:"directionMode"`
	InvertX          bool    `json:"invertX"`
	InvertY          bool    `json:"invertY"`
}

func (a *App) buildJoyConWebState() joyConWebState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	config := defaultJoyConProfileConfig()
	profileIndex := -1
	profileName := "（プロファイルなし）"
	if a.editorProfileIndex >= 0 && a.editorProfileIndex < len(a.config.Profiles) {
		profile := a.config.Profiles[a.editorProfileIndex]
		profileIndex = a.editorProfileIndex
		profileName = profile.Name
		config = normalizeJoyConProfileConfig(profile.JoyCon)
	}
	lastInputText := "―"
	if a.joyConStatus.LastInput != "" {
		lastInputText = joyConCodeText(a.joyConStatus.LastInput)
	}
	calibrationText := a.joyConCalibrationMessage
	if calibrationText == "" {
		if a.joyConCalibrationActive {
			calibrationText = "中心で静止した後、スティックを全方向へゆっくり倒してください。"
		} else {
			calibrationText = "未実行"
		}
	}
	lastControllerText := "―"
	if a.lastControllerInput.Kind != "" {
		lastControllerText = itemsText([]Item{a.lastControllerInput})
	}
	return joyConWebState{
		ProfileIndex:       profileIndex,
		ProfileName:        profileName,
		Enabled:            config.Enabled,
		PreferredDevice:    config.PreferredDevice,
		ReconnectEnabled:   config.Reconnect.Enabled,
		ReconnectMs:        config.Reconnect.IntervalMs,
		Stick:              config.Stick,
		Status:             a.joyConStatus,
		StatusText:         a.joyConStatusTextLocked(),
		XInputStatus:       a.xInputStatus,
		XInputStatusText:   a.xInputStatusTextLocked(),
		LastInputText:      lastInputText,
		LastControllerKind: a.lastControllerInput.Kind,
		LastControllerCode: a.lastControllerInput.Code,
		LastControllerText: lastControllerText,
		CalibrationActive:  a.joyConCalibrationActive,
		CalibrationText:    calibrationText,
	}
}

func (a *App) webAPIJoyCon(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, a.buildJoyConWebState())
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req joyConWebRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, fmt.Errorf("Joy-Con設定のJSONを解釈できません: %w", err))
		return
	}

	var message string
	var err error
	switch strings.TrimSpace(req.Op) {
	case "rescan":
		a.requestJoyConRescan()
		message = "Joy-Conを再検索しています。"
	case "save-stick":
		err = a.saveJoyConWebSettings(req)
		message = "Joy-Con接続・スティック設定を保存しました。"
	case "calibration-start":
		err = a.startJoyConCalibration()
		message = "キャリブレーションを開始しました。中心で静止した後、全方向へゆっくり倒してください。"
	case "calibration-finish":
		err = a.finishJoyConCalibration()
		message = "キャリブレーション結果を選択中のプロファイルへ保存しました。"
	case "calibration-cancel":
		a.cancelJoyConCalibration()
		message = "キャリブレーションを中止しました。"
	default:
		err = fmt.Errorf("未知のJoy-Con操作です: %s", req.Op)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "message": message, "joyCon": a.buildJoyConWebState()})
}

func (a *App) saveJoyConWebSettings(req joyConWebRequest) error {
	if math.IsNaN(req.DeadZone) || math.IsInf(req.DeadZone, 0) || req.DeadZone < 0.05 || req.DeadZone > 0.90 {
		return fmt.Errorf("デッドゾーンは0.05〜0.90で指定してください。")
	}
	if math.IsNaN(req.ReleaseZone) || math.IsInf(req.ReleaseZone, 0) || req.ReleaseZone < 0.01 || req.ReleaseZone >= req.DeadZone {
		return fmt.Errorf("解放判定は0.01以上かつデッドゾーン未満で指定してください。")
	}
	if req.DirectionMode != joyConDirectionMode4 && req.DirectionMode != joyConDirectionMode8 {
		return fmt.Errorf("方向判定は4方向または8方向を指定してください。")
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.editorProfileIndex < 0 || a.editorProfileIndex >= len(a.config.Profiles) {
		return fmt.Errorf("Joy-Con設定を保存するプロファイルがありません。")
	}
	current := normalizeJoyConProfileConfig(a.config.Profiles[a.editorProfileIndex].JoyCon)
	current.Enabled = req.Enabled
	current.PreferredDevice = strings.TrimSpace(req.PreferredDevice)
	current.Reconnect.Enabled = req.ReconnectEnabled
	current.Reconnect.IntervalMs = req.ReconnectMs
	current.Stick.DeadZone = req.DeadZone
	current.Stick.ReleaseZone = req.ReleaseZone
	current.Stick.DirectionMode = req.DirectionMode
	current.Stick.InvertX = req.InvertX
	current.Stick.InvertY = req.InvertY
	current = normalizeJoyConProfileConfig(current)
	a.config.Profiles[a.editorProfileIndex].JoyCon = current
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	if a.editorProfileIndex == a.activeProfileIndex {
		a.requestJoyConRescanLocked()
	}
	return nil
}

func (a *App) startJoyConCalibration() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.joyConStatus.Connected {
		return fmt.Errorf("Joy-Con（L）が接続されていません。先に「Joy-Conを接続・再検索」を実行してください。")
	}
	if a.editorProfileIndex < 0 || a.editorProfileIndex >= len(a.config.Profiles) {
		return fmt.Errorf("キャリブレーションを保存するプロファイルがありません。")
	}
	a.joyConCalibration = newJoyConCalibrationSession()
	a.joyConCalibration.profileID = a.config.Profiles[a.editorProfileIndex].Id
	a.joyConCalibrationActive = true
	a.joyConCalibrationMessage = "中心で静止した後、スティックを全方向へゆっくり倒してください。"
	return nil
}

func (a *App) finishJoyConCalibration() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.joyConCalibrationActive || a.joyConCalibration == nil {
		return fmt.Errorf("キャリブレーションは開始されていません。")
	}
	calibration, err := a.joyConCalibration.Result()
	if err != nil {
		a.joyConCalibrationMessage = err.Error()
		return err
	}
	profileIndex := a.profileIndexByIDLocked(a.joyConCalibration.profileID)
	if profileIndex < 0 {
		return fmt.Errorf("キャリブレーション開始時のプロファイルが削除されています。結果は保存していません。")
	}
	config := normalizeJoyConProfileConfig(a.config.Profiles[profileIndex].JoyCon)
	config.Stick.Calibration = calibration
	a.config.Profiles[profileIndex].JoyCon = config
	a.joyConCalibrationActive = false
	a.joyConCalibration = nil
	a.joyConCalibrationMessage = "完了: 開始時に選択していたプロファイルへ中心値と可動範囲を保存しました。"
	if err := a.saveConfigLocked(); err != nil {
		return err
	}
	if profileIndex == a.activeProfileIndex {
		a.requestJoyConRescanLocked()
	}
	return nil
}

func (a *App) cancelJoyConCalibration() {
	a.mu.Lock()
	a.joyConCalibrationActive = false
	a.joyConCalibration = nil
	a.joyConCalibrationMessage = "中止しました。"
	a.mu.Unlock()
}

func (a *App) collectJoyConCalibrationSampleLocked(status JoyConConnectionStatus) {
	if !a.joyConCalibrationActive || a.joyConCalibration == nil || !status.Connected {
		return
	}
	neutral := math.Abs(status.StickX) <= 0.18 && math.Abs(status.StickY) <= 0.18
	a.joyConCalibration.Add(status.RawStickX, status.RawStickY, neutral)
}
