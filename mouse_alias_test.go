package main

import "testing"

func TestMouseAliasMappingForRayPawnPlus(t *testing.T) {
	defer updateMouseAliasesFromSinoWealth(nil)

	btns, err := sinoWealthPresetButtons("mbm-all-f13-f16")
	if err != nil {
		t.Fatalf("preset: %v", err)
	}
	st := &SinoWealthConfigState{
		ProductName: "Ray Pawn+ (RM-3360LR)",
		Buttons:     btns,
	}
	updateMouseAliasesFromSinoWealth(st)

	checks := map[uint32]string{
		0x7C: "右サイド1",   // F13
		0x81: "右サイド2",   // F18
		0x82: "DPI+ボタン", // F19
		0x83: "DPI-ボタン", // F20
	}
	for vk, want := range checks {
		got, ok := mouseAliasNameForVK(vk)
		if !ok || got != want {
			t.Fatalf("vk 0x%X -> (%q, %v), want (%q, true)", vk, got, ok, want)
		}
		parsedVK, ok := parseMouseAliasVK(want)
		if !ok || parsedVK != vk {
			t.Fatalf("parseMouseAliasVK(%q) -> (0x%X, %v), want (0x%X, true)", want, parsedVK, ok, vk)
		}
	}

	// Unmapped F17 must remain unaliased
	if name, ok := mouseAliasNameForVK(0x80); ok {
		t.Fatalf("unexpected alias for F17: %q", name)
	}

	logical, dev := currentMouseLogicalButtons()
	if dev != "Ray Pawn+ (RM-3360LR)" || len(logical) != 4 {
		t.Fatalf("logical buttons=%+v dev=%q", logical, dev)
	}
}
