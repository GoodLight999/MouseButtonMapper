//go:build windows

package main

func (a *App) releaseJoyConHeldOutputs() {
	a.sendMu.Lock()
	defer a.sendMu.Unlock()
	inputs := a.appendJoyConHeldReleaseInputs(nil)
	a.callSendInput(inputs)
}
