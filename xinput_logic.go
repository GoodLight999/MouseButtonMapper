package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	xInputMaxUsers       = 4
	xInputTriggerPress   = uint8(30)
	xInputTriggerRelease = uint8(20)
	xInputStickPress     = 0.55
	xInputStickRelease   = 0.42
)

const (
	xInputButtonDPadUp    = "DPadUp"
	xInputButtonDPadDown  = "DPadDown"
	xInputButtonDPadLeft  = "DPadLeft"
	xInputButtonDPadRight = "DPadRight"
	xInputButtonStart     = "Start"
	xInputButtonBack      = "Back"
	xInputButtonLStick    = "LStickPress"
	xInputButtonRStick    = "RStickPress"
	xInputButtonLB        = "LB"
	xInputButtonRB        = "RB"
	xInputButtonA         = "A"
	xInputButtonB         = "B"
	xInputButtonX         = "X"
	xInputButtonY         = "Y"
	xInputButtonLT        = "LT"
	xInputButtonRT        = "RT"
	xInputLStickUp        = "LStickUp"
	xInputLStickDown      = "LStickDown"
	xInputLStickLeft      = "LStickLeft"
	xInputLStickRight     = "LStickRight"
	xInputRStickUp        = "RStickUp"
	xInputRStickDown      = "RStickDown"
	xInputRStickLeft      = "RStickLeft"
	xInputRStickRight     = "RStickRight"
)

var xInputKnownControls = []string{
	xInputButtonDPadUp, xInputButtonDPadDown, xInputButtonDPadLeft, xInputButtonDPadRight,
	xInputButtonStart, xInputButtonBack, xInputButtonLStick, xInputButtonRStick,
	xInputButtonLB, xInputButtonRB, xInputButtonA, xInputButtonB, xInputButtonX, xInputButtonY,
	xInputButtonLT, xInputButtonRT,
	xInputLStickUp, xInputLStickDown, xInputLStickLeft, xInputLStickRight,
	xInputRStickUp, xInputRStickDown, xInputRStickLeft, xInputRStickRight,
}

type xInputGamepadState struct {
	Buttons      uint16
	LeftTrigger  uint8
	RightTrigger uint8
	LeftX        int16
	LeftY        int16
	RightX       int16
	RightY       int16
}

type xInputStateTracker struct {
	slot int
	down map[string]bool
}

func newXInputStateTracker(slot int) *xInputStateTracker {
	return &xInputStateTracker{slot: slot, down: make(map[string]bool, len(xInputKnownControls))}
}

func normalizeXInputCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	slot := 1
	control := code
	lower := strings.ToLower(strings.ReplaceAll(code, " ", ""))
	if strings.HasPrefix(lower, "pad") {
		if sep := strings.IndexAny(lower, ":/"); sep > 3 {
			if n, err := strconv.Atoi(lower[3:sep]); err == nil && n >= 1 && n <= xInputMaxUsers {
				slot = n
				control = code[sep+1:]
			}
		}
	} else if strings.HasPrefix(lower, "p") {
		if sep := strings.IndexAny(lower, ":/"); sep > 1 {
			if n, err := strconv.Atoi(lower[1:sep]); err == nil && n >= 1 && n <= xInputMaxUsers {
				slot = n
				control = code[sep+1:]
			}
		}
	}

	clean := strings.ToLower(strings.NewReplacer(" ", "", "_", "", "-", "").Replace(control))
	canonical := ""
	switch clean {
	case "dpadup", "up", "十字上":
		canonical = xInputButtonDPadUp
	case "dpaddown", "down", "十字下":
		canonical = xInputButtonDPadDown
	case "dpadleft", "left", "十字左":
		canonical = xInputButtonDPadLeft
	case "dpadright", "right", "十字右":
		canonical = xInputButtonDPadRight
	case "start", "menu":
		canonical = xInputButtonStart
	case "back", "view", "select":
		canonical = xInputButtonBack
	case "lstickpress", "leftthumb", "l3":
		canonical = xInputButtonLStick
	case "rstickpress", "rightthumb", "r3":
		canonical = xInputButtonRStick
	case "lb", "leftshoulder":
		canonical = xInputButtonLB
	case "rb", "rightshoulder":
		canonical = xInputButtonRB
	case "a":
		canonical = xInputButtonA
	case "b":
		canonical = xInputButtonB
	case "x":
		canonical = xInputButtonX
	case "y":
		canonical = xInputButtonY
	case "lt", "lefttrigger":
		canonical = xInputButtonLT
	case "rt", "righttrigger":
		canonical = xInputButtonRT
	case "lstickup", "leftstickup", "左スティック上":
		canonical = xInputLStickUp
	case "lstickdown", "leftstickdown", "左スティック下":
		canonical = xInputLStickDown
	case "lstickleft", "leftstickleft", "左スティック左":
		canonical = xInputLStickLeft
	case "lstickright", "leftstickright", "左スティック右":
		canonical = xInputLStickRight
	case "rstickup", "rightstickup", "右スティック上":
		canonical = xInputRStickUp
	case "rstickdown", "rightstickdown", "右スティック下":
		canonical = xInputRStickDown
	case "rstickleft", "rightstickleft", "右スティック左":
		canonical = xInputRStickLeft
	case "rstickright", "rightstickright", "右スティック右":
		canonical = xInputRStickRight
	default:
		return strings.TrimSpace(code)
	}
	return fmt.Sprintf("P%d:%s", slot, canonical)
}

func splitXInputCode(code string) (slot int, control string, ok bool) {
	normalized := normalizeXInputCode(code)
	sep := strings.IndexByte(normalized, ':')
	if sep < 2 || normalized[0] != 'P' {
		return 0, "", false
	}
	n, err := strconv.Atoi(normalized[1:sep])
	if err != nil || n < 1 || n > xInputMaxUsers {
		return 0, "", false
	}
	control = normalized[sep+1:]
	for _, known := range xInputKnownControls {
		if control == known {
			return n, control, true
		}
	}
	return 0, "", false
}

func isKnownXInputCode(code string) bool {
	_, _, ok := splitXInputCode(code)
	return ok
}

func xInputCodeText(code string) string {
	slot, control, ok := splitXInputCode(code)
	if !ok {
		return strings.TrimSpace(code)
	}
	text := control
	switch control {
	case xInputButtonDPadUp:
		text = "十字上"
	case xInputButtonDPadDown:
		text = "十字下"
	case xInputButtonDPadLeft:
		text = "十字左"
	case xInputButtonDPadRight:
		text = "十字右"
	case xInputButtonLStick:
		text = "左スティック押込み"
	case xInputButtonRStick:
		text = "右スティック押込み"
	case xInputLStickUp:
		text = "左スティック上"
	case xInputLStickDown:
		text = "左スティック下"
	case xInputLStickLeft:
		text = "左スティック左"
	case xInputLStickRight:
		text = "左スティック右"
	case xInputRStickUp:
		text = "右スティック上"
	case xInputRStickDown:
		text = "右スティック下"
	case xInputRStickLeft:
		text = "右スティック左"
	case xInputRStickRight:
		text = "右スティック右"
	}
	return fmt.Sprintf("P%d:%s", slot, text)
}

func normalizeXInputAxis(value int16) float64 {
	if value < 0 {
		return float64(value) / 32768.0
	}
	return float64(value) / 32767.0
}

func updateXInputAxisDirections(next map[string]bool, previous map[string]bool, negativeCode, positiveCode string, value float64) {
	if previous[negativeCode] {
		if value <= -xInputStickRelease {
			next[negativeCode] = true
		}
	} else if value <= -xInputStickPress {
		next[negativeCode] = true
	}
	if previous[positiveCode] {
		if value >= xInputStickRelease {
			next[positiveCode] = true
		}
	} else if value >= xInputStickPress {
		next[positiveCode] = true
	}
}

func (t *xInputStateTracker) Update(state xInputGamepadState, at time.Time) []InputEvent {
	next := make(map[string]bool, len(xInputKnownControls))
	masks := []struct {
		mask uint16
		code string
	}{
		{0x0001, xInputButtonDPadUp}, {0x0002, xInputButtonDPadDown},
		{0x0004, xInputButtonDPadLeft}, {0x0008, xInputButtonDPadRight},
		{0x0010, xInputButtonStart}, {0x0020, xInputButtonBack},
		{0x0040, xInputButtonLStick}, {0x0080, xInputButtonRStick},
		{0x0100, xInputButtonLB}, {0x0200, xInputButtonRB},
		{0x1000, xInputButtonA}, {0x2000, xInputButtonB},
		{0x4000, xInputButtonX}, {0x8000, xInputButtonY},
	}
	for _, item := range masks {
		if state.Buttons&item.mask != 0 {
			next[item.code] = true
		}
	}
	if t.down[xInputButtonLT] {
		if state.LeftTrigger >= xInputTriggerRelease {
			next[xInputButtonLT] = true
		}
	} else if state.LeftTrigger >= xInputTriggerPress {
		next[xInputButtonLT] = true
	}
	if t.down[xInputButtonRT] {
		if state.RightTrigger >= xInputTriggerRelease {
			next[xInputButtonRT] = true
		}
	} else if state.RightTrigger >= xInputTriggerPress {
		next[xInputButtonRT] = true
	}

	updateXInputAxisDirections(next, t.down, xInputLStickLeft, xInputLStickRight, normalizeXInputAxis(state.LeftX))
	updateXInputAxisDirections(next, t.down, xInputLStickDown, xInputLStickUp, normalizeXInputAxis(state.LeftY))
	updateXInputAxisDirections(next, t.down, xInputRStickLeft, xInputRStickRight, normalizeXInputAxis(state.RightX))
	updateXInputAxisDirections(next, t.down, xInputRStickDown, xInputRStickUp, normalizeXInputAxis(state.RightY))

	events := diffXInputControls(t.slot, t.down, next, at, false)
	t.down = next
	return events
}

func (t *xInputStateTracker) Disconnect(at time.Time) []InputEvent {
	events := diffXInputControls(t.slot, t.down, map[string]bool{}, at, true)
	t.down = make(map[string]bool, len(xInputKnownControls))
	return events
}

func diffXInputControls(slot int, previous, next map[string]bool, at time.Time, synthetic bool) []InputEvent {
	all := make(map[string]struct{}, len(previous)+len(next))
	for code := range previous {
		all[code] = struct{}{}
	}
	for code := range next {
		all[code] = struct{}{}
	}
	codes := make([]string, 0, len(all))
	for code := range all {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	events := make([]InputEvent, 0, len(codes))
	appendPhase := func(down bool) {
		for _, control := range codes {
			if previous[control] == next[control] || next[control] != down {
				continue
			}
			code := fmt.Sprintf("P%d:%s", slot+1, control)
			events = append(events, InputEvent{
				Token: InputToken{Kind: "XInput", Code: code, DeviceID: fmt.Sprintf("P%d", slot+1)},
				Down:  down, SourceID: fmt.Sprintf("xinput:P%d", slot+1), OccurredAt: at, Synthetic: synthetic,
			})
		}
	}
	appendPhase(false)
	appendPhase(true)
	return events
}
