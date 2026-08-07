//go:build windows

package main

import "testing"

func TestOutputRecordingCapturesMouseWheel(t *testing.T) {
	a := &App{recordingMode: "output"}
	finished := a.recordWheelLocked(Item{Kind: "Mouse", Code: "WheelDown"}, "検出")
	if !finished {
		t.Fatal("wheel output recording should finish immediately")
	}
	if len(a.recordedItems) != 1 || a.recordedItems[0] != (Item{Kind: "Mouse", Code: "WheelDown"}) {
		t.Fatalf("recordedItems=%#v", a.recordedItems)
	}
}

func TestOutputRecordingCapturesSideButton(t *testing.T) {
	a := &App{recordingMode: "output"}
	a.recordMouseDownLocked("X1")
	if len(a.recordedItems) != 1 || a.recordedItems[0] != (Item{Kind: "Mouse", Code: "X1"}) {
		t.Fatalf("recordedItems after down=%#v", a.recordedItems)
	}
	if !a.recordHeld["Mouse:X1"] {
		t.Fatalf("recordHeld=%#v", a.recordHeld)
	}
	if !a.recordUpLocked(Item{Kind: "Mouse", Code: "X1"}, "解放") {
		t.Fatal("side-button output recording should finish after release")
	}
}

func TestOutputRecordingKeepsPrimaryClicksForUI(t *testing.T) {
	a := &App{recordingMode: "output"}
	a.recordMouseDownLocked("Left")
	if len(a.recordedItems) != 0 {
		t.Fatalf("primary click should not be captured while operating the UI: %#v", a.recordedItems)
	}
	if len(a.recordHeld) != 0 {
		t.Fatalf("primary click should not block completion: %#v", a.recordHeld)
	}
}
