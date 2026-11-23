package main

import (
	"encoding/hex"
	"testing"
)

func mustRealPawnPlusDumps(t *testing.T) ([]byte, []byte) {
	t.Helper()
	// Captured read-only from real Ray Pawn+ (RM-3360LR, VID 258A PID 1007 FW GM07)
	cfgHex := "041100000000060064060326c003070f171f7700000000000000000000ff0000" +
		"0000ffffff00e7b31e00ff00ff00ff00ffffffffff21020040ff00ff0207ff00" +
		"0000ff000000ff00ffffffff00ff00ffffffff00020603ff000000ff000000ff" +
		"ffff0000fffffffffa00ff"
	btnHex := "0412000000000600110100001102000011040000110800001110000041010000" +
		"4102000021004b0021004e00"

	cfgRaw, err := hex.DecodeString(cfgHex)
	if err != nil {
		t.Fatalf("decode cfgHex: %v", err)
	}
	btnRaw, err := hex.DecodeString(btnHex)
	if err != nil {
		t.Fatalf("decode btnHex: %v", err)
	}
	cfg := make([]byte, sinoWealthReportLen)
	btn := make([]byte, sinoWealthReportLen)
	copy(cfg, cfgRaw)
	copy(btn, btnRaw)
	return cfg, btn
}

func TestDecodeRealRayPawnPlusDump(t *testing.T) {
	cfg, btn := mustRealPawnPlusDumps(t)
	st, err := decodeSinoWealthState(0x258A, 0x1007, "GM07", "Ray Pawn+", 1, cfg, btn)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if st.SensorType != sinoWealthSensorPMW3360 || st.MaxDPI != 12000 {
		t.Fatalf("unexpected sensor: type=0x%02x maxDPI=%d", st.SensorType, st.MaxDPI)
	}
	if st.ReportRateHz != 500 {
		t.Fatalf("reportRate=%d want 500", st.ReportRateHz)
	}
	if st.ActiveDPIIndex != 1 || st.ActiveDPISlot != 2 {
		t.Fatalf("activeDPI idx=%d slot=%d want idx=1 slot=2", st.ActiveDPIIndex, st.ActiveDPISlot)
	}
	wantDPIs := []int{400, 800, 1600, 2400, 3200, 12000}
	for i, want := range wantDPIs {
		if !st.DPISlots[i].Enabled || st.DPISlots[i].DPI != want {
			t.Fatalf("dpi[%d]=%+v want enabled=%v dpi=%d", i, st.DPISlots[i], true, want)
		}
	}
	if st.DPISlots[6].Enabled || st.DPISlots[7].Enabled {
		t.Fatalf("slots 6,7 should be disabled")
	}
	if st.ButtonCount != 9 {
		t.Fatalf("buttonCount=%d want 9", st.ButtonCount)
	}
	if st.Buttons[7].Kind != "key" || st.Buttons[7].HIDUsage != 0x4B || st.Buttons[7].Display != "PageUp" {
		t.Fatalf("button 7=%+v want PageUp", st.Buttons[7])
	}
	if st.Buttons[8].Kind != "key" || st.Buttons[8].HIDUsage != 0x4E || st.Buttons[8].Display != "PageDown" {
		t.Fatalf("button 8=%+v want PageDown", st.Buttons[8])
	}
}

func TestBuildSinoWealthButtonsWritePreservesHeaderAndAppliesPreset(t *testing.T) {
	_, origBtn := mustRealPawnPlusDumps(t)
	preset, err := sinoWealthPresetButtons("mbm-all-f13-f16")
	if err != nil {
		t.Fatalf("preset: %v", err)
	}
	out, err := buildSinoWealthButtonsWriteReport(origBtn, preset)
	if err != nil {
		t.Fatalf("build buttons: %v", err)
	}
	if out[0x03] != sinoWealthButtonsWrite {
		t.Fatalf("config_write=0x%02x want 0x%02x", out[0x03], sinoWealthButtonsWrite)
	}
	if out[0x06] != 0x06 {
		t.Fatalf("header byte 0x06 altered: 0x%02x", out[0x06])
	}
	// Verify Button 5..8 -> F15, F16, F13, F14
	wantUsages := map[int]uint8{5: 0x6A, 6: 0x6B, 7: 0x68, 8: 0x69}
	for idx, want := range wantUsages {
		off := 0x08 + idx*4
		if out[off] != 0x21 || out[off+1] != 0x00 || out[off+2] != want || out[off+3] != 0x00 {
			t.Fatalf("button %d bytes=%02x want 21 00 %02x 00", idx, out[off:off+4], want)
		}
	}
}

func TestBuildSinoWealthConfigWritePreservesTailAndHighNibble(t *testing.T) {
	origCfg, _ := mustRealPawnPlusDumps(t)
	out, err := buildSinoWealthConfigWriteReport(origCfg, SinoWealthConfigPatch{
		ReportRateHz:   1000,
		ActiveDPIIndex: 2, // 1600 DPI (3rd enabled)
		RGBEffectMode:  2, // Static while keeping upper nibble 0x20 -> 0x22
	})
	if err != nil {
		t.Fatalf("build config: %v", err)
	}
	if out[0x03] != sinoWealthConfigWrite {
		t.Fatalf("config_write=0x%02x want 0x%02x", out[0x03], sinoWealthConfigWrite)
	}
	if out[0x09] != sinoWealthSensorPMW3360 {
		t.Fatalf("sensor altered: 0x%02x", out[0x09])
	}
	if (out[0x0A] & 0x0F) != 4 {
		t.Fatalf("report rate nibble=%d want 4 (1000Hz)", out[0x0A]&0x0F)
	}
	if out[0x0B] != 0x36 {
		t.Fatalf("active/count byte=0x%02x want 0x36", out[0x0B])
	}
	if out[0x35] != 0x22 {
		t.Fatalf("rgb effect byte=0x%02x want 0x22 (preserved high nibble 0x20)", out[0x35])
	}
	// Ensure bytes 0x36..519 are byte-for-byte identical
	for i := 0x36; i < sinoWealthReportLen; i++ {
		if out[i] != origCfg[i] {
			t.Fatalf("unexpected mutation at offset 0x%02x: got 0x%02x want 0x%02x", i, out[i], origCfg[i])
		}
	}
}
