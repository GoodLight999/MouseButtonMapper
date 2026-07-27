package main

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

type fakeJoyConBackend struct {
	mu           sync.Mutex
	devices      []JoyConDeviceInfo
	enumerateErr error
	transports   []*fakeJoyConTransport
	enumerates   int
	opens        int
}

func (b *fakeJoyConBackend) Enumerate() ([]JoyConDeviceInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.enumerates++
	if b.enumerateErr != nil {
		return nil, b.enumerateErr
	}
	return append([]JoyConDeviceInfo(nil), b.devices...), nil
}

func (b *fakeJoyConBackend) Open(device JoyConDeviceInfo) (JoyConTransport, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.opens++
	if len(b.transports) == 0 {
		return nil, errors.New("no fake Joy-Con transport")
	}
	transport := b.transports[0]
	b.transports = b.transports[1:]
	transport.device = device
	return transport, nil
}

func (b *fakeJoyConBackend) counts() (int, int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.enumerates, b.opens
}

type fakeJoyConTransport struct {
	device        JoyConDeviceInfo
	modeErr       error
	reads         chan joyConReadResult
	closed        chan struct{}
	closeOnce     sync.Once
	modeMu        sync.Mutex
	modeCalls     int
	modeStarted   chan struct{}
	modeRelease   chan struct{}
	modeStartOnce sync.Once
}

func newFakeJoyConTransport() *fakeJoyConTransport {
	return &fakeJoyConTransport{
		reads:  make(chan joyConReadResult, 16),
		closed: make(chan struct{}),
	}
}

func (t *fakeJoyConTransport) Device() JoyConDeviceInfo {
	return t.device
}

func (t *fakeJoyConTransport) SetFullReportMode() error {
	t.modeMu.Lock()
	t.modeCalls++
	t.modeMu.Unlock()
	if t.modeStarted != nil {
		t.modeStartOnce.Do(func() { close(t.modeStarted) })
	}
	if t.modeRelease != nil {
		<-t.modeRelease
	}
	return t.modeErr
}

func (t *fakeJoyConTransport) ReadState() (JoyConRawState, error) {
	select {
	case result := <-t.reads:
		return result.state, result.err
	case <-t.closed:
		return JoyConRawState{}, errJoyConFakeClosed
	}
}

func (t *fakeJoyConTransport) Close() error {
	t.closeOnce.Do(func() { close(t.closed) })
	return nil
}

func (t *fakeJoyConTransport) modeCallCount() int {
	t.modeMu.Lock()
	defer t.modeMu.Unlock()
	return t.modeCalls
}

var errJoyConFakeClosed = errors.New("fake Joy-Con closed")

func TestJoyConWorkerReadsAndReleasesOnDisconnect(t *testing.T) {
	device := testJoyConDevice("left-a")
	transport := newFakeJoyConTransport()
	backend := &fakeJoyConBackend{
		devices:    []JoyConDeviceInfo{device},
		transports: []*fakeJoyConTransport{transport},
	}
	events := make(chan InputEvent, 16)
	statuses := make(chan JoyConConnectionStatus, 16)
	config := defaultJoyConProfileConfig()
	config.Enabled = true
	config.Reconnect.Enabled = false

	worker, err := NewJoyConWorker(JoyConWorkerOptions{
		Backend: backend,
		Config:  func() JoyConProfileConfig { return config },
		Emit:    func(event InputEvent) { events <- event },
		StatusChanged: func(status JoyConConnectionStatus) {
			statuses <- status
		},
	})
	if err != nil {
		t.Fatalf("NewJoyConWorker: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	waitJoyConConnected(t, statuses)

	down := neutralJoyConRawState()
	down.Buttons[JoyConButtonZL] = true
	transport.reads <- joyConReadResult{state: down}
	assertWorkerEvent(t, events, JoyConButtonZL, true, false)

	transport.reads <- joyConReadResult{err: io.EOF}
	assertWorkerEvent(t, events, JoyConButtonZL, false, true)
	waitJoyConDisconnectedWithError(t, statuses)

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Joy-Con worker did not stop")
	}
	if transport.modeCallCount() != 1 {
		t.Fatalf("SetFullReportMode calls=%d", transport.modeCallCount())
	}
}

func TestJoyConWorkerRescanReconnectsWithoutParallelOpen(t *testing.T) {
	device := testJoyConDevice("left-a")
	first := newFakeJoyConTransport()
	second := newFakeJoyConTransport()
	second.modeStarted = make(chan struct{})
	second.modeRelease = make(chan struct{})
	backend := &fakeJoyConBackend{
		devices:    []JoyConDeviceInfo{device},
		transports: []*fakeJoyConTransport{first, second},
	}
	statuses := make(chan JoyConConnectionStatus, 32)
	config := defaultJoyConProfileConfig()
	config.Enabled = true
	config.Reconnect.IntervalMs = minJoyConReconnectMs

	worker, err := NewJoyConWorker(JoyConWorkerOptions{
		Backend: backend,
		Config:  func() JoyConProfileConfig { return config },
		Emit:    func(InputEvent) {},
		StatusChanged: func(status JoyConConnectionStatus) {
			statuses <- status
		},
	})
	if err != nil {
		t.Fatalf("NewJoyConWorker: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	firstStatus := waitJoyConConnected(t, statuses)
	if firstStatus.ReconnectCount != 1 {
		t.Fatalf("first ReconnectCount=%d", firstStatus.ReconnectCount)
	}

	worker.RequestRescan()
	select {
	case <-first.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("first transport did not close for manual rescan")
	}
	select {
	case <-second.modeStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("replacement transport did not begin report-mode negotiation")
	}

	// The first request has been consumed and the replacement connection is now
	// deliberately held in report-mode negotiation. A second request in this
	// window must coalesce with the in-flight rescan.
	worker.RequestRescan()
	close(second.modeRelease)
	secondStatus := waitJoyConReconnectCount(t, statuses, 2)
	if secondStatus.Device.Serial != "left-a" {
		t.Fatalf("second device=%+v", secondStatus.Device)
	}
	select {
	case <-second.closed:
		t.Fatal("coalesced rescan immediately closed the replacement transport")
	case <-time.After(100 * time.Millisecond):
	}
	enumerates, opens := backend.counts()
	if enumerates != 2 || opens != 2 {
		t.Fatalf("coalesced rescan counts: enumerates=%d opens=%d", enumerates, opens)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Joy-Con worker did not stop after cancel")
	}
}

func TestJoyConWorkerReportModeFailureClosesAndRetries(t *testing.T) {
	device := testJoyConDevice("left-a")
	failed := newFakeJoyConTransport()
	failed.modeErr = errors.New("write failed")
	success := newFakeJoyConTransport()
	backend := &fakeJoyConBackend{
		devices:    []JoyConDeviceInfo{device},
		transports: []*fakeJoyConTransport{failed, success},
	}
	statuses := make(chan JoyConConnectionStatus, 32)
	config := defaultJoyConProfileConfig()
	config.Enabled = true
	config.Reconnect.IntervalMs = minJoyConReconnectMs

	worker, err := NewJoyConWorker(JoyConWorkerOptions{
		Backend: backend,
		Config:  func() JoyConProfileConfig { return config },
		Emit:    func(InputEvent) {},
		StatusChanged: func(status JoyConConnectionStatus) {
			statuses <- status
		},
	})
	if err != nil {
		t.Fatalf("NewJoyConWorker: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()

	waitJoyConDisconnectedWithError(t, statuses)
	waitJoyConConnected(t, statuses)
	select {
	case <-failed.closed:
	default:
		t.Fatal("failed transport was not closed")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Joy-Con worker did not stop")
	}
}

func TestNewJoyConWorkerValidatesCallbacks(t *testing.T) {
	if _, err := NewJoyConWorker(JoyConWorkerOptions{}); err == nil {
		t.Fatal("missing backend unexpectedly succeeded")
	}
	backend := &fakeJoyConBackend{}
	if _, err := NewJoyConWorker(JoyConWorkerOptions{Backend: backend}); err == nil {
		t.Fatal("missing config unexpectedly succeeded")
	}
	if _, err := NewJoyConWorker(JoyConWorkerOptions{
		Backend: backend,
		Config:  func() JoyConProfileConfig { return JoyConProfileConfig{} },
	}); err == nil {
		t.Fatal("missing emit unexpectedly succeeded")
	}
}

func testJoyConDevice(serial string) JoyConDeviceInfo {
	return JoyConDeviceInfo{
		Path:        `\\?\hid#vid_057e&pid_2006#` + serial,
		Fingerprint: fingerprintJoyConDevicePath(serial),
		VendorID:    joyConNintendoVendorID,
		ProductID:   joyConLeftProductID,
		Product:     "Joy-Con (L)",
		Serial:      serial,
	}
}

func waitJoyConConnected(t *testing.T, statuses <-chan JoyConConnectionStatus) JoyConConnectionStatus {
	t.Helper()
	return waitJoyConStatus(t, statuses, func(status JoyConConnectionStatus) bool {
		return status.Connected
	})
}

func waitJoyConReconnectCount(t *testing.T, statuses <-chan JoyConConnectionStatus, count uint64) JoyConConnectionStatus {
	t.Helper()
	return waitJoyConStatus(t, statuses, func(status JoyConConnectionStatus) bool {
		return status.Connected && status.ReconnectCount == count
	})
}

func waitJoyConDisconnectedWithError(t *testing.T, statuses <-chan JoyConConnectionStatus) JoyConConnectionStatus {
	t.Helper()
	return waitJoyConStatus(t, statuses, func(status JoyConConnectionStatus) bool {
		return !status.Connected && status.LastError != ""
	})
}

func waitJoyConStatus(t *testing.T, statuses <-chan JoyConConnectionStatus, predicate func(JoyConConnectionStatus) bool) JoyConConnectionStatus {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case status := <-statuses:
			if predicate(status) {
				return status
			}
		case <-timer.C:
			t.Fatal("timed out waiting for Joy-Con status")
		}
	}
}

func assertWorkerEvent(t *testing.T, events <-chan InputEvent, button JoyConButton, down, synthetic bool) {
	t.Helper()
	select {
	case event := <-events:
		if JoyConButton(event.Token.Code) != button || event.Down != down || event.Synthetic != synthetic {
			t.Fatalf("event=%+v want button=%s down=%v synthetic=%v", event, button, down, synthetic)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for event button=%s down=%v", button, down)
	}
}
