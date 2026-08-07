package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

type lockedJoyConConfig struct {
	mu     sync.RWMutex
	config JoyConProfileConfig
}

func (c *lockedJoyConConfig) Get() JoyConProfileConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

func (c *lockedJoyConConfig) Set(config JoyConProfileConfig) {
	c.mu.Lock()
	c.config = config
	c.mu.Unlock()
}

func TestJoyConWorkerDoesNotEnumerateWhileDisabled(t *testing.T) {
	backend := &fakeJoyConBackend{}
	config := defaultJoyConProfileConfig()
	config.Enabled = false
	holder := &lockedJoyConConfig{config: config}
	statuses := make(chan JoyConConnectionStatus, 8)

	worker, err := NewJoyConWorker(JoyConWorkerOptions{
		Backend: backend,
		Config:  holder.Get,
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

	waitJoyConStatus(t, statuses, func(status JoyConConnectionStatus) bool {
		return !status.Connected && status.LastError == ""
	})
	if enumerates, opens := backend.counts(); enumerates != 0 || opens != 0 {
		t.Fatalf("disabled worker touched HID backend: enumerates=%d opens=%d", enumerates, opens)
	}

	config.Enabled = true
	config.Reconnect.Enabled = false
	holder.Set(config)
	worker.RequestRescan()
	waitJoyConStatus(t, statuses, func(status JoyConConnectionStatus) bool {
		return !status.Connected && status.LastError != ""
	})
	if enumerates, opens := backend.counts(); enumerates != 1 || opens != 0 {
		t.Fatalf("manual enable/rescan counts: enumerates=%d opens=%d", enumerates, opens)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("disabled Joy-Con worker did not stop")
	}
}

func TestJoyConWorkerReconnectDisabledWaitsForManualRescan(t *testing.T) {
	backend := &fakeJoyConBackend{}
	config := defaultJoyConProfileConfig()
	config.Enabled = true
	config.Reconnect.Enabled = false
	statuses := make(chan JoyConConnectionStatus, 8)

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

	waitJoyConDisconnectedWithError(t, statuses)
	if enumerates, _ := backend.counts(); enumerates != 1 {
		t.Fatalf("initial enumerates=%d", enumerates)
	}

	time.Sleep(2 * time.Duration(minJoyConReconnectMs) * time.Millisecond)
	if enumerates, _ := backend.counts(); enumerates != 1 {
		t.Fatalf("reconnect-disabled worker retried automatically: enumerates=%d", enumerates)
	}

	worker.RequestRescan()
	waitForJoyConBackendEnumerates(t, backend, 2)

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("manual-rescan Joy-Con worker did not stop")
	}
}

func waitForJoyConBackendEnumerates(t *testing.T, backend *fakeJoyConBackend, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		enumerates, _ := backend.counts()
		if enumerates >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	enumerates, _ := backend.counts()
	t.Fatalf("enumerates=%d want at least %d", enumerates, want)
}
