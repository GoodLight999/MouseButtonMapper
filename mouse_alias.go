package main

import (
	"strconv"
	"strings"
	"sync"
)

type MouseLogicalButton struct {
	Name   string `json:"name"`
	VK     uint32 `json:"vk"`
	Key    string `json:"key"`
	Source string `json:"source"`
}

type mouseAliasRegistry struct {
	mu           sync.RWMutex
	vkToName     map[uint32]string
	nameToVK     map[string]uint32
	buttons      []MouseLogicalButton
	deviceName   string
	hasConnected bool
}

var mouseAliases = &mouseAliasRegistry{
	vkToName: map[uint32]string{},
	nameToVK: map[string]uint32{},
}

func hidFKeyUsageToVK(usage uint8) (uint32, string, bool) {
	// USB HID Keyboard Usage 0x68..0x73 -> Windows VK_F13 (0x7C=124) .. VK_F24 (0x87=135)
	if usage < 0x68 || usage > 0x73 {
		return 0, "", false
	}
	offset := uint32(usage - 0x68)
	vk := uint32(0x7C) + offset
	name := "F" + strconv.Itoa(13+int(offset))
	return vk, name, true
}

func sinoWealthLogicalNameForSlot(index int) (canonical string, extraAliases []string) {
	switch index {
	case 5:
		return "DPI+ボタン", []string{"dpi+", "dpiup", "dpi上", "dpi+button", "上部ボタン1"}
	case 6:
		return "DPI-ボタン", []string{"dpi-", "dpidown", "dpi下", "dpi-button", "上部ボタン2"}
	case 7:
		return "右サイド1", []string{"右サイド上", "右サイドボタン1", "rside1", "rightside1"}
	case 8:
		return "右サイド2", []string{"右サイド下", "右サイドボタン2", "rside2", "rightside2"}
	default:
		return "", nil
	}
}

func normalizeAliasLookupKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "　", "")
	s = strings.ReplaceAll(s, "（", "(")
	s = strings.ReplaceAll(s, "）", ")")
	return s
}

// updateMouseAliasesFromSinoWealth refreshes logical button names (右サイド1/2, DPI±ボタン)
// whenever a recognized SinoWealth mouse is read or updated.
func updateMouseAliasesFromSinoWealth(st *SinoWealthConfigState) {
	mouseAliases.mu.Lock()
	defer mouseAliases.mu.Unlock()

	if st == nil {
		mouseAliases.vkToName = map[uint32]string{}
		mouseAliases.nameToVK = map[string]uint32{}
		mouseAliases.buttons = nil
		mouseAliases.deviceName = ""
		mouseAliases.hasConnected = false
		return
	}

	vkToName := map[uint32]string{}
	nameToVK := map[string]uint32{}
	var buttons []MouseLogicalButton

	// Present right-side buttons first, then DPI buttons, for intuitive UI ordering.
	order := []int{7, 8, 5, 6}
	for _, idx := range order {
		if idx < 0 || idx >= len(st.Buttons) {
			continue
		}
		btn := st.Buttons[idx]
		if btn.Kind != "key" || btn.Modifiers != 0 {
			continue
		}
		vk, keyName, ok := hidFKeyUsageToVK(btn.HIDUsage)
		if !ok {
			continue
		}
		canonical, extras := sinoWealthLogicalNameForSlot(idx)
		if canonical == "" {
			continue
		}
		vkToName[vk] = canonical
		nameToVK[normalizeAliasLookupKey(canonical)] = vk
		for _, alias := range extras {
			nameToVK[normalizeAliasLookupKey(alias)] = vk
		}
		buttons = append(buttons, MouseLogicalButton{
			Name:   canonical,
			VK:     vk,
			Key:    keyName,
			Source: st.ProductName,
		})
	}

	mouseAliases.vkToName = vkToName
	mouseAliases.nameToVK = nameToVK
	mouseAliases.buttons = buttons
	mouseAliases.deviceName = st.ProductName
	mouseAliases.hasConnected = true
}

func mouseAliasNameForVK(vk uint32) (string, bool) {
	mouseAliases.mu.RLock()
	defer mouseAliases.mu.RUnlock()
	name, ok := mouseAliases.vkToName[vk]
	return name, ok
}

func parseMouseAliasVK(tok string) (uint32, bool) {
	key := normalizeAliasLookupKey(tok)
	if key == "" {
		return 0, false
	}
	mouseAliases.mu.RLock()
	vk, ok := mouseAliases.nameToVK[key]
	mouseAliases.mu.RUnlock()
	if ok {
		return vk, true
	}
	// Default fallback aliases so saved/typed logical names work even if the mouse is unplugged.
	switch key {
	case "右サイド1", "右サイド上", "右サイドボタン1", "rside1", "rightside1":
		return 0x7C, true // F13
	case "右サイド2", "右サイド下", "右サイドボタン2", "rside2", "rightside2":
		return 0x81, true // F18
	case "dpi+ボタン", "dpi+", "dpiup", "dpi上", "上部ボタン1":
		return 0x82, true // F19
	case "dpi-ボタン", "dpi-", "dpidown", "dpi下", "上部ボタン2":
		return 0x83, true // F20
	}
	return 0, false
}

func currentMouseLogicalButtons() ([]MouseLogicalButton, string) {
	mouseAliases.mu.RLock()
	defer mouseAliases.mu.RUnlock()
	out := make([]MouseLogicalButton, len(mouseAliases.buttons))
	copy(out, mouseAliases.buttons)
	return out, mouseAliases.deviceName
}
