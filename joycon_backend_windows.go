//go:build windows

package main

type WindowsJoyConBackend struct{}

func (WindowsJoyConBackend) Enumerate() ([]JoyConDeviceInfo, error) {
	return EnumerateJoyConHIDDevices()
}

func (WindowsJoyConBackend) Open(device JoyConDeviceInfo) (JoyConTransport, error) {
	return OpenJoyConHIDSession(device)
}
