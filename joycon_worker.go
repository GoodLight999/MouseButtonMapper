package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const joyConStatusNotifyInterval = 50 * time.Millisecond

var errJoyConRescanRequested = errors.New("Joy-Con rescan requested")

type JoyConTransport interface {
	Device() JoyConDeviceInfo
	SetFullReportMode() error
	ReadState() (JoyConRawState, error)
	Close() error
}

type JoyConBackend interface {
	Enumerate() ([]JoyConDeviceInfo, error)
	Open(JoyConDeviceInfo) (JoyConTransport, error)
}

type JoyConWorkerOptions struct {
	Backend       JoyConBackend
	Config        func() JoyConProfileConfig
	Emit          func(InputEvent)
	StatusChanged func(JoyConConnectionStatus)
	Now           func() time.Time
}

type JoyConWorker struct {
	backend       JoyConBackend
	config        func() JoyConProfileConfig
	emit          func(InputEvent)
	statusChanged func(JoyConConnectionStatus)
	now           func() time.Time
	rescan        chan struct{}
	rescanPending atomic.Bool

	statusMu       sync.RWMutex
	status         JoyConConnectionStatus
	lastNotified   JoyConConnectionStatus
	lastNotifiedAt time.Time
	hasNotified    bool
}

func NewJoyConWorker(options JoyConWorkerOptions) (*JoyConWorker, error) {
	if options.Backend == nil {
		return nil, errors.New("Joy-Con worker backend is required")
	}
	if options.Config == nil {
		return nil, errors.New("Joy-Con worker config callback is required")
	}
	if options.Emit == nil {
		return nil, errors.New("Joy-Con worker event callback is required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &JoyConWorker{
		backend:       options.Backend,
		config:        options.Config,
		emit:          options.Emit,
		statusChanged: options.StatusChanged,
		now:           options.Now,
		rescan:        make(chan struct{}, 1),
		status: JoyConConnectionStatus{
			BatteryPercent: -1,
		},
	}, nil
}

func (w *JoyConWorker) RequestRescan() {
	if w == nil || !w.rescanPending.CompareAndSwap(false, true) {
		return
	}
	select {
	case w.rescan <- struct{}{}:
	default:
		w.rescanPending.Store(false)
	}
}

// completeRescanAttempt marks one coalesced manual rescan as handled. Drain the
// request signal before clearing the gate so a request made during enumeration,
// open, or report-mode negotiation cannot immediately tear down the connection
// that just satisfied it. Requests made after the gate is cleared remain queued.
func (w *JoyConWorker) completeRescanAttempt() {
	if w == nil || !w.rescanPending.Load() {
		return
	}
	for {
		select {
		case <-w.rescan:
		default:
			w.rescanPending.Store(false)
			return
		}
	}
}

func (w *JoyConWorker) Status() JoyConConnectionStatus {
	if w == nil {
		return JoyConConnectionStatus{BatteryPercent: -1}
	}
	w.statusMu.RLock()
	defer w.statusMu.RUnlock()
	return w.status
}

func (w *JoyConWorker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("Joy-Con worker is nil")
	}
	for {
		if ctx.Err() != nil {
			return nil
		}
		config := normalizeJoyConProfileConfig(w.config())
		if !config.Enabled {
			w.completeRescanAttempt()
			status := w.Status()
			status.Connected = false
			status.BatteryPercent = -1
			status.Charging = false
			status.LastError = ""
			w.publishStatus(status)
			if !w.waitForRescanOrStop(ctx) {
				return nil
			}
			continue
		}

		device, candidates, err := w.findDevice(config.PreferredDevice)
		if err != nil {
			w.completeRescanAttempt()
			w.publishCandidates(candidates)
			w.publishFailure(err)
			if !w.waitAfterFailure(ctx, config) {
				return nil
			}
			continue
		}

		w.publishCandidates(candidates)
		transport, err := w.backend.Open(device)
		if err != nil {
			w.completeRescanAttempt()
			w.publishCandidates(candidates)
			w.publishFailure(err)
			if !w.waitAfterFailure(ctx, config) {
				return nil
			}
			continue
		}

		if err := transport.SetFullReportMode(); err != nil {
			_ = transport.Close()
			w.completeRescanAttempt()
			w.publishFailure(fmt.Errorf("set Joy-Con full report mode: %w", err))
			if !w.waitAfterFailure(ctx, config) {
				return nil
			}
			continue
		}

		w.completeRescanAttempt()
		err = w.runConnected(ctx, transport, config)
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return nil
		}
		if errors.Is(err, errJoyConRescanRequested) {
			continue
		}
		if err != nil {
			w.publishFailure(err)
		}
		if !config.Reconnect.Enabled {
			if !w.waitForRescanOrStop(ctx) {
				return nil
			}
			continue
		}
		if !w.waitForRetry(ctx, config.Reconnect.IntervalMs) {
			return nil
		}
	}
}

func (w *JoyConWorker) waitAfterFailure(ctx context.Context, config JoyConProfileConfig) bool {
	if !config.Reconnect.Enabled {
		return w.waitForRescanOrStop(ctx)
	}
	return w.waitForRetry(ctx, config.Reconnect.IntervalMs)
}

func (w *JoyConWorker) findDevice(preferred string) (JoyConDeviceInfo, []JoyConDeviceInfo, error) {
	devices, err := w.backend.Enumerate()
	if err != nil {
		return JoyConDeviceInfo{}, nil, fmt.Errorf("enumerate Joy-Con HID devices: %w", err)
	}
	device, ok := chooseJoyConDevice(devices, preferred)
	if !ok {
		if len(devices) == 0 {
			return JoyConDeviceInfo{}, devices, errors.New("Joy-Con (L) or compatible HID gamepad is not connected")
		}
		names := make([]string, 0, len(devices))
		for _, candidate := range devices {
			names = append(names, candidate.DisplayName())
		}
		return JoyConDeviceInfo{}, devices, fmt.Errorf("Joy-Con (L) was not identified automatically; select a compatible candidate: %s", strings.Join(names, "; "))
	}
	return device, devices, nil
}

func (w *JoyConWorker) publishCandidates(candidates []JoyConDeviceInfo) {
	status := w.Status()
	status.Candidates = append([]JoyConDeviceInfo(nil), candidates...)
	w.publishStatus(status)
}

type joyConReadResult struct {
	state JoyConRawState
	err   error
}

func (w *JoyConWorker) runConnected(ctx context.Context, transport JoyConTransport, config JoyConProfileConfig) error {
	device := transport.Device()
	tracker := newJoyConStateTracker(device.StableID())
	readResults := make(chan joyConReadResult, 1)
	readDone := make(chan struct{})
	stopRead := make(chan struct{})
	var stopReadOnce sync.Once

	go func() {
		defer close(readDone)
		for {
			state, err := transport.ReadState()
			select {
			case readResults <- joyConReadResult{state: state, err: err}:
			case <-ctx.Done():
				return
			case <-stopRead:
				return
			}
			if err != nil {
				return
			}
		}
	}()

	previousStatus := w.Status()
	status := JoyConConnectionStatus{
		Connected:      true,
		Device:         device,
		Candidates:     append([]JoyConDeviceInfo(nil), previousStatus.Candidates...),
		BatteryPercent: -1,
		ReconnectCount: previousStatus.ReconnectCount + 1,
	}
	w.publishStatus(status)

	closeAndRelease := func() {
		stopReadOnce.Do(func() { close(stopRead) })
		_ = transport.Close()
		for _, event := range tracker.Disconnect(w.now()) {
			w.emit(event)
		}
		status.Connected = false
		status.BatteryPercent = -1
		status.Charging = false
		status.RawStickX = 0
		status.RawStickY = 0
		status.StickX = 0
		status.StickY = 0
		w.publishStatus(status)
		select {
		case <-readDone:
		case <-time.After(2 * time.Second):
		}
	}
	defer closeAndRelease()

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-w.rescan:
			return errJoyConRescanRequested
		case result := <-readResults:
			if result.err != nil {
				return fmt.Errorf("read Joy-Con input: %w", result.err)
			}
			at := w.now()
			for _, event := range tracker.Apply(result.state, config.Stick, at) {
				w.emit(event)
				status.LastInput = event.Token.Code
			}
			status.Connected = true
			status.Device = device
			status.BatteryPercent = result.state.BatteryPercent
			status.Charging = result.state.Charging
			status.RawStickX = result.state.StickX
			status.RawStickY = result.state.StickY
			status.StickX, status.StickY = normalizeJoyConStick(result.state.StickX, result.state.StickY, config.Stick)
			status.LastReportAt = at
			status.LastError = ""
			w.publishStatus(status)
		}
	}
}

func (w *JoyConWorker) waitForRetry(ctx context.Context, intervalMs int) bool {
	interval := time.Duration(normalizeJoyConProfileConfig(JoyConProfileConfig{
		Reconnect: JoyConReconnectConfig{IntervalMs: intervalMs},
	}).Reconnect.IntervalMs) * time.Millisecond
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-w.rescan:
		return true
	case <-timer.C:
		return true
	}
}

func (w *JoyConWorker) waitForRescanOrStop(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-w.rescan:
		return true
	}
}

func (w *JoyConWorker) publishFailure(err error) {
	status := w.Status()
	status.Connected = false
	status.BatteryPercent = -1
	status.Charging = false
	status.LastError = strings.TrimSpace(err.Error())
	w.publishStatus(status)
}

func (w *JoyConWorker) publishStatus(status JoyConConnectionStatus) {
	now := time.Now()
	if w.now != nil {
		now = w.now()
	}
	w.statusMu.Lock()
	w.status = status
	notify := !w.hasNotified ||
		status.Connected != w.lastNotified.Connected ||
		status.LastError != w.lastNotified.LastError ||
		status.LastInput != w.lastNotified.LastInput ||
		status.ReconnectCount != w.lastNotified.ReconnectCount ||
		now.Sub(w.lastNotifiedAt) >= joyConStatusNotifyInterval
	if notify {
		w.lastNotified = status
		w.lastNotifiedAt = now
		w.hasNotified = true
	}
	w.statusMu.Unlock()
	if notify && w.statusChanged != nil {
		w.statusChanged(status)
	}
}
