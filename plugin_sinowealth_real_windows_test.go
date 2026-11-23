//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRealHardwareReadOnlySmoke verifies the Go HID backend and running-app picker
// against the live Windows machine when MBM_REAL_HARDWARE_TEST=1 is set.
func TestRealHardwareReadOnlySmoke(t *testing.T) {
	if os.Getenv("MBM_REAL_HARDWARE_TEST") != "1" {
		t.Skip("set MBM_REAL_HARDWARE_TEST=1 to run live hardware read-only smoke test")
	}

	tmpDir := t.TempDir()
	fakeApp := &App{configPath: filepath.Join(tmpDir, "config.json")}
	plugin := newSinoWealthPlugin()
	defer plugin.Shutdown()

	desc := plugin.Descriptor(fakeApp)
	t.Logf("Plugin descriptor: detected=%v summary=%q", desc.Detected, desc.Summary)
	if !desc.Detected {
		t.Fatal("expected SinoWealth 258A:1007 mouse to be detected")
	}

	st, backupPath, err := plugin.ReadDevice(fakeApp)
	if err != nil {
		t.Fatalf("ReadDevice failed: %v", err)
	}
	t.Logf("ReadDevice OK: %s FW=%s Sensor=%s Rate=%dHz ActiveDPI=#%d(%d) Buttons=%d Backup=%s",
		st.ProductName, st.Firmware, st.SensorName, st.ReportRateHz,
		st.ActiveDPIIndex+1, st.DPISlots[st.ActiveDPIIndex].DPI, st.ButtonCount, backupPath)
	if st.Firmware != "GM07" || st.SensorType != sinoWealthSensorPMW3360 {
		t.Fatalf("unexpected device identity: fw=%q sensor=0x%02x", st.Firmware, st.SensorType)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected auto initial backup at %s: %v", backupPath, err)
	}

	apps := listRunningApps(0)
	t.Logf("listRunningApps returned %d visible apps (selfElevated=%v)", len(apps), isCurrentProcessElevated())
	if len(apps) == 0 {
		t.Fatal("expected at least one visible running app")
	}
	for i := 0; i < len(apps) && i < 5; i++ {
		t.Logf("  app[%d]: %s (elevated=%v) title=%q", i, apps[i].ProcessName, apps[i].Elevated, apps[i].Title)
	}
}
