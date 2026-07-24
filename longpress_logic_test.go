package main

import (
	"testing"
	"time"
)

func TestNormalizeLongPressMs(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{0, 500},
		{1, 100},
		{100, 100},
		{750, 750},
		{9000, 5000},
	}
	for _, tt := range tests {
		if got := normalizeLongPressMs(tt.in); got != tt.want {
			t.Fatalf("normalizeLongPressMs(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestLongPressReachedBoundary(t *testing.T) {
	if longPressReached(499*time.Millisecond, 500) {
		t.Fatal("499ms must remain a short press for a 500ms threshold")
	}
	if !longPressReached(500*time.Millisecond, 500) {
		t.Fatal("the exact threshold must count as a long press")
	}
}

func TestNormalizeLongPressAction(t *testing.T) {
	if got := normalizeLongPressAction("cancel"); got != longPressActionCancel {
		t.Fatalf("cancel normalized to %q", got)
	}
	if got := normalizeLongPressAction(""); got != longPressActionExecute {
		t.Fatalf("empty action normalized to %q", got)
	}
}
