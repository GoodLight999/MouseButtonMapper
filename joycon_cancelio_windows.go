//go:build windows

package main

import "syscall"

const joyConErrorNotFound syscall.Errno = 1168

var joyConCancelIoEx = joyConKernel32DLL.NewProc("CancelIoEx")

func cancelJoyConPendingIO(handle uintptr) error {
	if handle == 0 || handle == ^uintptr(0) {
		return nil
	}
	ok, _, callErr := joyConCancelIoEx.Call(handle, 0)
	if ok != 0 {
		return nil
	}
	if errno, ok := callErr.(syscall.Errno); ok && (errno == 0 || errno == joyConErrorNotFound) {
		// ERROR_NOT_FOUND means there was no pending request to cancel.
		return nil
	}
	return joyConWindowsCallError("CancelIoEx", callErr)
}
