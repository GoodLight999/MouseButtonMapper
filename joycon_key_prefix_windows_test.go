//go:build windows

package main

import "testing"

func TestKeyboardTriggerConsumesJoyConPrefix(t *testing.T) {
	app := newKeyboardPrefixTestApp(Rule{
		Enabled: true,
		Input: []Item{
			{Kind: "JoyCon", Code: "ZL"},
			{Kind: "Key", Code: "R"},
		},
		Output: []Item{{Kind: "Key", Code: "A"}},
	})
	app.controllerDown["JoyCon:ZL"] = true

	app.handleKeyEvent('R', true)
	if !app.controllerConsumed["JoyCon:ZL"] {
		t.Fatal("Joy-Con prefix was not marked consumed by a keyboard-triggered combination")
	}
}

func TestKeyboardTriggerConsumesMousePrefix(t *testing.T) {
	app := newKeyboardPrefixTestApp(Rule{
		Enabled: true,
		Input: []Item{
			{Kind: "Mouse", Code: "X1"},
			{Kind: "Key", Code: "R"},
		},
		Output: []Item{{Kind: "Key", Code: "A"}},
	})
	app.mouseDown["X1"] = true

	app.handleKeyEvent('R', true)
	if !app.consumedPrefix["X1"] {
		t.Fatal("mouse prefix was not marked consumed by a keyboard-triggered combination")
	}
}

func newKeyboardPrefixTestApp(rule Rule) *App {
	return &App{
		enabled:             true,
		rules:               []Rule{rule},
		mouseDown:           map[string]bool{},
		keyDown:             map[uint32]bool{},
		controllerDown:      map[string]bool{},
		controllerPending:   map[string]bool{},
		controllerConsumed:  map[string]bool{},
		controllerHoldRules: map[string]Rule{},
		pendingTap:          map[string]bool{},
		consumedPrefix:      map[string]bool{},
		suppressedDown:      map[string]bool{},
		longPress:           map[string]*longPressState{},
		recordHeld:          map[string]bool{},
		actionCh:            make(chan outputJob, 4),
		shutdownCh:          make(chan struct{}),
		configSaveCh:        make(chan []byte, 1),
		joyConOutputRefs:    map[uint32]joyConOutputReference{},
	}
}
