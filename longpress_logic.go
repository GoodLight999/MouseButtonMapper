package main

import (
	"strings"
	"time"
)

const (
	longPressActionExecute = "Execute"
	longPressActionCancel  = "Cancel"
	defaultLongPressMs     = 500
	minLongPressMs         = 100
	maxLongPressMs         = 5000
)

func normalizeLongPressAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "cancel", "キャンセル":
		return longPressActionCancel
	default:
		return longPressActionExecute
	}
}

func normalizeLongPressMs(ms int) int {
	if ms == 0 {
		return defaultLongPressMs
	}
	if ms < minLongPressMs {
		return minLongPressMs
	}
	if ms > maxLongPressMs {
		return maxLongPressMs
	}
	return ms
}

func longPressReached(elapsed time.Duration, thresholdMs int) bool {
	return elapsed >= time.Duration(normalizeLongPressMs(thresholdMs))*time.Millisecond
}

func longPressActionText(action string) string {
	if normalizeLongPressAction(action) == longPressActionCancel {
		return "短押しをキャンセル"
	}
	return "別の操作を実行"
}
