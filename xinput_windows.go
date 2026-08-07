//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	xInputErrorSuccess            = 0
	xInputErrorDeviceNotConnected = 1167
	xInputPollInterval            = 8 * time.Millisecond
)

type xInputNativeGamepad struct {
	Buttons      uint16
	LeftTrigger  uint8
	RightTrigger uint8
	LeftX        int16
	LeftY        int16
	RightX       int16
	RightY       int16
}

type xInputNativeState struct {
	PacketNumber uint32
	Gamepad      xInputNativeGamepad
}

type XInputConnectionStatus struct {
	Available      bool      `json:"Available"`
	DLL            string    `json:"DLL,omitempty"`
	ConnectedSlots [4]bool   `json:"ConnectedSlots"`
	LastError      string    `json:"LastError,omitempty"`
	LastPollAt     time.Time `json:"LastPollAt,omitempty"`
}

type xInputAPI struct {
	dll      *syscall.DLL
	getState *syscall.Proc
	name     string
}

func loadXInputAPI() (*xInputAPI, error) {
	candidates := []string{"xinput1_4.dll", "xinput1_3.dll", "xinput9_1_0.dll"}
	var failures []error
	for _, name := range candidates {
		dll, err := syscall.LoadDLL(name)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", name, err))
			continue
		}
		proc, err := dll.FindProc("XInputGetState")
		if err != nil {
			_ = dll.Release()
			failures = append(failures, fmt.Errorf("%s!XInputGetState: %w", name, err))
			continue
		}
		return &xInputAPI{dll: dll, getState: proc, name: name}, nil
	}
	return nil, fmt.Errorf("XInputGetState is unavailable: %w", errors.Join(failures...))
}

func (api *xInputAPI) Close() {
	if api != nil && api.dll != nil {
		_ = api.dll.Release()
	}
}

func (api *xInputAPI) State(slot int) (xInputNativeState, bool, error) {
	if api == nil || api.getState == nil {
		return xInputNativeState{}, false, errors.New("XInput API is unavailable")
	}
	var state xInputNativeState
	result, _, _ := api.getState.Call(uintptr(slot), uintptr(unsafe.Pointer(&state)))
	switch uint32(result) {
	case xInputErrorSuccess:
		return state, true, nil
	case xInputErrorDeviceNotConnected:
		return xInputNativeState{}, false, nil
	default:
		return xInputNativeState{}, false, fmt.Errorf("XInputGetState(P%d) returned %d", slot+1, uint32(result))
	}
}

type XInputWorker struct {
	api           *xInputAPI
	emit          func(InputEvent)
	statusChanged func(XInputConnectionStatus)
	trackers      [xInputMaxUsers]*xInputStateTracker
	connected     [xInputMaxUsers]bool
	packet        [xInputMaxUsers]uint32
	status        XInputConnectionStatus
}

func NewXInputWorker(emit func(InputEvent), statusChanged func(XInputConnectionStatus)) (*XInputWorker, error) {
	if emit == nil {
		return nil, errors.New("XInput event callback is required")
	}
	api, err := loadXInputAPI()
	if err != nil {
		return nil, err
	}
	worker := &XInputWorker{api: api, emit: emit, statusChanged: statusChanged}
	worker.status.Available = true
	worker.status.DLL = api.name
	for slot := range xInputMaxUsers {
		worker.trackers[slot] = newXInputStateTracker(slot)
	}
	return worker, nil
}

func (w *XInputWorker) Run(ctx context.Context) error {
	if w == nil || w.api == nil {
		return errors.New("XInput worker is not initialized")
	}
	defer w.api.Close()
	w.publishStatus()
	ticker := time.NewTicker(xInputPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			w.disconnectAll(time.Now())
			return nil
		case at := <-ticker.C:
			w.poll(at)
		}
	}
}

func (w *XInputWorker) poll(at time.Time) {
	changedStatus := false
	errorsBySlot := make([]string, 0, xInputMaxUsers)
	for slot := range xInputMaxUsers {
		native, connected, err := w.api.State(slot)
		if err != nil {
			errorsBySlot = append(errorsBySlot, err.Error())
			// Treat an API error like a temporary disconnect for safety. Retaining
			// the previous state could otherwise leave a Hold output pressed forever.
			if w.connected[slot] {
				for _, event := range w.trackers[slot].Disconnect(at) {
					w.emit(event)
				}
				w.connected[slot] = false
				w.status.ConnectedSlots[slot] = false
				changedStatus = true
			}
			continue
		}
		if !connected {
			if w.connected[slot] {
				for _, event := range w.trackers[slot].Disconnect(at) {
					w.emit(event)
				}
				w.connected[slot] = false
				w.status.ConnectedSlots[slot] = false
				changedStatus = true
			}
			continue
		}
		if !w.connected[slot] {
			w.connected[slot] = true
			w.status.ConnectedSlots[slot] = true
			changedStatus = true
		} else if w.packet[slot] == native.PacketNumber {
			continue
		}
		w.packet[slot] = native.PacketNumber
		state := xInputGamepadState{
			Buttons:      native.Gamepad.Buttons,
			LeftTrigger:  native.Gamepad.LeftTrigger,
			RightTrigger: native.Gamepad.RightTrigger,
			LeftX:        native.Gamepad.LeftX,
			LeftY:        native.Gamepad.LeftY,
			RightX:       native.Gamepad.RightX,
			RightY:       native.Gamepad.RightY,
		}
		for _, event := range w.trackers[slot].Update(state, at) {
			w.emit(event)
		}
	}
	message := strings.Join(errorsBySlot, "; ")
	if w.status.LastError != message {
		w.status.LastError = message
		changedStatus = true
	}
	w.status.LastPollAt = at
	if changedStatus {
		w.publishStatus()
	}
}

func (w *XInputWorker) disconnectAll(at time.Time) {
	for slot := range xInputMaxUsers {
		if !w.connected[slot] {
			continue
		}
		for _, event := range w.trackers[slot].Disconnect(at) {
			w.emit(event)
		}
		w.connected[slot] = false
		w.status.ConnectedSlots[slot] = false
	}
	w.publishStatus()
}

func (w *XInputWorker) publishStatus() {
	if w.statusChanged != nil {
		w.statusChanged(w.status)
	}
}

func (a *App) startXInputSubsystem() {
	a.mu.RLock()
	enabled := a.config.Controller.Enabled
	alreadyRunning := a.xInputCancel != nil
	a.mu.RUnlock()
	if !enabled || alreadyRunning {
		return
	}
	worker, err := NewXInputWorker(a.handleXInputInputEvent, func(status XInputConnectionStatus) {
		a.mu.Lock()
		a.xInputStatus = status
		a.postActivityRefreshLocked()
		a.mu.Unlock()
	})
	if err != nil {
		a.mu.Lock()
		a.xInputStatus = XInputConnectionStatus{LastError: err.Error()}
		a.mu.Unlock()
		a.logf("XInput worker init failed: %v", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	a.mu.Lock()
	if !a.config.Controller.Enabled || a.xInputCancel != nil {
		a.mu.Unlock()
		cancel()
		return
	}
	a.xInputCancel = cancel
	a.xInputDone = done
	a.xInputStatus = XInputConnectionStatus{Available: true, DLL: worker.api.name}
	a.mu.Unlock()
	a.workerWG.Add(1)
	go func() {
		defer a.workerWG.Done()
		defer close(done)
		if err := worker.Run(ctx); err != nil {
			a.logf("XInput worker stopped with error: %v", err)
		}
	}()
}

func (a *App) stopXInputSubsystem() {
	a.mu.Lock()
	cancel := a.xInputCancel
	done := a.xInputDone
	a.xInputCancel = nil
	a.xInputDone = nil
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			a.logf("XInput worker shutdown timeout")
		}
	}
}

func (a *App) handleXInputInputEvent(event InputEvent) {
	a.handleControllerInputEvent("XInput", event)
}

func (a *App) xInputStatusTextLocked() string {
	if !a.config.Controller.Enabled {
		return "XInput機能は無効"
	}
	if a.xInputStatus.LastError != "" {
		return "XInput利用不可: " + a.xInputStatus.LastError
	}
	slots := make([]string, 0, xInputMaxUsers)
	for slot, connected := range a.xInputStatus.ConnectedSlots {
		if connected {
			slots = append(slots, fmt.Sprintf("P%d", slot+1))
		}
	}
	if len(slots) == 0 {
		if a.xInputStatus.Available {
			return "XInput待機中 (" + a.xInputStatus.DLL + ")"
		}
		return "XInput未初期化"
	}
	return "XInput接続中: " + strings.Join(slots, ", ") + " (" + a.xInputStatus.DLL + ")"
}
