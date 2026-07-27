//go:build windows

package main

import (
	"strings"
	"testing"
)

func TestStopJoyConSubsystemClearsWorkerReference(t *testing.T) {
	done := make(chan struct{})
	close(done)
	cancelled := false
	app := &App{
		joyConWorker: &JoyConWorker{},
		joyConCancel: func() { cancelled = true },
		joyConDone:   done,
	}

	app.stopJoyConSubsystem()

	if !cancelled {
		t.Fatal("Joy-Con worker cancel function was not called")
	}
	app.mu.RLock()
	defer app.mu.RUnlock()
	if app.joyConWorker != nil || app.joyConCancel != nil || app.joyConDone != nil {
		t.Fatalf("Joy-Con subsystem references were not cleared: worker=%p cancel=%v done=%v", app.joyConWorker, app.joyConCancel != nil, app.joyConDone != nil)
	}
}

func TestJoyConCalibrationSavesToProfileSelectedAtStart(t *testing.T) {
	app := newJoyConCalibrationTestApp()
	if err := app.startJoyConCalibration(); err != nil {
		t.Fatalf("startJoyConCalibration: %v", err)
	}

	app.mu.Lock()
	app.editorProfileIndex = 1
	addCompleteJoyConCalibrationSamples(app.joyConCalibration)
	app.mu.Unlock()

	if err := app.finishJoyConCalibration(); err != nil {
		t.Fatalf("finishJoyConCalibration: %v", err)
	}

	app.mu.RLock()
	defer app.mu.RUnlock()
	first := app.config.Profiles[0].JoyCon.Stick.Calibration
	second := app.config.Profiles[1].JoyCon.Stick.Calibration
	if first.X.Center != 2000 || first.X.Min != 500 || first.X.Max != 3500 {
		t.Fatalf("start profile calibration=%+v", first)
	}
	if second != defaultJoyConStickConfig().Calibration {
		t.Fatalf("editor profile switched during calibration was modified: %+v", second)
	}
	if app.joyConCalibration != nil || app.joyConCalibrationActive {
		t.Fatalf("calibration binding was not cleared: session=%p active=%v", app.joyConCalibration, app.joyConCalibrationActive)
	}
}

func TestJoyConCalibrationDoesNotFallThroughAfterBoundProfileDeletion(t *testing.T) {
	app := newJoyConCalibrationTestApp()
	if err := app.startJoyConCalibration(); err != nil {
		t.Fatalf("startJoyConCalibration: %v", err)
	}

	app.mu.Lock()
	addCompleteJoyConCalibrationSamples(app.joyConCalibration)
	app.config.Profiles = app.config.Profiles[1:]
	app.editorProfileIndex = 0
	app.mu.Unlock()

	err := app.finishJoyConCalibration()
	if err == nil || !strings.Contains(err.Error(), "削除") {
		t.Fatalf("finishJoyConCalibration error=%v", err)
	}
	app.mu.RLock()
	defer app.mu.RUnlock()
	if got := app.config.Profiles[0].JoyCon.Stick.Calibration; got != defaultJoyConStickConfig().Calibration {
		t.Fatalf("remaining profile received deleted profile calibration: %+v", got)
	}
}

func TestJoyConWebStateExposesStableEditorProfileIndex(t *testing.T) {
	app := newJoyConCalibrationTestApp()
	app.editorProfileIndex = 1
	state := app.buildJoyConWebState()
	if state.ProfileIndex != 1 || state.ProfileName != "Duplicate" {
		t.Fatalf("web state profile index/name=%d/%q", state.ProfileIndex, state.ProfileName)
	}
	if !strings.Contains(joyConUIJS, "Number(state.activeProfile)!==lastEditorProfile") {
		t.Fatal("Joy-Con UI does not use the main state's stable editor profile index")
	}
}

func newJoyConCalibrationTestApp() *App {
	stick := defaultJoyConStickConfig()
	return &App{
		config: Config{
			Version:         9,
			ActiveProfileId: "profile-a",
			Profiles: []Profile{
				{Id: "profile-a", Name: "Duplicate", JoyCon: JoyConProfileConfig{Stick: stick}},
				{Id: "profile-b", Name: "Duplicate", JoyCon: JoyConProfileConfig{Stick: stick}},
			},
		},
		editorProfileIndex: 0,
		activeProfileIndex: 0,
		joyConStatus: JoyConConnectionStatus{
			Connected: true,
		},
		configSaveCh: make(chan []byte, 2),
	}
}

func addCompleteJoyConCalibrationSamples(session *JoyConCalibrationSession) {
	session.Add(500, 500, false)
	session.Add(3500, 3500, false)
	for range 8 {
		session.Add(2000, 2000, true)
	}
}
