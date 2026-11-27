//go:build windows

package main

import "testing"

func TestPlanSinoWealthDirectWriteHybridAndStandalone(t *testing.T) {
	curBtns, err := sinoWealthPresetButtons("mbm-all-safe")
	if err != nil {
		t.Fatalf("preset: %v", err)
	}
	prof := Profile{
		Id:   "default",
		Name: "既定",
		Rules: []Rule{
			{Enabled: true, Input: []Item{{Kind: "Mouse", Code: "X2"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: "17"}, {Kind: "Key", Code: "67"}}}, // Ctrl+C
			{Enabled: true, Input: []Item{{Kind: "Mouse", Code: "X1"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: "17"}, {Kind: "Key", Code: "86"}}}, // Ctrl+V
			{Enabled: true, Input: []Item{{Kind: "Key", Code: "124"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: "17"}, {Kind: "Key", Code: "65"}}},  // F13 -> Ctrl+A
			{Enabled: true, Input: []Item{{Kind: "Key", Code: "129"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: "17"}, {Kind: "Key", Code: "88"}}},  // F18 -> Ctrl+X
			{Enabled: true, Input: []Item{{Kind: "Key", Code: "130"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: "91"}, {Kind: "Key", Code: "86"}}},  // F19 -> Win+V
			{Enabled: true, Input: []Item{{Kind: "Key", Code: "131"}}, Mode: "Hold", Output: []Item{{Kind: "Key", Code: "91"}, {Kind: "Key", Code: "72"}}}, // F20 -> Win+H
			{Enabled: true, Input: []Item{{Kind: "Mouse", Code: "X2"}, {Kind: "Mouse", Code: "WheelUp"}}, Mode: "Tap", Output: []Item{{Kind: "Key", Code: "17"}, {Kind: "Key", Code: "91"}, {Kind: "Key", Code: "37"}}},
		},
	}

	// Hybrid (ext4): keeps X1/X2 as mouse X1/X2, converts slots 5..8 directly to hardware shortcuts.
	btns, written, idxs := planSinoWealthDirectWriteFromProfile(prof, curBtns, "ext4")
	if len(written) != 4 || len(idxs) != 4 {
		t.Fatalf("hybrid written=%v idxs=%v want 4", written, idxs)
	}
	if btns[3].Kind != "mouse" || btns[3].Value != "x1" || btns[4].Kind != "mouse" || btns[4].Value != "x2" {
		t.Fatalf("hybrid must preserve X1/X2, got slot3=%+v slot4=%+v", btns[3], btns[4])
	}
	wantRaw := map[int][4]byte{
		5: {0x21, 0x08, 0x19, 0x00}, // Win+V
		6: {0x21, 0x08, 0x0B, 0x00}, // Win+H
		7: {0x21, 0x01, 0x04, 0x00}, // Ctrl+A
		8: {0x21, 0x01, 0x1B, 0x00}, // Ctrl+X
	}
	for slot, want := range wantRaw {
		got, err := encodeSinoWealthButton(btns[slot])
		if err != nil || got != want {
			t.Fatalf("slot %d encoded=%02x err=%v want=%02x", slot, got, err, want)
		}
	}

	// Standalone (all): also converts X1/X2 to Ctrl+V / Ctrl+C.
	btnsAll, writtenAll, _ := planSinoWealthDirectWriteFromProfile(prof, curBtns, "all")
	if len(writtenAll) != 6 {
		t.Fatalf("standalone written=%v want 6", writtenAll)
	}
	gotX1, _ := encodeSinoWealthButton(btnsAll[3])
	gotX2, _ := encodeSinoWealthButton(btnsAll[4])
	if gotX1 != [4]byte{0x21, 0x01, 0x19, 0x00} || gotX2 != [4]byte{0x21, 0x01, 0x06, 0x00} {
		t.Fatalf("standalone X1=%02x X2=%02x want Ctrl+V / Ctrl+C", gotX1, gotX2)
	}
}
