package video

import (
	"testing"
	"time"
)

func TestWindowIdleFollowsDrawTicks(t *testing.T) {
	defer func() {
		drawTickLast.Store(0)
		drawTickFast.Store(0)
	}()
	drawTickLast.Store(0)
	drawTickFast.Store(0)

	now := time.Now()
	if windowIdle(now.Add(time.Hour)) {
		t.Fatal("idle before the game loop ever ticked")
	}
	NoteDrawTick()
	NoteDrawTick()
	if windowIdle(time.Now()) {
		t.Fatal("idle right after drawing")
	}
	if !windowIdle(time.Now().Add(time.Second)) {
		t.Fatal("not idle a second after the last fast tick")
	}

	// A slow tick, as a hidden window gets, does not count as drawing.
	drawTickLast.Store(time.Now().Add(-time.Second).UnixNano())
	drawTickFast.Store(time.Now().Add(-time.Second).UnixNano())
	NoteDrawTick()
	if !windowIdle(time.Now()) {
		t.Fatal("a slow tick made the window active")
	}
}

func TestContainsIDR(t *testing.T) {
	idr := []byte{0, 0, 0, 1, 0x67, 1, 2, 0, 0, 0, 1, 0x68, 3, 0, 0, 1, 0x65, 4}
	slice := []byte{0, 0, 0, 1, 0x41, 9}
	if !containsIDR(idr) {
		t.Fatal("missed the IDR slice")
	}
	if containsIDR(slice) {
		t.Fatal("took a P slice for an IDR")
	}
}
