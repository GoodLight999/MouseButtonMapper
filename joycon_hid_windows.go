//go:build windows

package main

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	joyConDIGCFPresent         = 0x00000002
	joyConDIGCFDeviceInterface = 0x00000010
	joyConGenericRead          = 0x80000000
	joyConGenericWrite         = 0x40000000
	joyConFileShareRead        = 0x00000001
	joyConFileShareWrite       = 0x00000002
	joyConOpenExisting         = 3
	joyConErrorNoMoreItems     = 259
)

var errJoyConSessionClosed = errors.New("Joy-Con HID session is closed")

var (
	joyConHIDDLL      = syscall.NewLazyDLL("hid.dll")
	joyConSetupAPIDLL = syscall.NewLazyDLL("setupapi.dll")
	joyConKernel32DLL = syscall.NewLazyDLL("kernel32.dll")

	joyConHidDGetHidGuid        = joyConHIDDLL.NewProc("HidD_GetHidGuid")
	joyConHidDGetAttributes     = joyConHIDDLL.NewProc("HidD_GetAttributes")
	joyConHidDGetProductString  = joyConHIDDLL.NewProc("HidD_GetProductString")
	joyConHidDGetSerialString   = joyConHIDDLL.NewProc("HidD_GetSerialNumberString")
	joyConHidDGetPreparsedData  = joyConHIDDLL.NewProc("HidD_GetPreparsedData")
	joyConHidDFreePreparsedData = joyConHIDDLL.NewProc("HidD_FreePreparsedData")
	joyConHidPGetCaps           = joyConHIDDLL.NewProc("HidP_GetCaps")

	joyConSetupDiGetClassDevs             = joyConSetupAPIDLL.NewProc("SetupDiGetClassDevsW")
	joyConSetupDiEnumDeviceInterfaces     = joyConSetupAPIDLL.NewProc("SetupDiEnumDeviceInterfaces")
	joyConSetupDiGetDeviceInterfaceDetail = joyConSetupAPIDLL.NewProc("SetupDiGetDeviceInterfaceDetailW")
	joyConSetupDiDestroyDeviceInfoList    = joyConSetupAPIDLL.NewProc("SetupDiDestroyDeviceInfoList")

	joyConCreateFile  = joyConKernel32DLL.NewProc("CreateFileW")
	joyConReadFile    = joyConKernel32DLL.NewProc("ReadFile")
	joyConWriteFile   = joyConKernel32DLL.NewProc("WriteFile")
	joyConCloseHandle = joyConKernel32DLL.NewProc("CloseHandle")
)

type joyConGUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type joyConDeviceInterfaceData struct {
	CbSize             uint32
	InterfaceClassGUID joyConGUID
	Flags              uint32
	Reserved           uintptr
}

type joyConHIDDAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

type joyConHIDPCaps struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

const joyConHIDPStatusSuccess = 0x00110000

func EnumerateJoyConHIDDevices() ([]JoyConDeviceInfo, error) {
	var classGUID joyConGUID
	joyConHidDGetHidGuid.Call(uintptr(unsafe.Pointer(&classGUID)))

	deviceInfoSet, _, callErr := joyConSetupDiGetClassDevs.Call(
		uintptr(unsafe.Pointer(&classGUID)),
		0,
		0,
		joyConDIGCFPresent|joyConDIGCFDeviceInterface,
	)
	if deviceInfoSet == 0 || deviceInfoSet == ^uintptr(0) {
		return nil, joyConWindowsCallError("SetupDiGetClassDevsW", callErr)
	}
	defer joyConSetupDiDestroyDeviceInfoList.Call(deviceInfoSet)

	devices := make([]JoyConDeviceInfo, 0, 4)
	for index := uint32(0); ; index++ {
		interfaceData := joyConDeviceInterfaceData{CbSize: uint32(unsafe.Sizeof(joyConDeviceInterfaceData{}))}
		ok, _, enumErr := joyConSetupDiEnumDeviceInterfaces.Call(
			deviceInfoSet, 0, uintptr(unsafe.Pointer(&classGUID)), uintptr(index), uintptr(unsafe.Pointer(&interfaceData)),
		)
		if ok == 0 {
			if errno, ok := enumErr.(syscall.Errno); ok && errno == joyConErrorNoMoreItems {
				break
			}
			return devices, joyConWindowsCallError("SetupDiEnumDeviceInterfaces", enumErr)
		}

		path, err := joyConDevicePath(deviceInfoSet, &interfaceData)
		if err != nil {
			continue
		}
		info, err := inspectJoyConHIDPath(path)
		if err != nil {
			continue
		}
		// Keep only top-level joystick/gamepad collections plus explicit Nintendo
		// identities. This exposes compatible clones for manual selection without
		// ever probing keyboards, mice, consumer controls, or vendor-only HID nodes.
		if !info.IsGameControllerCollection() && !info.MightBeLeftJoyCon() {
			continue
		}
		if !info.IsLeftJoyCon() && info.VendorID == joyConNintendoVendorID && info.ProductID == joyConProProductID {
			if controllerType, probeErr := probeJoyConControllerType(path); probeErr == nil && controllerType == joyConTypeLeft {
				info.ControllerType = controllerType
			}
		}
		devices = append(devices, info)
	}
	return devices, nil
}

func joyConDevicePath(deviceInfoSet uintptr, interfaceData *joyConDeviceInterfaceData) (string, error) {
	var requiredSize uint32
	joyConSetupDiGetDeviceInterfaceDetail.Call(
		deviceInfoSet,
		uintptr(unsafe.Pointer(interfaceData)),
		0,
		0,
		uintptr(unsafe.Pointer(&requiredSize)),
		0,
	)
	if requiredSize < 6 {
		return "", fmt.Errorf("SetupDiGetDeviceInterfaceDetailW returned invalid size %d", requiredSize)
	}

	detail := make([]byte, requiredSize)
	cbSize := uint32(6)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		cbSize = 8
	}
	*(*uint32)(unsafe.Pointer(&detail[0])) = cbSize

	ok, _, callErr := joyConSetupDiGetDeviceInterfaceDetail.Call(
		deviceInfoSet,
		uintptr(unsafe.Pointer(interfaceData)),
		uintptr(unsafe.Pointer(&detail[0])),
		uintptr(requiredSize),
		uintptr(unsafe.Pointer(&requiredSize)),
		0,
	)
	if ok == 0 {
		return "", joyConWindowsCallError("SetupDiGetDeviceInterfaceDetailW", callErr)
	}

	pathWords := unsafe.Slice((*uint16)(unsafe.Pointer(&detail[4])), (len(detail)-4)/2)
	path := syscall.UTF16ToString(pathWords)
	if path == "" {
		return "", errors.New("Joy-Con HID device path is empty")
	}
	return path, nil
}

func inspectJoyConHIDPath(path string) (JoyConDeviceInfo, error) {
	handle, err := openJoyConWindowsHandle(path, 0)
	if err != nil {
		return JoyConDeviceInfo{}, err
	}
	defer closeJoyConWindowsHandle(handle)

	attributes := joyConHIDDAttributes{Size: uint32(unsafe.Sizeof(joyConHIDDAttributes{}))}
	ok, _, callErr := joyConHidDGetAttributes.Call(handle, uintptr(unsafe.Pointer(&attributes)))
	if ok == 0 {
		return JoyConDeviceInfo{}, joyConWindowsCallError("HidD_GetAttributes", callErr)
	}

	info := JoyConDeviceInfo{
		Path:        path,
		Fingerprint: fingerprintJoyConDevicePath(path),
		VendorID:    attributes.VendorID,
		ProductID:   attributes.ProductID,
		Version:     attributes.VersionNumber,
		Product:     joyConHIDString(joyConHidDGetProductString, handle),
		Serial:      joyConHIDString(joyConHidDGetSerialString, handle),
	}
	if caps, capsErr := joyConHIDCaps(handle); capsErr == nil {
		info.UsagePage = caps.UsagePage
		info.Usage = caps.Usage
		info.InputReportLength = caps.InputReportByteLength
		info.OutputReportLength = caps.OutputReportByteLength
	}
	if (info.VendorID == joyConNintendoVendorID && info.ProductID == joyConLeftProductID) || isExplicitLeftJoyConProduct(info.Product) {
		info.ControllerType = joyConTypeLeft
	}
	return info, nil
}

func joyConHIDCaps(handle uintptr) (joyConHIDPCaps, error) {
	var preparsed uintptr
	ok, _, callErr := joyConHidDGetPreparsedData.Call(handle, uintptr(unsafe.Pointer(&preparsed)))
	if ok == 0 || preparsed == 0 {
		return joyConHIDPCaps{}, joyConWindowsCallError("HidD_GetPreparsedData", callErr)
	}
	defer joyConHidDFreePreparsedData.Call(preparsed)
	var caps joyConHIDPCaps
	status, _, _ := joyConHidPGetCaps.Call(preparsed, uintptr(unsafe.Pointer(&caps)))
	if uint32(status) != joyConHIDPStatusSuccess {
		return joyConHIDPCaps{}, fmt.Errorf("HidP_GetCaps failed: status=0x%08x", uint32(status))
	}
	return caps, nil
}

type joyConProbeResult struct {
	controllerType uint8
	err            error
}

func probeJoyConControllerType(path string) (uint8, error) {
	handle, err := openJoyConWindowsHandle(path, joyConGenericRead|joyConGenericWrite)
	if err != nil {
		return 0, err
	}
	defer closeJoyConWindowsHandle(handle)
	report, err := buildJoyConDeviceInfoCommand(0)
	if err != nil {
		return 0, err
	}
	var written uint32
	ok, _, callErr := joyConWriteFile.Call(
		handle,
		uintptr(unsafe.Pointer(&report[0])),
		uintptr(len(report)),
		uintptr(unsafe.Pointer(&written)),
		0,
	)
	if ok == 0 {
		return 0, joyConWindowsCallError("WriteFile device-info probe", callErr)
	}
	if int(written) != len(report) {
		return 0, fmt.Errorf("Joy-Con device-info probe wrote %d of %d bytes", written, len(report))
	}

	resultCh := make(chan joyConProbeResult, 1)
	go func() {
		buffer := make([]byte, joyConInputReportLength)
		for {
			var read uint32
			ok, _, readErr := joyConReadFile.Call(
				handle,
				uintptr(unsafe.Pointer(&buffer[0])),
				uintptr(len(buffer)),
				uintptr(unsafe.Pointer(&read)),
				0,
			)
			if ok == 0 {
				resultCh <- joyConProbeResult{err: joyConWindowsCallError("ReadFile device-info probe", readErr)}
				return
			}
			if controllerType, matched := parseJoyConControllerTypeReply(buffer[:read]); matched {
				resultCh <- joyConProbeResult{controllerType: controllerType}
				return
			}
		}
	}()

	timer := time.NewTimer(200 * time.Millisecond)
	defer timer.Stop()
	select {
	case result := <-resultCh:
		return result.controllerType, result.err
	case <-timer.C:
		_ = cancelJoyConPendingIO(handle)
		select {
		case <-resultCh:
		case <-time.After(100 * time.Millisecond):
		}
		return 0, errors.New("Joy-Con device-info probe timed out")
	}
}

func joyConHIDString(proc *syscall.LazyProc, handle uintptr) string {
	buffer := make([]uint16, 126)
	ok, _, _ := proc.Call(
		handle,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)*2),
	)
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(buffer)
}

func openJoyConWindowsHandle(path string, access uintptr) (uintptr, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, fmt.Errorf("invalid Joy-Con HID path: %w", err)
	}
	handle, _, callErr := joyConCreateFile.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		access,
		joyConFileShareRead|joyConFileShareWrite,
		0,
		joyConOpenExisting,
		0,
		0,
	)
	if handle == 0 || handle == ^uintptr(0) {
		return 0, joyConWindowsCallError("CreateFileW", callErr)
	}
	return handle, nil
}

func closeJoyConWindowsHandle(handle uintptr) {
	if handle != 0 && handle != ^uintptr(0) {
		joyConCloseHandle.Call(handle)
	}
}

type JoyConHIDSession struct {
	device    JoyConDeviceInfo
	handle    atomic.Uintptr
	closed    atomic.Bool
	packet    atomic.Uint32
	inputOnly atomic.Bool
	writeMu   sync.Mutex
}

func OpenJoyConHIDSession(device JoyConDeviceInfo) (*JoyConHIDSession, error) {
	if !device.CanOpenAsCompatibleJoyCon() {
		return nil, fmt.Errorf("device is not an approved Joy-Con-compatible controller: vid=%04x pid=%04x", device.VendorID, device.ProductID)
	}
	if device.Path == "" {
		return nil, errors.New("Joy-Con HID device path is empty")
	}
	inputOnly := shouldOpenJoyConInputOnly(device)
	access := uintptr(joyConGenericRead | joyConGenericWrite)
	if inputOnly {
		access = joyConGenericRead
	}
	handle, err := openJoyConWindowsHandle(device.Path, access)
	if err != nil && access != joyConGenericRead {
		handle, err = openJoyConWindowsHandle(device.Path, joyConGenericRead)
		inputOnly = err == nil
	}
	if err != nil {
		return nil, err
	}
	device.InputOnly = inputOnly
	session := &JoyConHIDSession{device: device}
	session.inputOnly.Store(inputOnly)
	session.handle.Store(handle)
	return session, nil
}

func (s *JoyConHIDSession) Device() JoyConDeviceInfo {
	if s == nil {
		return JoyConDeviceInfo{}
	}
	return s.device
}

func (s *JoyConHIDSession) SetFullReportMode() error {
	if s == nil || s.closed.Load() {
		return errJoyConSessionClosed
	}
	if s.inputOnly.Load() || (s.device.OutputReportLength > 0 && s.device.OutputReportLength < joyConOutputReportLength) {
		s.inputOnly.Store(true)
		s.device.InputOnly = true
		return nil
	}
	packet := byte(s.packet.Add(1) - 1)
	report, err := buildJoyConFullReportModeCommand(packet)
	if err != nil {
		return err
	}
	if err := s.WriteReport(report); err != nil {
		// SDL supports a class of third-party Switch controllers that only expose
		// simple input reports. Continue in read-only-compatible mode instead of
		// rejecting a controller that may already be streaming report 0x3f.
		s.inputOnly.Store(true)
		s.device.InputOnly = true
		return nil
	}
	return nil
}

func (s *JoyConHIDSession) WriteReport(report []byte) error {
	if s == nil || s.closed.Load() {
		return errJoyConSessionClosed
	}
	if len(report) != joyConOutputReportLength {
		return fmt.Errorf("Joy-Con output report length is %d, expected %d", len(report), joyConOutputReportLength)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	handle := s.handle.Load()
	if handle == 0 {
		return errJoyConSessionClosed
	}
	var written uint32
	ok, _, callErr := joyConWriteFile.Call(
		handle,
		uintptr(unsafe.Pointer(&report[0])),
		uintptr(len(report)),
		uintptr(unsafe.Pointer(&written)),
		0,
	)
	if ok == 0 {
		return joyConWindowsCallError("WriteFile", callErr)
	}
	if int(written) != len(report) {
		return fmt.Errorf("Joy-Con WriteFile wrote %d of %d bytes", written, len(report))
	}
	return nil
}

func (s *JoyConHIDSession) ReadReport() ([]byte, error) {
	if s == nil || s.closed.Load() {
		return nil, errJoyConSessionClosed
	}
	handle := s.handle.Load()
	if handle == 0 {
		return nil, errJoyConSessionClosed
	}

	reportLength := int(s.device.InputReportLength)
	if reportLength <= 0 {
		reportLength = joyConInputReportLength
	}
	if reportLength > 1024 {
		return nil, fmt.Errorf("Joy-Con-compatible input report is too large: %d", reportLength)
	}
	buffer := make([]byte, reportLength)
	var read uint32
	ok, _, callErr := joyConReadFile.Call(
		handle,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
		uintptr(unsafe.Pointer(&read)),
		0,
	)
	if ok == 0 {
		if s.closed.Load() {
			return nil, errJoyConSessionClosed
		}
		return nil, joyConWindowsCallError("ReadFile", callErr)
	}
	if read == 0 {
		return nil, errors.New("Joy-Con ReadFile returned zero bytes")
	}
	return buffer[:read], nil
}

func (s *JoyConHIDSession) ReadState() (JoyConRawState, error) {
	report, err := s.ReadReport()
	if err != nil {
		return JoyConRawState{}, err
	}
	if len(report) > 0 {
		switch report[0] {
		case joyConReportFull, joyConReportSubcommandReply, joyConReportSimple:
			return parseJoyConInputReport(report)
		}
	}
	compactInputOnly := len(report) == 7 || (len(report) == 8 && report[0] == 0)
	if compactInputOnly && (s.inputOnly.Load() || s.device.ForcedCompatible || s.device.InputReportLength <= 8) {
		state, parseErr := parseJoyConInputOnlyReport(report)
		if parseErr == nil {
			s.inputOnly.Store(true)
			s.device.InputOnly = true
			return state, nil
		}
	}
	return JoyConRawState{}, fmt.Errorf("unsupported Joy-Con-compatible HID report (%d bytes): % x", len(report), reportPrefix(report, 16))
}

func reportPrefix(report []byte, limit int) []byte {
	if limit < 0 {
		limit = 0
	}
	if len(report) > limit {
		return report[:limit]
	}
	return report
}

func (s *JoyConHIDSession) Close() error {
	if s == nil || !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	// Serialize handle invalidation with WriteReport. ReadFile is interrupted by
	// CancelIoEx below, while a concurrent WriteFile must finish before the
	// underlying handle is closed.
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	handle := s.handle.Swap(0)
	if handle == 0 {
		return nil
	}
	_ = cancelJoyConPendingIO(handle)
	ok, _, callErr := joyConCloseHandle.Call(handle)
	if ok == 0 {
		return joyConWindowsCallError("CloseHandle", callErr)
	}
	return nil
}

func joyConWindowsCallError(operation string, callErr error) error {
	if callErr == nil || callErr == syscall.Errno(0) {
		return fmt.Errorf("%s failed", operation)
	}
	return fmt.Errorf("%s failed: %w", operation, callErr)
}
