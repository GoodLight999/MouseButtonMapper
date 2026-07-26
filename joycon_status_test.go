package main

import (
	"testing"
	"time"
)

func TestJoyConStatusNotificationsAreThrottledButStateStaysCurrent(t *testing.T) {
	now := time.Unix(100, 0)
	notified := make([]JoyConConnectionStatus, 0, 4)
	worker := &JoyConWorker{
		now: func() time.Time { return now },
		statusChanged: func(status JoyConConnectionStatus) {
			notified = append(notified, status)
		},
	}

	status := JoyConConnectionStatus{Connected: true, BatteryPercent: 100, ReconnectCount: 1}
	worker.publishStatus(status)
	if len(notified) != 1 {
		t.Fatalf("initial notifications=%d", len(notified))
	}

	now = now.Add(10 * time.Millisecond)
	status.StickX = 0.5
	worker.publishStatus(status)
	if len(notified) != 1 {
		t.Fatalf("stick-only update was not throttled: notifications=%d", len(notified))
	}
	if got := worker.Status().StickX; got != 0.5 {
		t.Fatalf("throttled status was not stored: StickX=%f", got)
	}

	now = now.Add(joyConStatusNotifyInterval)
	status.StickX = 0.75
	worker.publishStatus(status)
	if len(notified) != 2 || notified[1].StickX != 0.75 {
		t.Fatalf("periodic notification=%#v", notified)
	}

	now = now.Add(time.Millisecond)
	status.LastInput = "ZL"
	worker.publishStatus(status)
	if len(notified) != 3 || notified[2].LastInput != "ZL" {
		t.Fatalf("input change was not notified immediately: %#v", notified)
	}

	now = now.Add(time.Millisecond)
	status.Connected = false
	status.LastError = "disconnected"
	worker.publishStatus(status)
	if len(notified) != 4 || notified[3].Connected {
		t.Fatalf("disconnect was not notified immediately: %#v", notified)
	}
}
