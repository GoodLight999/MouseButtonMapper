package main

import (
	"strings"
	"testing"
)

func TestNormalizeJoyConProfileConfig(t *testing.T) {
	config := normalizeJoyConProfileConfig(JoyConProfileConfig{
		PreferredDevice: "  serial:ABC  ",
		Reconnect: JoyConReconnectConfig{
			IntervalMs: 1,
		},
	})
	if config.PreferredDevice != "serial:ABC" {
		t.Fatalf("PreferredDevice=%q", config.PreferredDevice)
	}
	if config.Reconnect.IntervalMs != minJoyConReconnectMs {
		t.Fatalf("IntervalMs=%d", config.Reconnect.IntervalMs)
	}
	if config.Stick.DeadZone != defaultJoyConStickConfig().DeadZone {
		t.Fatalf("Stick=%+v", config.Stick)
	}

	config.Reconnect.IntervalMs = maxJoyConReconnectMs + 1
	config = normalizeJoyConProfileConfig(config)
	if config.Reconnect.IntervalMs != maxJoyConReconnectMs {
		t.Fatalf("clamped IntervalMs=%d", config.Reconnect.IntervalMs)
	}
}

func TestJoyConDeviceStableIDPrefersSerial(t *testing.T) {
	device := JoyConDeviceInfo{
		Fingerprint: "0123456789abcdef",
		VendorID:    joyConNintendoVendorID,
		ProductID:   joyConLeftProductID,
		Serial:      "ABC123",
	}
	if got, want := device.StableID(), "serial:abc123"; got != want {
		t.Fatalf("StableID=%q want=%q", got, want)
	}
	device.Serial = ""
	if got, want := device.StableID(), "path:0123456789abcdef"; got != want {
		t.Fatalf("StableID=%q want=%q", got, want)
	}
}

func TestFingerprintJoyConDevicePathIsStableAndNonRevealing(t *testing.T) {
	path := `\\?\hid#vid_057e&pid_2006#private-user-path`
	first := fingerprintJoyConDevicePath(path)
	second := fingerprintJoyConDevicePath(strings.ToUpper(path))
	if first != second {
		t.Fatalf("fingerprint differs: %q != %q", first, second)
	}
	if len(first) != 16 {
		t.Fatalf("fingerprint length=%d", len(first))
	}
	if strings.Contains(first, "private") {
		t.Fatalf("fingerprint leaked path: %q", first)
	}
}

func TestChooseJoyConDevice(t *testing.T) {
	devices := []JoyConDeviceInfo{
		{VendorID: 0x1234, ProductID: 0x5678, Product: "Other HID"},
		{VendorID: joyConNintendoVendorID, ProductID: joyConLeftProductID, Serial: "left-a"},
		{VendorID: joyConNintendoVendorID, ProductID: joyConLeftProductID, Serial: "left-b"},
	}
	selected, ok := chooseJoyConDevice(devices, JoyConProfileConfig{PreferredDevice: "serial:left-b"})
	if !ok || selected.Serial != "left-b" {
		t.Fatalf("preferred selection=%+v ok=%v", selected, ok)
	}
	selected, ok = chooseJoyConDevice(devices, JoyConProfileConfig{PreferredDevice: "missing"})
	if !ok || selected.Serial != "left-a" {
		t.Fatalf("fallback selection=%+v ok=%v", selected, ok)
	}
	if _, ok := chooseJoyConDevice(devices[:1], JoyConProfileConfig{}); ok {
		t.Fatal("non-Joy-Con device was selected")
	}
}

func TestBuildJoyConFullReportModeCommand(t *testing.T) {
	report, err := buildJoyConFullReportModeCommand(0x2f)
	if err != nil {
		t.Fatalf("buildJoyConFullReportModeCommand: %v", err)
	}
	if len(report) != joyConOutputReportLength {
		t.Fatalf("len=%d", len(report))
	}
	if report[0] != 0x01 || report[1] != 0x0f {
		t.Fatalf("header=%#x/%#x", report[0], report[1])
	}
	wantRumble := []byte{0x00, 0x01, 0x40, 0x40, 0x00, 0x01, 0x40, 0x40}
	for i, want := range wantRumble {
		if report[2+i] != want {
			t.Fatalf("rumble[%d]=%#x want=%#x", i, report[2+i], want)
		}
	}
	if report[10] != 0x03 || report[11] != joyConReportFull {
		t.Fatalf("subcommand=%#x data=%#x", report[10], report[11])
	}
}

func TestBuildJoyConSubcommandRejectsOversizedPayload(t *testing.T) {
	payload := make([]byte, joyConOutputReportLength-10)
	if _, err := buildJoyConSubcommandReport(0, 0x03, payload); err == nil {
		t.Fatal("oversized payload unexpectedly succeeded")
	}
}

func TestJoyConCloneClassification(t *testing.T) {
	if !(JoyConDeviceInfo{VendorID: 0x1234, ProductID: 0x5678, Product: "Joy-Con (L)"}).IsLeftJoyCon() {
		t.Fatal("explicit clone product name was not accepted")
	}
	proLike := JoyConDeviceInfo{VendorID: joyConNintendoVendorID, ProductID: joyConProProductID, Product: "Pro Controller"}
	if proLike.IsLeftJoyCon() || !proLike.MightBeLeftJoyCon() {
		t.Fatalf("Switch Pro PID candidate classification is wrong: %+v", proLike)
	}
	proLike.ControllerType = joyConTypeLeft
	if !proLike.IsLeftJoyCon() {
		t.Fatal("probed left Joy-Con type was not accepted")
	}
}

func TestParseJoyConControllerTypeReply(t *testing.T) {
	report := make([]byte, 18)
	report[0] = joyConReportSubcommandReply
	report[13] = 0x80
	report[14] = 0x02
	report[17] = joyConTypeLeft
	got, ok := parseJoyConControllerTypeReply(report)
	if !ok || got != joyConTypeLeft {
		t.Fatalf("controller type=(%d,%v)", got, ok)
	}
	report[13] = 0
	if _, ok := parseJoyConControllerTypeReply(report); ok {
		t.Fatal("unacknowledged reply was accepted")
	}
}

func TestChoosePreferredGenericGamepadAsForcedCompatible(t *testing.T) {
	device := JoyConDeviceInfo{
		Fingerprint: "clone-1",
		VendorID:    0x20d6, ProductID: 0xa711,
		Product:   "Wireless Controller",
		UsagePage: hidUsagePageGenericDesktop,
		Usage:     hidUsageGamePad,
	}
	selected, ok := chooseJoyConDevice([]JoyConDeviceInfo{device}, JoyConProfileConfig{CompatibleDevice: manualJoyConDeviceConfig(device)})
	if !ok || !selected.ForcedCompatible {
		t.Fatalf("manual compatible selection=%+v ok=%v", selected, ok)
	}
	if !selected.CanOpenAsCompatibleJoyCon() {
		t.Fatal("manual compatible device cannot be opened")
	}
	if _, ok := chooseJoyConDevice([]JoyConDeviceInfo{device}, JoyConProfileConfig{}); ok {
		t.Fatal("unknown generic gamepad was auto-selected without opt-in")
	}
}

func TestGameControllerCollectionClassification(t *testing.T) {
	gamepad := JoyConDeviceInfo{UsagePage: hidUsagePageGenericDesktop, Usage: hidUsageGamePad}
	joystick := JoyConDeviceInfo{UsagePage: hidUsagePageGenericDesktop, Usage: hidUsageJoystick}
	keyboard := JoyConDeviceInfo{UsagePage: hidUsagePageGenericDesktop, Usage: 0x06}
	if !gamepad.IsGameControllerCollection() || !joystick.IsGameControllerCollection() {
		t.Fatal("game controller HID collection was rejected")
	}
	if keyboard.IsGameControllerCollection() {
		t.Fatal("keyboard HID collection was accepted as a game controller")
	}
}

func TestShouldOpenJoyConInputOnlyDistinguishesUnknownCapsFromNoOutput(t *testing.T) {
	knownWithoutCaps := JoyConDeviceInfo{VendorID: joyConNintendoVendorID, ProductID: joyConLeftProductID}
	if shouldOpenJoyConInputOnly(knownWithoutCaps) {
		t.Fatal("known Joy-Con with unavailable HID caps was incorrectly forced read-only")
	}
	forcedWritable := JoyConDeviceInfo{ForcedCompatible: true, InputReportLength: 64, OutputReportLength: 64}
	if shouldOpenJoyConInputOnly(forcedWritable) {
		t.Fatal("manually registered writable HID candidate must try Nintendo R/W setup before read-only fallback")
	}
	compact := JoyConDeviceInfo{InputReportLength: 8, OutputReportLength: 8}
	if !shouldOpenJoyConInputOnly(compact) {
		t.Fatal("compact input-only controller was not detected")
	}
}

func TestManualJoyConRegistrationSelectsVendorSpecificInterface(t *testing.T) {
	device := JoyConDeviceInfo{
		Fingerprint: "vendor-specific", VendorID: 0x1234, ProductID: 0xabcd,
		Product: "Wireless Gamepad", UsagePage: 0xff00, Usage: 0x01,
		InputReportLength: 64, OutputReportLength: 64,
	}
	config := JoyConProfileConfig{CompatibleDevice: manualJoyConDeviceConfig(device)}
	selected, ok := chooseJoyConDevice([]JoyConDeviceInfo{device}, config)
	if !ok || !selected.ForcedCompatible || selected.ControllerType != joyConTypeLeft {
		t.Fatalf("vendor-specific manual selection=%+v ok=%v", selected, ok)
	}
}

func TestManualJoyConRegistrationPrefersExactInterfaceFingerprint(t *testing.T) {
	first := JoyConDeviceInfo{Fingerprint: "if-a", VendorID: 0x1234, ProductID: 0xabcd, Serial: "same"}
	second := JoyConDeviceInfo{Fingerprint: "if-b", VendorID: 0x1234, ProductID: 0xabcd, Serial: "same"}
	config := JoyConProfileConfig{CompatibleDevice: manualJoyConDeviceConfig(second)}
	selected, ok := chooseJoyConDevice([]JoyConDeviceInfo{first, second}, config)
	if !ok || selected.Fingerprint != "if-b" {
		t.Fatalf("exact interface selection=%+v ok=%v", selected, ok)
	}
}

func TestManualJoyConRegistrationFallsBackToSerialAfterPathChanges(t *testing.T) {
	registered := JoyConDeviceInfo{Fingerprint: "old-path", VendorID: 0x1234, ProductID: 0xabcd, Serial: "same"}
	reconnected := JoyConDeviceInfo{Fingerprint: "new-path", VendorID: 0x1234, ProductID: 0xabcd, Serial: "same"}
	config := JoyConProfileConfig{CompatibleDevice: manualJoyConDeviceConfig(registered)}
	selected, ok := chooseJoyConDevice([]JoyConDeviceInfo{reconnected}, config)
	if !ok || selected.Fingerprint != "new-path" {
		t.Fatalf("serial fallback selection=%+v ok=%v", selected, ok)
	}
}

func TestParseJoyConVIDPIDFromPath(t *testing.T) {
	vid, pid := parseJoyConVIDPIDFromPath(`\\?\hid#vid_20D6&pid_A711&mi_00#example`)
	if vid != 0x20d6 || pid != 0xa711 {
		t.Fatalf("vid/pid=%04x/%04x", vid, pid)
	}
}

func TestUnknownProductNamedJoyConStillRequiresManualRegistration(t *testing.T) {
	device := JoyConDeviceInfo{Fingerprint: "clone", VendorID: 0x1234, ProductID: 0xabcd, Product: "Joy-Con (L)"}
	if !device.IsLeftJoyCon() {
		t.Fatal("product classification should remain available for diagnostics")
	}
	if _, ok := chooseJoyConDevice([]JoyConDeviceInfo{device}, JoyConProfileConfig{}); ok {
		t.Fatal("unknown VID/PID clone was auto-opened without explicit registration")
	}
}

func TestManualJoyConSerialFallbackRejectsAmbiguousInterfaces(t *testing.T) {
	registered := JoyConDeviceInfo{Fingerprint: "old", VendorID: 0x1234, ProductID: 0xabcd, Serial: "same"}
	first := JoyConDeviceInfo{Fingerprint: "new-a", VendorID: 0x1234, ProductID: 0xabcd, Serial: "same"}
	second := JoyConDeviceInfo{Fingerprint: "new-b", VendorID: 0x1234, ProductID: 0xabcd, Serial: "same"}
	config := JoyConProfileConfig{CompatibleDevice: manualJoyConDeviceConfig(registered)}
	if _, ok := chooseJoyConDevice([]JoyConDeviceInfo{first, second}, config); ok {
		t.Fatal("ambiguous serial fallback selected an arbitrary HID interface")
	}
}
